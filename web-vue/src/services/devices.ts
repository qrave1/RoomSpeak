import type { AudioDevice } from '@/types'

class DevicesService {
  async requestPermissions(): Promise<void> {
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
      stream.getTracks().forEach(track => track.stop())
    } catch (err) {
      console.error('Error getting permissions:', err)
      throw err
    }
  }

  async getAudioDevices(): Promise<{
    audioInputDevices: AudioDevice[]
    audioOutputDevices: AudioDevice[]
  }> {
    await this.requestPermissions()

    const devices = await navigator.mediaDevices.enumerateDevices()
    const audioInputDevices = devices
      .filter(d => d.kind === 'audioinput')
      .map(d => ({
        deviceId: d.deviceId,
        label: d.label || `Microphone ${d.deviceId.substring(0, 5)}`,
        kind: d.kind
      }))

    const audioOutputDevices = devices
      .filter(d => d.kind === 'audiooutput')
      .map(d => ({
        deviceId: d.deviceId,
        label: d.label || `Speaker ${d.deviceId.substring(0, 5)}`,
        kind: d.kind
      }))

    return { audioInputDevices, audioOutputDevices }
  }

  async getUserMedia(deviceId?: string): Promise<MediaStream> {
    const constraints: MediaStreamConstraints = {
      audio: deviceId ? { deviceId: { exact: deviceId } } : true
    }

    return await navigator.mediaDevices.getUserMedia(constraints)
  }

  async setOutputDevice(audioElement: HTMLAudioElement, deviceId: string): Promise<void> {
    if ('setSinkId' in audioElement) {
      try {
        await (audioElement as any).setSinkId(deviceId)
      } catch (err) {
        console.error('Error setting audio output:', err)
        throw err
      }
    }
  }

  stopStream(stream: MediaStream | null) {
    if (stream) {
      stream.getTracks().forEach(track => track.stop())
    }
  }
}

export const devicesService = new DevicesService()



