import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiService } from '@/services/api'
import type { Channel, CreateChannelRequest } from '@/types'

export const useChannelsStore = defineStore('channels', () => {
  const channels = ref<Channel[]>([])
  const isLoading = ref(false)
  const error = ref<string | null>(null)

  async function fetchChannels() {
    isLoading.value = true
    error.value = null
    try {
      channels.value = await apiService.getChannels()
    } catch (err) {
      error.value = 'Failed to fetch channels'
      console.error('Error fetching channels:', err)
    } finally {
      isLoading.value = false
    }
  }

  async function createChannel(data: CreateChannelRequest): Promise<boolean> {
    isLoading.value = true
    error.value = null
    try {
      const success = await apiService.createChannel(data)
      if (success) {
        await fetchChannels()
      }
      return success
    } catch (err) {
      error.value = 'Failed to create channel'
      console.error('Error creating channel:', err)
      return false
    } finally {
      isLoading.value = false
    }
  }

  async function deleteChannel(channelId: string): Promise<boolean> {
    isLoading.value = true
    error.value = null
    try {
      const success = await apiService.deleteChannel(channelId)
      if (success) {
        await fetchChannels()
      }
      return success
    } catch (err) {
      error.value = 'Failed to delete channel'
      console.error('Error deleting channel:', err)
      return false
    } finally {
      isLoading.value = false
    }
  }

  function getChannelById(id: string): Channel | undefined {
    return channels.value.find(c => c.id === id)
  }

  return {
    channels,
    isLoading,
    error,
    fetchChannels,
    createChannel,
    deleteChannel,
    getChannelById
  }
})





