import {defineStore} from 'pinia'
import {computed, ref} from 'vue'
import {wsService} from '@/services/websocket'
import {webrtcService} from '@/services/webrtc'
import {devicesService} from '@/services/devices'
import type {AudioDevice, Participant, WSMessage} from '@/types'
import {useAuthStore} from './auth'

export const useVoiceStore = defineStore('voice', () => {
    const authStore = useAuthStore()

    // State
    const isConnected = ref(false)
    const currentChannelId = ref<string | null>(null)
    const currentChannelName = ref<string | null>(null)
    const participants = ref<Participant[]>([])
    const isMuted = ref(false)
    const isSpeaking = ref(false)

    // Devices
    const audioInputDevices = ref<AudioDevice[]>([])
    const audioOutputDevices = ref<AudioDevice[]>([])
    const selectedInputDevice = ref<string>('')
    const selectedOutputDevice = ref<string>('')

    // Computed
    const currentParticipant = computed(() => {
        return participants.value.find(p => p.username === authStore.user?.username)
    })

    // Initialize devices
    async function initializeDevices() {
        try {
            const {audioInputDevices: inputs, audioOutputDevices: outputs} =
                await devicesService.getAudioDevices()

            audioInputDevices.value = inputs
            audioOutputDevices.value = outputs

            if (inputs.length > 0 && inputs[0]) {
                selectedInputDevice.value = inputs[0].deviceId
            }
            if (outputs.length > 0 && outputs[0]) {
                selectedOutputDevice.value = outputs[0].deviceId
            }

            // Listen for device changes
            navigator.mediaDevices.addEventListener('devicechange', async () => {
                const {audioInputDevices: newInputs, audioOutputDevices: newOutputs} =
                    await devicesService.getAudioDevices()
                audioInputDevices.value = newInputs
                audioOutputDevices.value = newOutputs
            })
        } catch (err) {
            console.error('Error initializing devices:', err)
        }
    }

    // Connect WebSocket only
    function connectWebSocket() {
        if (!wsService.isConnected()) {
            wsService.connect()
            wsService.onMessage(handleWSMessage)
            wsService.onClose(() => {
                // WebSocket closed, only disconnect if we were in a channel
                if (isConnected.value) {
                    disconnect()
                }
            })
        }
    }

    // Join channel
    async function joinChannel(channelId: string, channelName: string) {
        try {
            // Ensure WebSocket is connected
            connectWebSocket()

            // Send join message
            wsService.send({
                type: 'join',
                data: {
                    name: authStore.user?.username,
                    channel_id: channelId
                }
            })

            // Initialize WebRTC
            await webrtcService.initialize(
                selectedInputDevice.value,
                selectedOutputDevice.value
            )

            // Create offer
            await webrtcService.createOffer()

            // Update state
            currentChannelId.value = channelId
            currentChannelName.value = channelName
            isConnected.value = true
        } catch (err) {
            console.error('Error joining channel:', err)
            throw err
        }
    }

    // Handle WebSocket messages
    function handleWSMessage(message: WSMessage) {
        switch (message.type) {
            case 'offer':
                if (message.sdp) {
                    webrtcService.handleOffer(message.sdp)
                }
                break

            case 'answer':
                if (message.sdp) {
                    webrtcService.handleAnswer(message.sdp)
                }
                break

            case 'candidate':
                if (message.candidate) {
                    webrtcService.addIceCandidate(message.candidate)
                }
                break

            case 'participants_detailed':
                if (message.data?.participants) {
                    participants.value = message.data.participants
                }
                break

            case 'user_action':
                // TODO: убрать какашку с типами
                if (message.data && typeof message.data.user_name === 'string' && typeof message.data.is_muted === 'boolean') {
                    const participantIndex = participants.value.findIndex(
                        p => p.username === message.data.user_name
                    )
                    if (participantIndex !== -1) {
                        const participant = participants.value[participantIndex]
                        if (participant) {
                            participant.is_muted = message.data.is_muted
                        }
                    }
                }
                break

            case 'error':
                console.error('Server error:', message.message)
                disconnect()
                break

            case 'pong':
                // Heartbeat response
                break

            default:
                console.warn('Unknown message type:', message.type)
        }
    }

    // Toggle mute
    function toggleMute() {
        isMuted.value = !isMuted.value
        webrtcService.toggleMute(isMuted.value)

        // Notify server
        wsService.send({
            type: 'user_action',
            data: {
                user_name: authStore.user?.username,
                is_muted: isMuted.value
            }
        })
    }

    // Update input device
    async function updateInputDevice(deviceId: string) {
        selectedInputDevice.value = deviceId
        if (isConnected.value) {
            await webrtcService.updateInputDevice(deviceId)
        }
    }

    // Update output device
    function updateOutputDevice(deviceId: string) {
        selectedOutputDevice.value = deviceId
        if (isConnected.value) {
            webrtcService.updateOutputDevice(deviceId)
        }
    }

    // Disconnect
    function disconnect() {
        // Send leave message
        if (wsService.isConnected()) {
            wsService.send({type: 'leave'})
        }

        // Disconnect WebRTC
        webrtcService.disconnect()

        // Reset state
        isConnected.value = false
        currentChannelId.value = null
        currentChannelName.value = null
        participants.value = []
        isMuted.value = false
        isSpeaking.value = false
    }

    // Cleanup on logout
    function cleanup() {
        disconnect()
        wsService.disconnect()
    }

    return {
        // State
        isConnected,
        currentChannelId,
        currentChannelName,
        participants,
        isMuted,
        isSpeaking,
        audioInputDevices,
        audioOutputDevices,
        selectedInputDevice,
        selectedOutputDevice,
        currentParticipant,

        // Actions
        initializeDevices,
        connectWebSocket,
        joinChannel,
        toggleMute,
        updateInputDevice,
        updateOutputDevice,
        disconnect,
        cleanup
    }
})

