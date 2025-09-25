import {checkAuth, login, register, logout} from './auth.js';
import {getChannels, createChannel, deleteChannel} from './channels.js';
import {getAudioDevices, updateAudioDevices, updateOutputDevice} from './devices.js';
import {initializeWebRTC, createOffer} from './webrtc.js';
import {initializeWebSocket, handleWSMessage} from './websocket.js';
import {toggleMute} from './ui.js';

window.app = function () {
    return {
        // State
        name: '',
        channels: [],
        newChannelName: '',
        newChannelIsPublic: false,
        currentChannelID: '',
        currentChannelName: '',
        audioInputDevices: [],
        audioOutputDevices: [],
        selectedInputDevice: '',
        selectedOutputDevice: '',
        ws: null,
        pingInterval: null,
        pc: null,
        localStream: null,
        participants: [],
        remoteAudioElements: [],
        showDeleteModal: false,
        channelIDToDelete: '',
        channelNameToDelete: '',
        isAuthenticated: false,
        showLogin: true,
        auth: {
            username: '',
            password: ''
        },
        isMuted: false,

        // Init
        async init() {
            const authResult = await checkAuth();
            this.isAuthenticated = authResult.isAuthenticated;
            if (this.isAuthenticated) {
                this.name = authResult.user.username;
                await this.loadInitialData();
            }
        },

        async loadInitialData() {
            const {audioInputDevices, audioOutputDevices} = await getAudioDevices();
            this.audioInputDevices = audioInputDevices;
            this.audioOutputDevices = audioOutputDevices;
            if (this.audioInputDevices.length > 0) {
                this.selectedInputDevice = this.audioInputDevices[0].deviceId;
            }
            if (this.audioOutputDevices.length > 0) {
                this.selectedOutputDevice = this.audioOutputDevices[0].deviceId;
            }

            navigator.mediaDevices.addEventListener('devicechange', async () => {
                const {audioInputDevices, audioOutputDevices} = await getAudioDevices();
                this.audioInputDevices = audioInputDevices;
                this.audioOutputDevices = audioOutputDevices;
            });

            this.channels = await getChannels();
            this.ws = initializeWebSocket(
                this.onWsOpen.bind(this),
                this.onWsMessage.bind(this),
                this.onWsClose.bind(this),
                this.onWsError.bind(this)
            );
        },

        // Auth
        async login() {
            const success = await login(this.auth);
            if (success) {
                window.location.reload();
            }
        },
        async register() {
            const success = await register(this.auth);
            if (success) {
                this.showLogin = true;
            }
        },
        logout() {
            logout();
        },

        // Channels
        async createChannel() {
            const success = await createChannel(this.newChannelName, this.newChannelIsPublic);
            if (success) {
                this.newChannelName = '';
                this.newChannelIsPublic = false;
                this.channels = await getChannels();
            }
        },
        openDeleteModal(channel) {
            this.channelIDToDelete = channel.id;
            this.channelNameToDelete = channel.name;
            this.showDeleteModal = true;
        },
        async confirmDelete() {
            const success = await deleteChannel(this.channelIDToDelete);
            if (success) {
                this.channels = await getChannels();
            }
            this.showDeleteModal = false;
            this.channelIDToDelete = '';
        },
        async joinChannel(channel) {
            this.currentChannelID = channel.id;
            this.currentChannelName = channel.name;
            await this.connect();
        },

        // Devices
        async onDeviceChange() {
            this.localStream = await updateAudioDevices(this.localStream, this.selectedInputDevice);
            if (this.pc) {
                const sender = this.pc.getSenders().find(s => s.track.kind === 'audio');
                if (sender) {
                    await sender.replaceTrack(this.localStream.getAudioTracks()[0]);
                }
            }
        },
        onOutputDeviceChange() {
            updateOutputDevice(this.remoteAudioElements, this.selectedOutputDevice);
        },

        // WebRTC & WebSocket
        async connect() {
            if (!this.name.trim()) {
                alert('Please enter your name.');
                return;
            }

            try {
                this.ws.send(JSON.stringify({
                    type: 'join',
                    data: {name: this.name, channel_id: this.currentChannelID}
                }));
                const {pc, localStream} = await initializeWebRTC(this.ws, this.localStream, this.selectedInputDevice, this.selectedOutputDevice, this.remoteAudioElements);
                this.pc = pc;
                this.localStream = localStream;
                await createOffer(this.pc, this.ws);
            } catch (err) {
                console.error('Connection error:', err);
            }
        },

        onWsOpen() {
            this.pingInterval = setInterval(() => {
                if (this.ws && this.ws.readyState === WebSocket.OPEN) {
                    this.ws.send(JSON.stringify({type: "ping"}));
                }
            }, 30000);
        },
        onWsMessage(event) {
            handleWSMessage(event, this.pc, this.updateParticipants.bind(this), this.disconnect.bind(this));
        },
        onWsClose() {
            clearInterval(this.pingInterval);
            this.pingInterval = null;
            if (this.pc) this.disconnect();
            alert('Connection closed');
        },
        onWsError(err) {
            console.error(err);
        },

        updateParticipants(participants) {
            this.participants = participants;
        },

        toggleMute() {
            this.isMuted = toggleMute(this.isMuted, this.localStream, this.ws);
        },

        disconnect() {
            if (this.pc) {
                this.pc.close();
                this.pc = null;
            }

            if (this.ws && this.ws.readyState === WebSocket.OPEN) {
                this.ws.send(JSON.stringify({type: "leave"}));
            }

            if (this.localStream) {
                this.localStream.getTracks().forEach(track => track.stop());
                this.localStream = null;
            }

            for (const audio of this.remoteAudioElements) {
                audio.pause();
                audio.srcObject = null;
                audio.remove();
            }
            this.remoteAudioElements = [];
            this.currentChannelID = '';
        }
    }
}