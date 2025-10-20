<template>
  <div class="flex flex-col h-full">
    <!-- Header -->
    <div class="flex items-center justify-between px-4 py-3 border-b border-gray-700">
      <h3 class="text-sm font-semibold text-gray-200 uppercase tracking-wide">Channels</h3>
      <button
        @click="refreshChannels"
        class="text-gray-400 hover:text-white transition-colors p-1"
        title="Refresh channels"
      >
        <i class="pi pi-refresh text-xs"></i>
      </button>
    </div>

    <!-- Channels List -->
    <div class="flex-1 overflow-y-auto px-2 py-2">
      <ul class="space-y-1">
        <li v-for="channel in channels" :key="channel.id" class="group">
          <div class="flex items-center justify-between px-2 py-1.5 rounded hover:bg-gray-700 transition-colors">
            <div class="flex items-center flex-1 min-w-0">
              <div class="flex items-center space-x-2 flex-1 min-w-0">
                <span class="text-sm text-gray-300 truncate">{{ channel.name }}</span>
                <span
                  v-if="channel.active_users && channel.active_users.length > 0"
                  class="text-xs text-gray-500 bg-gray-600 px-1.5 py-0.5 rounded-full"
                >
                  {{ channel.active_users.length }}
                </span>
              </div>
            </div>
            <div class="flex items-center space-x-1 opacity-0 group-hover:opacity-100 transition-opacity">
              <button
                @click="$emit('join-channel', channel)"
                class="text-gray-400 hover:text-white p-1 rounded hover:bg-gray-600 transition-colors"
                title="Join channel"
              >
                <i class="pi pi-sign-in text-xs"></i>
              </button>
              <button
                @click="$emit('delete-channel', channel)"
                class="text-gray-400 hover:text-red-400 p-1 rounded hover:bg-gray-600 transition-colors"
                title="Delete channel"
              >
                <i class="pi pi-trash text-xs"></i>
              </button>
            </div>
          </div>

          <!-- Active Members -->
          <div
            v-if="channel.active_users && channel.active_users.length > 0"
            class="px-4 pb-1"
          >
            <div class="flex items-center space-x-2">
              <template v-for="member in channel.active_users.slice(0, 3)" :key="member.id">
                <div class="flex items-center space-x-1">
                  <div class="w-1.5 h-1.5 bg-green-400 rounded-full"></div>
                  <span class="text-xs text-gray-500">{{ member.username }}</span>
                </div>
              </template>
              <span
                v-if="channel.active_users.length > 3"
                class="text-xs text-gray-500"
              >
                +{{ channel.active_users.length - 3 }}
              </span>
            </div>
          </div>
        </li>
      </ul>

      <!-- Empty state -->
      <div v-if="channels.length === 0 && !isLoading" class="text-center py-8">
        <i class="pi pi-inbox text-2xl text-gray-600 mb-2"></i>
        <p class="text-sm text-gray-500">No channels yet</p>
      </div>

      <!-- Loading state -->
      <div v-if="isLoading" class="text-center py-8">
        <i class="pi pi-spin pi-spinner text-2xl text-gray-600"></i>
      </div>
    </div>

    <!-- Create Channel -->
    <div class="px-4 py-3 border-t border-gray-700">
      <div class="space-y-2">
        <div class="relative">
          <input
            type="text"
            v-model="newChannelName"
            placeholder="Channel Name"
            class="w-full px-3 py-1.5 bg-gray-700 border border-gray-600 rounded text-sm text-white placeholder-gray-400 focus:outline-none focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500"
          />
        </div>
        <div class="flex items-center justify-between">
          <label class="flex items-center">
            <input
              type="checkbox"
              v-model="newChannelIsPublic"
              class="mr-2 bg-gray-700 border-gray-600 rounded text-indigo-600 focus:ring-indigo-500 focus:ring-1"
            />
            <span class="text-xs text-gray-400">Public</span>
          </label>
          <button
            @click="createChannel"
            :disabled="!newChannelName.trim()"
            class="px-3 py-1 bg-green-600 hover:bg-green-700 text-white text-xs font-medium rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Create
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useChannelsStore } from '@/stores/channels'
import type { Channel } from '@/types'

defineEmits<{
  (e: 'join-channel', channel: Channel): void
  (e: 'delete-channel', channel: Channel): void
}>()

const channelsStore = useChannelsStore()

const newChannelName = ref('')
const newChannelIsPublic = ref(false)

const channels = computed(() => channelsStore.channels)
const isLoading = computed(() => channelsStore.isLoading)

async function refreshChannels() {
  await channelsStore.fetchChannels()
}

async function createChannel() {
  if (!newChannelName.value.trim()) return

  const success = await channelsStore.createChannel({
    name: newChannelName.value,
    is_public: newChannelIsPublic.value
  })

  if (success) {
    newChannelName.value = ''
    newChannelIsPublic.value = false
  }
}
</script>




