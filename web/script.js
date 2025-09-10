function app() {
    return {
        name: '',
        channels: [],
        newChannelName: '',
        currentChannel: '',

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

        async init() {
            await this.getAudioDevices();
            // Обновляем список устройств при изменении
            navigator.mediaDevices.addEventListener('devicechange', () => this.getAudioDevices());

            await this.initializeWebSocket();
            await this.getChannels();
        },

        async getChannels() {
            try {
                const response = await fetch('/api/channels');
                this.channels = await response.json();
            } catch (err) {
                console.error('Error getting channels:', err);
            }
        },

        async createChannel() {
            if (!this.newChannelName.trim()) {
                return;
            }

            try {
                await fetch('/api/channels', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json'
                    },
                    body: JSON.stringify({channel_id: this.newChannelName})
                });
                this.newChannelName = '';
                await this.getChannels();
            } catch (err) {
                console.error('Error creating channel:', err);
            }
        },

        showDeleteModal: false,
        channelToDelete: '',

        async deleteChannel(channel) {
            this.channelToDelete = channel;
            this.showDeleteModal = true;
        },

        async confirmDelete() {
            try {
                await fetch(`/api/channels/${this.channelToDelete}`, {
                    method: 'DELETE'
                });
                await this.getChannels();
            } catch (err) {
                console.error('Error deleting channel:', err);
            }
            this.showDeleteModal = false;
            this.channelToDelete = '';
        },

        async joinChannel(channel) {
            this.currentChannel = channel;
            await this.connect();
        },

        async getAudioDevices() {
            // Получаем разрешение на доступ к микрофону (требуется для получения меток устройств)
            try {
                const stream = await navigator.mediaDevices.getUserMedia({audio: true});
                stream.getTracks().forEach(track => track.stop());
            } catch (err) {
                console.error('Error getting permissions:', err);
            }

            // Получаем список устройств
            const devices = await navigator.mediaDevices.enumerateDevices();

            this.audioInputDevices = devices.filter(d => d.kind === 'audioinput');
            this.audioOutputDevices = devices.filter(d => d.kind === 'audiooutput');

            if (this.audioInputDevices.length > 0 && !this.selectedInputDevice) {
                this.selectedInputDevice = this.audioInputDevices[0].deviceId;
            }

            if (this.audioOutputDevices.length > 0 && !this.selectedOutputDevice) {
                this.selectedOutputDevice = this.audioOutputDevices[0].deviceId;
            }
        },

        async updateAudioDevices() {
            if (this.localStream) {
                this.localStream.getTracks().forEach(track => track.stop());
            }

            const constraints = {
                audio: {
                    deviceId: this.selectedInputDevice ? {exact: this.selectedInputDevice} : undefined
                }
            };

            this.localStream = await navigator.mediaDevices.getUserMedia(constraints);

            if (this.pc) {
                const sender = this.pc.getSenders().find(s => s.track.kind === 'audio');
                if (sender) {
                    await sender.replaceTrack(this.localStream.getAudioTracks()[0]);
                }
            }
        },

        async updateOutputDevice() {
            if (this.selectedOutputDevice) {
                for (const audio of this.remoteAudioElements) {
                    if ('setSinkId' in audio) {
                        audio.setSinkId(this.selectedOutputDevice)
                            .catch(err => console.error('Error setting audio output:', err));
                    }
                }
            }
        },


        async connect() {
            if (!this.name.trim()) {
                alert('Please enter your name.');
                return;
            }

            try {
                this.ws.send(JSON.stringify({
                    type: 'join',
                    data: {name: this.name, channel_id: this.currentChannel}
                }));
                await this.initializeWebRTC();
            } catch (err) {
                console.error('Connection error:', err);
            }
        },

        async initializeWebSocket() {
            // TODO убрать залупу
            this.ws = new WebSocket(`${window.location.protocol === 'https:' ? 'wss' : 'ws'}://${window.location.host}/ws`);

            this.ws.onopen = () => {
                // ping-pong для поддержания соединения
                this.pingInterval = setInterval(() => {
                    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
                        this.ws.send(JSON.stringify({type: "ping"}));
                    }
                }, 30000); // каждые 30 сек
            };

            this.ws.onerror = (err) => console.error(err);

            this.ws.onmessage = (event) => this.handleWSMessage(event);

            this.ws.onclose = () => {
                clearInterval(this.pingInterval);
                this.pingInterval = null;
                if (this.pc) this.disconnect();
                alert('Connection closed');
            };
        },

        async initializeWebRTC() {
            const iceReq = await fetch('/ice')
            const iceServersResponse = await iceReq.json()

            const iceServers = [
                {urls: 'stun:stun.l.google.com:19302'},
                {
                    urls: iceServersResponse.urls,
                    username: iceServersResponse.username,
                    credential: iceServersResponse.credential,
                }
            ]

            this.pc = new RTCPeerConnection({
                iceServers: iceServers
            });

            // Настройка обработчиков WebRTC
            this.pc.onicecandidate = (e) => {
                if (e.candidate) {
                    this.ws.send(JSON.stringify({
                        type: 'candidate',
                        data: {
                            candidate: e.candidate
                        }
                    }));
                }
            };

            // Используем выбранное устройство ввода
            const constraints = {
                audio: {
                    deviceId: this.selectedInputDevice ? {exact: this.selectedInputDevice} : undefined
                }
            };

            // Получение медиа потока
            this.localStream = await navigator.mediaDevices.getUserMedia(constraints);

            // Добавляем треки в peer connection
            this.localStream.getTracks().forEach(track => {
                this.pc.addTrack(track, this.localStream);
            });

            // Обработка входящих треков
            this.pc.ontrack = (event) => {
                if (event.track.kind === 'audio') {
                    const audio = new Audio();
                    audio.srcObject = event.streams[0];

                    // Устанавливаем выбранное устройство вывода
                    if (this.selectedOutputDevice && 'setSinkId' in audio) {
                        audio.setSinkId(this.selectedOutputDevice)
                            .catch(err => console.error('Error setting audio output:', err));
                    }

                    this.remoteAudioElements.push(audio);

                    audio.play();
                }
            };

            const offer = await this.pc.createOffer();
            await this.pc.setLocalDescription(offer);
            this.ws.send(JSON.stringify({
                type: 'offer',
                data: {
                    sdp: offer.sdp
                }
            }));
        },

        async handleWSMessage(event) {
            const message = JSON.parse(event.data);

            switch (message.type) {
                case 'offer':
                    await this.pc.setRemoteDescription(message);
                    const answer = await this.pc.createAnswer();
                    await this.pc.setLocalDescription(answer);
                    this.ws.send(JSON.stringify({
                        type: 'offer',
                        data: {
                            sdp: answer.sdp
                        }
                    }));
                    break;

                case 'answer':
                    await this.pc.setRemoteDescription(message);
                    break;

                case 'candidate':
                    try {
                        await this.pc.addIceCandidate(message.candidate);
                    } catch (err) {
                        console.error('Error adding ICE candidate:', err);
                    }
                    break;
                case 'participants':
                    this.updateParticipants(message.list);
                    break;
                case 'error':
                    console.error('Server error:', message.message);
                    alert('Server error: ' + message.message);
                    this.disconnect();
                    break;
                case 'pong':
                    console.log('Pong received');
                    break;

                default:
                    console.warn('Unknown message type:', message.type);
            }
        },

        updateParticipants(participants) {
            this.participants = participants
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
                audio.srcObject = null;
            }
            this.remoteAudioElements = [];
            this.currentChannel = '';
        }
    }
}