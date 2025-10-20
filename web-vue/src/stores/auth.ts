import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { apiService } from '@/services/api'
import type { User, LoginRequest, RegisterRequest } from '@/types'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<User | null>(null)
  const isAuthenticated = computed(() => !!user.value)
  const isLoading = ref(false)

  async function checkAuth() {
    isLoading.value = true
    try {
      const result = await apiService.checkAuth()
      if (result.isAuthenticated && result.user) {
        user.value = result.user
      } else {
        user.value = null
      }
      return result.isAuthenticated
    } finally {
      isLoading.value = false
    }
  }

  async function login(credentials: LoginRequest) {
    isLoading.value = true
    try {
      const success = await apiService.login(credentials)
      if (success) {
        await checkAuth()
      }
      return success
    } finally {
      isLoading.value = false
    }
  }

  async function register(credentials: RegisterRequest) {
    isLoading.value = true
    try {
      return await apiService.register(credentials)
    } finally {
      isLoading.value = false
    }
  }

  function logout() {
    apiService.logout()
    user.value = null
  }

  return {
    user,
    isAuthenticated,
    isLoading,
    checkAuth,
    login,
    register,
    logout
  }
})

