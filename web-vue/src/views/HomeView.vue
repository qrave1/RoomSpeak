<template>
  <div class="flex h-screen text-white">
    <!-- Sidebar -->
    <div class="w-60 bg-gray-800 flex flex-col">
      <!-- Header -->
      <div class="flex items-center justify-between px-4 py-3 border-b border-gray-700">
        <h3 class="text-sm font-semibold text-gray-200 uppercase tracking-wide">RoomSpeak</h3>
      </div>

      <!-- Content -->
      <div class="flex-1 flex items-center justify-center p-4">
        <div class="text-center">
          <div class="w-16 h-16 bg-indigo-600 rounded-full flex items-center justify-center mx-auto mb-4">
            <i class="pi pi-comments text-2xl text-white"></i>
          </div>
          <p class="text-sm text-gray-400">Welcome to</p>
          <h2 class="text-lg font-semibold text-gray-200">RoomSpeak</h2>
        </div>
      </div>

      <!-- User Info -->
      <div class="px-4 py-3 border-t border-gray-700">
        <div class="flex items-center space-x-2">
          <div class="w-8 h-8 bg-gray-600 rounded-full flex items-center justify-center">
            <i class="pi pi-user text-xs text-gray-300"></i>
          </div>
          <span class="text-sm text-gray-300 truncate">{{ authStore.user?.username }}</span>
        </div>
      </div>

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
    <div class="flex-grow bg-gray-900 flex items-center justify-center">
      <div class="text-center">
        <div class="w-16 h-16 bg-gray-700 rounded-full flex items-center justify-center mx-auto mb-4">
          <i class="pi pi-comments text-2xl text-gray-400"></i>
        </div>
        <h1 class="text-xl font-semibold text-gray-200 mb-2">Hello World!</h1>
        <p class="text-gray-400 text-sm">Welcome, {{ authStore.user?.username }}!</p>
        <p class="text-gray-500 text-xs mt-2">You are successfully authenticated.</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useToast } from 'primevue/usetoast'

const router = useRouter()
const authStore = useAuthStore()
const toast = useToast()

function handleLogout() {
  authStore.logout()
  toast.add({
    severity: 'info',
    summary: 'Logged out',
    detail: 'You have been logged out',
    life: 3000
  })
  router.push({ name: 'auth' })
}
</script>
