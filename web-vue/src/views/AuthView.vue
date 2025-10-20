<template>
  <!-- Auth Screen -->
  <div class="flex items-center justify-center h-screen bg-gray-900">
    <div class="bg-gray-800 p-6 rounded-lg shadow-lg w-full max-w-sm">
      <!-- Login Form -->
      <div v-if="showLogin">
        <div class="text-center mb-6">
          <div class="w-12 h-12 bg-indigo-600 rounded-full flex items-center justify-center mx-auto mb-3">
            <i class="pi pi-comments text-white"></i>
          </div>
          <h2 class="text-xl font-semibold text-gray-200">Welcome back</h2>
          <p class="text-sm text-gray-400 mt-1">Sign in to your account</p>
        </div>
        <form @submit.prevent="handleLogin" class="space-y-4">
          <div>
            <input
              type="text"
              id="login-username"
              v-model="loginForm.username"
              placeholder="Username"
              :disabled="isLoading"
              class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded text-sm text-white placeholder-gray-400 focus:outline-none focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500"
            />
          </div>
          <div>
            <input
              type="password"
              id="login-password"
              v-model="loginForm.password"
              placeholder="Password"
              :disabled="isLoading"
              class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded text-sm text-white placeholder-gray-400 focus:outline-none focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500"
            />
          </div>
          <button
            type="submit"
            :disabled="isLoading"
            class="w-full bg-indigo-600 hover:bg-indigo-700 text-white font-medium py-2 px-4 rounded text-sm transition-colors disabled:opacity-50"
          >
            {{ isLoading ? 'Loading...' : 'Sign In' }}
          </button>
        </form>
        <p class="text-center text-xs text-gray-400 mt-4">
          Don't have an account?
          <button @click="showLogin = false" class="text-indigo-400 hover:underline">Sign up</button>
        </p>
      </div>

      <!-- Registration Form -->
      <div v-else>
        <div class="text-center mb-6">
          <div class="w-12 h-12 bg-green-600 rounded-full flex items-center justify-center mx-auto mb-3">
            <i class="pi pi-user-plus text-white"></i>
          </div>
          <h2 class="text-xl font-semibold text-gray-200">Create account</h2>
          <p class="text-sm text-gray-400 mt-1">Join RoomSpeak today</p>
        </div>
        <form @submit.prevent="handleRegister" class="space-y-4">
          <div>
            <input
              type="text"
              id="register-username"
              v-model="registerForm.username"
              placeholder="Username"
              :disabled="isLoading"
              class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded text-sm text-white placeholder-gray-400 focus:outline-none focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500"
            />
          </div>
          <div>
            <input
              type="password"
              id="register-password"
              v-model="registerForm.password"
              placeholder="Password"
              :disabled="isLoading"
              class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded text-sm text-white placeholder-gray-400 focus:outline-none focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500"
            />
          </div>
          <button
            type="submit"
            :disabled="isLoading"
            class="w-full bg-green-600 hover:bg-green-700 text-white font-medium py-2 px-4 rounded text-sm transition-colors disabled:opacity-50"
          >
            {{ isLoading ? 'Loading...' : 'Create Account' }}
          </button>
        </form>
        <p class="text-center text-xs text-gray-400 mt-4">
          Already have an account?
          <button @click="showLogin = true" class="text-indigo-400 hover:underline">Sign in</button>
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useToast } from 'primevue/usetoast'

const router = useRouter()
const authStore = useAuthStore()
const toast = useToast()

const showLogin = ref(true)
const loginForm = ref({
  username: '',
  password: ''
})
const registerForm = ref({
  username: '',
  password: ''
})

const isLoading = computed(() => authStore.isLoading)

async function handleLogin() {
  if (!loginForm.value.username || !loginForm.value.password) {
    toast.add({
      severity: 'warn',
      summary: 'Warning',
      detail: 'Please fill in all fields',
      life: 3000
    })
    return
  }

  const success = await authStore.login(loginForm.value)
  if (success) {
    toast.add({
      severity: 'success',
      summary: 'Success',
      detail: 'Successfully logged in',
      life: 3000
    })
    router.push({ name: 'home' })
  } else {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: 'Invalid username or password',
      life: 3000
    })
  }
}

async function handleRegister() {
  if (!registerForm.value.username || !registerForm.value.password) {
    toast.add({
      severity: 'warn',
      summary: 'Warning',
      detail: 'Please fill in all fields',
      life: 3000
    })
    return
  }

  const success = await authStore.register(registerForm.value)
  if (success) {
    toast.add({
      severity: 'success',
      summary: 'Success',
      detail: 'Registration successful. Please login.',
      life: 3000
    })
    showLogin.value = true
    registerForm.value = { username: '', password: '' }
  } else {
    toast.add({
      severity: 'error',
      summary: 'Error',
      detail: 'Registration failed. Try a different username.',
      life: 3000
    })
  }
}
</script>
