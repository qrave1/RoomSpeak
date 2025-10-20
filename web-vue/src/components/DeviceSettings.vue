<template>
  <div class="px-4 py-3 border-t border-gray-700">
    <div class="space-y-2">
      <!-- Audio Input -->
      <div>
        <label for="audio-input" class="block text-xs font-medium text-gray-400 mb-1">
          Audio Input
        </label>
        <select
          id="audio-input"
          v-model="selectedInput"
          @change="handleInputChange"
          class="w-full px-2 py-1 bg-gray-700 border border-gray-600 rounded text-xs text-white focus:outline-none focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500"
        >
          <option v-for="device in audioInputDevices" :key="device.deviceId" :value="device.deviceId">
            {{ device.label }}
          </option>
        </select>
      </div>

      <!-- Audio Output -->
      <div>
        <label for="audio-output" class="block text-xs font-medium text-gray-400 mb-1">
          Audio Output
        </label>
        <select
          id="audio-output"
          v-model="selectedOutput"
          @change="handleOutputChange"
          class="w-full px-2 py-1 bg-gray-700 border border-gray-600 rounded text-xs text-white focus:outline-none focus:ring-1 focus:ring-indigo-500 focus:border-indigo-500"
        >
          <option v-for="device in audioOutputDevices" :key="device.deviceId" :value="device.deviceId">
            {{ device.label }}
          </option>
        </select>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useVoiceStore } from '@/stores/voice'

const voiceStore = useVoiceStore()

const audioInputDevices = computed(() => voiceStore.audioInputDevices)
const audioOutputDevices = computed(() => voiceStore.audioOutputDevices)

const selectedInput = computed({
  get: () => voiceStore.selectedInputDevice,
  set: (value) => voiceStore.updateInputDevice(value)
})

const selectedOutput = computed({
  get: () => voiceStore.selectedOutputDevice,
  set: (value) => voiceStore.updateOutputDevice(value)
})

function handleInputChange() {
  // The computed setter already handles the update
}

function handleOutputChange() {
  // The computed setter already handles the update
}
</script>




