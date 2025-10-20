<template>
  <div class="w-60 bg-gray-800 border-l border-gray-700 flex flex-col">
    <!-- Header -->
    <div class="px-4 py-3 border-b border-gray-700">
      <h3 class="text-sm font-semibold text-gray-200 uppercase tracking-wide">
        Online — {{ onlineUsers.length }}
      </h3>
    </div>

    <!-- Online Users List -->
    <div class="flex-1 overflow-y-auto px-2 py-2">
      <template v-if="onlineUsers.length > 0">
        <div
            v-for="user in onlineUsers"
            :key="user.id"
            class="px-2 py-1.5 rounded hover:bg-gray-700 transition-colors group mb-1"
        >
          <!-- User info -->
          <div class="flex items-center space-x-3">
            <!-- Avatar -->
            <div class="relative">
              <div
                  class="w-8 h-8 rounded-full flex items-center justify-center"
                  :class="`avatar-gradient-${(user.username.charCodeAt(0) % 8) + 1}`"
              >
                <span class="text-xs font-medium text-white">
                  {{ user.username.charAt(0).toUpperCase() }}
                </span>
              </div>
              <!-- Online Status -->
              <div
                  class="absolute -bottom-0.5 -right-0.5 w-3 h-3 bg-green-500 border-2 border-gray-800 rounded-full"></div>
            </div>

            <!-- Username -->
            <div class="flex-1 min-w-0">
              <span class="text-sm text-gray-300 truncate block">{{ user.username }}</span>
              <!-- Channel info -->
              <div v-if="user.channel_name" class="flex items-center space-x-1 text-xs text-gray-500">
                <i class="pi pi-volume-up" style="font-size: 0.7rem"></i>
                <span class="truncate">{{ user.channel_name }}</span>
              </div>
            </div>
          </div>
        </div>
      </template>

      <!-- Empty State -->
      <div v-else class="text-center py-8">
        <div class="w-12 h-12 bg-gray-700 rounded-full flex items-center justify-center mx-auto mb-3">
          <i class="pi pi-users text-gray-400"></i>
        </div>
        <p class="text-sm text-gray-400">No users online</p>
      </div>
    </div>

    <!-- Footer with total count -->
    <div class="px-4 py-2 border-t border-gray-700">
      <div class="text-xs text-gray-500 text-center">
        {{ onlineUsers.length }} user{{ onlineUsers.length !== 1 ? 's' : '' }} online
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import {onMounted, onUnmounted, ref} from 'vue'
import {apiService} from '@/services/api'
import type {OnlineUser} from '@/types'

const onlineUsers = ref<OnlineUser[]>([])

// Fetch online users
async function fetchOnlineUsers() {
  onlineUsers.value = await apiService.getOnlineUsers()
}

// Auto-refresh every 3 seconds
let refreshInterval: number | null = null

onMounted(async () => {
  // Initial fetch
  await fetchOnlineUsers()

  // Start auto-refresh
  refreshInterval = window.setInterval(fetchOnlineUsers, 3000)
})

onUnmounted(() => {
  if (refreshInterval !== null) {
    clearInterval(refreshInterval)
  }
})
</script>
