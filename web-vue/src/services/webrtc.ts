import type { ICEServerResponse } from '@/types'
import { wsService } from './websocket'

class WebRTCService {
  private peerConnection: RTCPeerConnection | null = null
  private localStream: MediaStream | null = null
  private remoteAudioElements: HTMLAudioElement[] = []

  async initialize(inputDeviceId?: string, outputDeviceId?: string): Promise<{
    pc: RTCPeerConnection
    localStream: MediaStream
  }> {
    // Получаем ICE серверы
    const iceResponse = await fetch('/api/v1/ice')
    const iceData: ICEServerResponse = await iceResponse.json()

    const iceServers: RTCIceServer[] = [
      { urls: 'stun:stun.l.google.com:19302' },
      {
        urls: iceData.urls,
        username: iceData.username,
        credential: iceData.credential
      }
    ]

    // Создаем peer connection
    this.peerConnection = new RTCPeerConnection({ iceServers })

    // Обработчик ICE кандидатов
    this.peerConnection.onicecandidate = (event) => {
      if (event.candidate) {
        wsService.send({
          type: 'candidate',
          data: {
            candidate: event.candidate
          }
        })
      }
    }

    // Получаем локальный поток
    const constraints: MediaStreamConstraints = {
      audio: inputDeviceId ? { deviceId: { exact: inputDeviceId } } : true
    }

    this.localStream = await navigator.mediaDevices.getUserMedia(constraints)

    // Добавляем локальные треки
    this.localStream.getTracks().forEach(track => {
      this.peerConnection!.addTrack(track, this.localStream!)
    })

    // Обработчик удаленных треков
    this.peerConnection.ontrack = (event) => {
      if (event.track.kind === 'audio') {
        const audio = new Audio()
        audio.srcObject = event.streams[0]
        audio.autoplay = true
        document.body.appendChild(audio)

        // Устанавливаем устройство вывода
        if (outputDeviceId && 'setSinkId' in audio) {
          (audio as any).setSinkId(outputDeviceId)
            .catch((err: Error) => console.error('Error setting audio output:', err))
        }

        this.remoteAudioElements.push(audio)

        audio.play().catch(err => console.error('Failed to play remote audio:', err))
      }
    }

    return {
      pc: this.peerConnection,
      localStream: this.localStream
    }
  }

  async createOffer(): Promise<void> {
    if (!this.peerConnection) {
      throw new Error('Peer connection not initialized')
    }

    const offer = await this.peerConnection.createOffer()
    await this.peerConnection.setLocalDescription(offer)

    wsService.send({
      type: 'offer',
      data: {
        sdp: offer.sdp
      }
    })
  }

  async handleOffer(sdp: string): Promise<void> {
    if (!this.peerConnection) {
      throw new Error('Peer connection not initialized')
    }

    await this.peerConnection.setRemoteDescription({
      type: 'offer',
      sdp
    })

    const answer = await this.peerConnection.createAnswer()
    await this.peerConnection.setLocalDescription(answer)

    wsService.send({
      type: 'offer',
      data: {
        sdp: answer.sdp
      }
    })
  }

  async handleAnswer(sdp: string): Promise<void> {
    if (!this.peerConnection) {
      throw new Error('Peer connection not initialized')
    }

    await this.peerConnection.setRemoteDescription({
      type: 'answer',
      sdp
    })
  }

  async addIceCandidate(candidate: RTCIceCandidateInit): Promise<void> {
    if (!this.peerConnection) {
      throw new Error('Peer connection not initialized')
    }

    try {
      await this.peerConnection.addIceCandidate(candidate)
    } catch (err) {
      console.error('Error adding ICE candidate:', err)
    }
  }

  toggleMute(isMuted: boolean): void {
    if (this.localStream) {
      this.localStream.getAudioTracks().forEach(track => {
        track.enabled = !isMuted
      })
    }
  }

  async updateInputDevice(deviceId: string): Promise<void> {
    if (!this.localStream || !this.peerConnection) {
      return
    }

    // Останавливаем текущий поток
    this.localStream.getTracks().forEach(track => track.stop())

    // Получаем новый поток
    const newStream = await navigator.mediaDevices.getUserMedia({
      audio: { deviceId: { exact: deviceId } }
    })

    // Заменяем трек в peer connection
    const sender = this.peerConnection.getSenders().find(s => s.track?.kind === 'audio')
    if (sender) {
      await sender.replaceTrack(newStream.getAudioTracks()[0])
    }

    this.localStream = newStream
  }

  updateOutputDevice(deviceId: string): void {
    this.remoteAudioElements.forEach(audio => {
      if ('setSinkId' in audio) {
        (audio as any).setSinkId(deviceId)
          .catch((err: Error) => console.error('Error setting audio output:', err))
      }
    })
  }

  disconnect(): void {
    // Останавливаем локальный поток
    if (this.localStream) {
      this.localStream.getTracks().forEach(track => track.stop())
      this.localStream = null
    }

    // Удаляем удаленные аудио элементы
    this.remoteAudioElements.forEach(audio => {
      audio.pause()
      audio.srcObject = null
      audio.remove()
    })
    this.remoteAudioElements = []

    // Закрываем peer connection
    if (this.peerConnection) {
      this.peerConnection.close()
      this.peerConnection = null
    }
  }

  getPeerConnection(): RTCPeerConnection | null {
    return this.peerConnection
  }

  getLocalStream(): MediaStream | null {
    return this.localStream
  }
}

export const webrtcService = new WebRTCService()



