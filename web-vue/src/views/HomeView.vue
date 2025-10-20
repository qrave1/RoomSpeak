<template>
  <div class="flex h-screen text-white">
    <!-- Sidebar -->
    <div class="w-60 bg-gray-800 flex flex-col">
      <!-- Channels List -->
      <ChannelsList
        @join-channel="handleJoinChannel"
        @delete-channel="handleDeleteChannel"
      />

      <!-- User Info -->
      <div class="px-4 py-3 border-t border-gray-700">
        <div class="flex items-center space-x-2">
          <div class="w-8 h-8 bg-gray-600 rounded-full flex items-center justify-center">
            <i class="pi pi-user text-xs text-gray-300"></i>
          </div>
          <span class="text-sm text-gray-300 truncate">{{ authStore.user?.username }}</span>
        </div>
      </div>

      <!-- Device Settings -->
      <DeviceSettings />

      <!-- Logout -->
      <div class="px-4 py-3 border-t border-gray-700">
        <button
          @click="handleLogout"
          class="w-full px-3 py-1.5 bg-red-600 hover:bg-red-700 text-white text-xs font-medium rounded transition-colors"
        >
          Logout
        </button>
      </div>
    </div>

    <!-- Main Content -->
    <div class="flex-grow bg-gray-900 flex">
      <!-- Chat Area -->
      <div class="flex-grow flex items-center justify-center">
        <!-- Not connected -->
        <div v-if="!voiceStore.isConnected" class="text-center">
          <div class="w-16 h-16 bg-gray-700 rounded-full flex items-center justify-center mx-auto mb-4">
            <i class="pi pi-comments text-2xl text-gray-400"></i>
          </div>
          <h1 class="text-xl font-semibold text-gray-200 mb-2">Welcome to RoomSpeak</h1>
          <p class="text-gray-400 text-sm">Select a channel to join or create a new one.</p>
        </div>

        <!-- Connected to channel -->
        <div v-else class="w-full max-w-md">
          <div class="bg-gray-800 rounded-lg p-6">
            <div class="flex items-center justify-between mb-4">
              <h3 class="text-lg font-semibold text-gray-200">
                {{ voiceStore.currentChannelName }}
              </h3>
              <button
                @click="handleDisconnect"
                class="text-gray-400 hover:text-red-400 transition-colors"
                title="Disconnect"
              >
                <i class="pi pi-phone text-lg"></i>
              </button>
            </div>

            <!-- Mute Button -->
            <button
              @click="voiceStore.toggleMute()"
              class="w-full bg-gray-700 hover:bg-gray-600 text-white font-medium py-2 px-4 rounded transition-colors mb-4"
              :class="{ 'ring-2 ring-green-400': voiceStore.isSpeaking }"
            >
              <i
                class="pi mr-2"
                :class="voiceStore.isMuted ? 'pi-microphone-slash' : 'pi-microphone'"
              ></i>
              <span>{{ voiceStore.isMuted ? 'Unmute' : 'Mute' }}</span>
              <span v-if="voiceStore.isSpeaking" class="ml-2 text-xs text-green-400">Speaking</span>
            </button>
          </div>
        </div>
      </div>

      <!-- Right Sidebar: Members in Channel or Online Users -->
      <div v-if="voiceStore.isConnected" class="w-60 bg-gray-800 border-l border-gray-700 flex flex-col">
        <!-- Members Header -->
        <div class="px-4 py-3 border-b border-gray-700">
          <h3 class="text-sm font-semibold text-gray-200 uppercase tracking-wide">
            Members — {{ voiceStore.participants.length }}
          </h3>
        </div>

        <!-- Members List -->
        <div class="flex-1 overflow-y-auto px-2 py-2">
          <template v-if="voiceStore.participants.length > 0">
            <div
              v-for="participant in voiceStore.participants"
              :key="participant.id"
              class="flex items-center space-x-3 px-2 py-1.5 rounded hover:bg-gray-700 transition-colors group"
            >
              <!-- Avatar -->
              <div class="relative">
                <div
                  class="w-8 h-8 rounded-full flex items-center justify-center"
                  :class="`avatar-gradient-${(participant.username.charCodeAt(0) % 8) + 1}`"
                >
                  <span class="text-xs font-medium text-white">
                    {{ participant.username.charAt(0).toUpperCase() }}
                  </span>
                </div>
                <!-- Online Status -->
                <div class="absolute -bottom-0.5 -right-0.5 w-3 h-3 bg-green-500 border-2 border-gray-800 rounded-full"></div>
              </div>

              <!-- Username -->
              <div class="flex-1 min-w-0">
                <div class="flex items-center space-x-2">
                  <span class="text-sm text-gray-300 truncate">{{ participant.username }}</span>
                  <!-- Mute Status -->
                  <i
                    v-if="participant.is_muted"
                    class="pi pi-microphone-slash text-xs text-red-500"
                  ></i>
                </div>
              </div>

              <!-- Speaking Indicator -->
              <div
                v-if="participant.username === authStore.user?.username && voiceStore.isSpeaking"
                class="w-2 h-2 bg-green-400 rounded-full animate-pulse"
              ></div>
            </div>
          </template>

          <!-- Empty State -->
          <div v-else class="text-center py-8">
            <div class="w-12 h-12 bg-gray-700 rounded-full flex items-center justify-center mx-auto mb-3">
              <i class="pi pi-users text-gray-400"></i>
            </div>
            <p class="text-sm text-gray-400">No members online</p>
          </div>
        </div>
      </div>

      <!-- Online Users Sidebar (when not in channel) -->
      <OnlineUsers v-else />
    </div>

    <!-- Delete Modal -->
    <div
      v-if="showDeleteModal"
      class="fixed inset-0 bg-gray-900 bg-opacity-50 flex items-center justify-center"
      @click="showDeleteModal = false"
    >
      <div class="bg-gray-800 rounded-lg shadow-lg p-6 w-full max-w-sm" @click.stop>
        <h3 class="text-lg font-semibold text-gray-200 mb-3">Delete Channel</h3>
        <p class="text-gray-300 text-sm mb-6">
          Are you sure you want to delete the channel "<span class="font-medium">{{ channelToDelete?.name }}</span>"?
        </p>
        <div class="flex justify-end space-x-3">
          <button
            @click="showDeleteModal = false"
            class="px-4 py-2 bg-gray-700 hover:bg-gray-600 text-white text-sm font-medium rounded transition-colors"
          >
            Cancel
          </button>
          <button
            @click="confirmDelete"
            class="px-4 py-2 bg-red-600 hover:bg-red-700 text-white text-sm font-medium rounded transition-colors"
          >
            Delete
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useChannelsStore } from '@/stores/channels'
import { useVoiceStore } from '@/stores/voice'
import { useToast } from 'primevue/usetoast'
import ChannelsList from '@/components/ChannelsList.vue'
import DeviceSettings from '@/components/DeviceSettings.vue'
import OnlineUsers from '@/components/OnlineUsers.vue'
import type { Channel } from '@/types'

const router = useRouter()
const authStore = useAuthStore()
const channelsStore = useChannelsStore()
const voiceStore = useVoiceStore()
const toast = useToast()

const showDeleteModal = ref(false)
const channelToDelete = ref<Channel | null>(null)

// Initialize
onMounted(async () => {
  // Connect WebSocket
  voiceStore.connectWebSocket()

  // Fetch channels
  await channelsStore.fetchChannels()

  // Initialize audio devices
  await voiceStore.initializeDevices()

  // Start auto-refresh channels
  const refreshInterval = setInterval(() => {
    channelsStore.fetchChannels()
  }, 3000)

  // Cleanup on unmount
  onUnmounted(() => {
    clearInterval(refreshInterval)
  })
})

// Handle join channel
async function handleJoinChannel(channel: Channel) {
  try {
    await voiceStore.joinChannel(channel.id, channel.name)
    toast.add({
      severity: 'success',
      summary: 'Connected',
      detail: `Joined ${channel.name}`,
      life: 3000
    })
  } catch (err) {
    console.error('Error joining channel:', err)
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: 'Failed to join channel',
      life: 3000
    })
  }
}

// Handle delete channel
function handleDeleteChannel(channel: Channel) {
  channelToDelete.value = channel
  showDeleteModal.value = true
}

// Confirm delete
async function confirmDelete() {
  if (!channelToDelete.value) return

  const success = await channelsStore.deleteChannel(channelToDelete.value.id)
  if (success) {
    toast.add({
      severity: 'success',
      summary: 'Deleted',
      detail: 'Channel deleted successfully',
      life: 3000
    })
  } else {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: 'Failed to delete channel',
      life: 3000
    })
  }

  showDeleteModal.value = false
  channelToDelete.value = null
}

// Handle disconnect
function handleDisconnect() {
  voiceStore.disconnect()
  toast.add({
    severity: 'info',
    summary: 'Disconnected',
    detail: 'Left the channel',
    life: 3000
  })
}

// Handle logout
function handleLogout() {
  voiceStore.cleanup()
  authStore.logout()
  toast.add({
    severity: 'info',
    summary: 'Logged out',
    detail: 'You have been logged out',
    life: 3000
  })
  router.push({ name: 'auth' })
}

// Cleanup on unmount
onUnmounted(() => {
  voiceStore.cleanup()
})
</script>
