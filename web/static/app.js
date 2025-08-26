class RoomSpeak {
    constructor() {
        // WebSocket подключение
        this.ws = null;
        this.clientId = null;
        this.currentRoom = null;
        this.userName = null;
        this.isMuted = false;

        // WebRTC
        this.peerConnection = null;
        this.localStream = null;
        this.remoteStreams = new Map();

        // UI элементы
        this.screens = {
            mainMenu: document.getElementById('main-menu'),
            createRoom: document.getElementById('create-room-screen'),
            joinRoom: document.getElementById('join-room-screen'),
            room: document.getElementById('room-screen')
        };

        this.elements = {
            // Кнопки главного меню
            createRoomBtn: document.getElementById('create-room-btn'),
            joinRoomBtn: document.getElementById('join-room-btn'),
            
            // Формы
            createRoomForm: document.getElementById('create-room-form'),
            joinRoomForm: document.getElementById('join-room-form'),
            
            // Кнопки отмены
            cancelCreate: document.getElementById('cancel-create'),
            cancelJoin: document.getElementById('cancel-join'),
            
            // Элементы комнаты
            roomTitle: document.getElementById('room-title'),
            leaveRoomBtn: document.getElementById('leave-room-btn'),
            participantsList: document.getElementById('participants-list'),
            participantsCount: document.getElementById('participants-count'),
            audioStreams: document.getElementById('audio-streams'),
            
            // Контроли
            muteBtn: document.getElementById('mute-btn'),
            muteIcon: document.getElementById('mute-icon'),
            muteText: document.getElementById('mute-text'),
            
            // Статус и ошибки
            connectionStatus: document.getElementById('connection-status'),
            statusText: document.getElementById('status-text'),
            statusIndicator: document.getElementById('status-indicator'),
            errorMessage: document.getElementById('error-message'),
            errorText: document.getElementById('error-text'),
            closeError: document.getElementById('close-error'),
            
            // Список комнат
            roomsList: document.getElementById('rooms-list')
        };

        this.init();
    }

    async init() {
        this.setupEventListeners();
        await this.loadRooms();
        this.connectWebSocket();
    }

    setupEventListeners() {
        // Главное меню
        this.elements.createRoomBtn.addEventListener('click', () => this.showScreen('createRoom'));
        this.elements.joinRoomBtn.addEventListener('click', () => this.showScreen('joinRoom'));

        // Формы
        this.elements.createRoomForm.addEventListener('submit', (e) => this.handleCreateRoom(e));
        this.elements.joinRoomForm.addEventListener('submit', (e) => this.handleJoinRoom(e));

        // Кнопки отмены
        this.elements.cancelCreate.addEventListener('click', () => this.showScreen('mainMenu'));
        this.elements.cancelJoin.addEventListener('click', () => this.showScreen('mainMenu'));

        // Комната
        this.elements.leaveRoomBtn.addEventListener('click', () => this.leaveRoom());
        this.elements.muteBtn.addEventListener('click', () => this.toggleMute());

        // Закрытие ошибки
        this.elements.closeError.addEventListener('click', () => this.hideError());
    }

    showScreen(screenName) {
        // Скрываем все экраны
        Object.values(this.screens).forEach(screen => screen.classList.remove('active'));
        
        // Показываем нужный экран
        if (this.screens[screenName]) {
            this.screens[screenName].classList.add('active');
        }
    }

    updateConnectionStatus(status, text) {
        this.elements.statusText.textContent = text;
        this.elements.statusIndicator.className = `status-indicator ${status}`;
    }

    showError(message) {
        this.elements.errorText.textContent = message;
        this.elements.errorMessage.classList.add('show');
        
        // Автоматически скрыть через 5 секунд
        setTimeout(() => this.hideError(), 5000);
    }

    hideError() {
        this.elements.errorMessage.classList.remove('show');
    }

    // WebSocket методы
    connectWebSocket() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws`;

        this.updateConnectionStatus('connecting', 'Подключение...');

        this.ws = new WebSocket(wsUrl);

        this.ws.onopen = () => {
            console.log('WebSocket connected');
            this.updateConnectionStatus('connected', 'Подключено');
        };

        this.ws.onmessage = (event) => {
            const message = JSON.parse(event.data);
            this.handleWebSocketMessage(message);
        };

        this.ws.onclose = () => {
            console.log('WebSocket disconnected');
            this.updateConnectionStatus('disconnected', 'Отключено');
            
            // Переподключение через 3 секунды
            setTimeout(() => this.connectWebSocket(), 3000);
        };

        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
            this.showError('Ошибка подключения к серверу');
        };
    }

    sendWebSocketMessage(action, data) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify({ action, data }));
        }
    }

    handleWebSocketMessage(message) {
        console.log('WebSocket message:', message);

        switch (message.action) {
            case 'connected':
                this.clientId = message.data.clientId;
                break;

            case 'room_created':
                this.joinCreatedRoom(message.data);
                break;

            case 'joined_room':
                this.handleJoinedRoom(message.data);
                break;

            case 'user_joined':
                this.handleUserJoined(message.data);
                break;

            case 'user_left':
                this.handleUserLeft(message.data);
                break;

            case 'mute_changed':
                this.handleMuteChanged(message.data);
                break;

            case 'offer':
                this.handleOffer(message.data);
                break;

            case 'answer':
                this.handleAnswer(message.data);
                break;

            case 'ice_candidate':
                this.handleIceCandidate(message.data);
                break;

            case 'room_users':
                this.updateParticipantsList(message.data.users);
                break;

            case 'error':
                this.showError(message.data.message);
                break;

            default:
                console.log('Unknown message action:', message.action);
        }
    }

    // Управление комнатами
    async loadRooms() {
        try {
            const response = await fetch('/api/rooms');
            const rooms = await response.json();
            this.displayRooms(rooms);
        } catch (error) {
            console.error('Error loading rooms:', error);
            this.elements.roomsList.innerHTML = '<p class="loading">Ошибка загрузки комнат</p>';
        }
    }

    displayRooms(rooms) {
        if (rooms.length === 0) {
            this.elements.roomsList.innerHTML = '<p class="loading">Нет доступных комнат</p>';
            return;
        }

        this.elements.roomsList.innerHTML = rooms.map(room => `
            <div class="room-card" onclick="app.quickJoinRoom('${room.room_id}', '${room.name}')">
                <h4>${room.name}</h4>
                <p>ID: ${room.room_id}</p>
                <p>Создана: ${new Date(room.created_at).toLocaleString()}</p>
            </div>
        `).join('');
    }

    quickJoinRoom(roomId, roomName) {
        const userName = prompt(`Введите ваше имя для входа в комнату "${roomName}":`);
        if (userName) {
            this.userName = userName;
            this.joinRoom(roomId);
        }
    }

    async handleCreateRoom(e) {
        e.preventDefault();
        
        const roomName = document.getElementById('room-name').value;
        const userName = document.getElementById('user-name-create').value;

        this.userName = userName;
        
        this.sendWebSocketMessage('create_room', { roomName });
    }

    async handleJoinRoom(e) {
        e.preventDefault();
        
        const roomId = document.getElementById('room-id').value;
        const userName = document.getElementById('user-name-join').value;

        this.userName = userName;
        this.joinRoom(roomId);
    }

    joinRoom(roomId) {
        this.sendWebSocketMessage('join_room', { roomId });
    }

    async joinCreatedRoom(roomData) {
        this.currentRoom = roomData;
        this.elements.roomTitle.textContent = roomData.name;
        
        await this.setupWebRTC();
        this.joinRoom(roomData.room_id);
    }

    async handleJoinedRoom(data) {
        this.currentRoom = { room_id: data.roomId, name: `Комната ${data.roomId}` };
        this.elements.roomTitle.textContent = this.currentRoom.name;
        
        await this.setupWebRTC();
        this.updateParticipantsList(data.users);
        this.showScreen('room');
    }

    leaveRoom() {
        if (this.peerConnection) {
            this.peerConnection.close();
            this.peerConnection = null;
        }

        if (this.localStream) {
            this.localStream.getTracks().forEach(track => track.stop());
            this.localStream = null;
        }

        this.currentRoom = null;
        this.remoteStreams.clear();
        this.showScreen('mainMenu');
        this.loadRooms();
    }

    // WebRTC методы
    async setupWebRTC() {
        try {
            // Получаем доступ к микрофону
            this.localStream = await navigator.mediaDevices.getUserMedia({ 
                audio: {
                    echoCancellation: true,
                    noiseSuppression: true,
                    autoGainControl: true
                }
            });

            // Создаем PeerConnection
            this.peerConnection = new RTCPeerConnection({
                iceServers: [
                    { urls: 'stun:stun.l.google.com:19302' }
                ]
            });

            // Добавляем локальный поток
            this.localStream.getTracks().forEach(track => {
                this.peerConnection.addTrack(track, this.localStream);
            });

            // Обработчики WebRTC событий
            this.peerConnection.onicecandidate = (event) => {
                if (event.candidate) {
                    this.sendWebSocketMessage('ice_candidate', {
                        candidate: event.candidate.candidate,
                        sdpMid: event.candidate.sdpMid,
                        sdpMLineIndex: event.candidate.sdpMLineIndex
                    });
                }
            };

            this.peerConnection.ontrack = (event) => {
                console.log('Received remote track:', event);
                this.handleRemoteTrack(event);
            };

            this.peerConnection.onconnectionstatechange = () => {
                console.log('Connection state:', this.peerConnection.connectionState);
            };

            // Создаем и отправляем offer
            const offer = await this.peerConnection.createOffer();
            await this.peerConnection.setLocalDescription(offer);

            this.sendWebSocketMessage('offer', {
                type: 'offer',
                sdp: offer.sdp
            });

        } catch (error) {
            console.error('Error setting up WebRTC:', error);
            this.showError('Не удалось получить доступ к микрофону');
        }
    }

    async handleAnswer(data) {
        try {
            const answer = new RTCSessionDescription({
                type: 'answer',
                sdp: data.sdp
            });
            await this.peerConnection.setRemoteDescription(answer);
        } catch (error) {
            console.error('Error handling answer:', error);
        }
    }

    async handleIceCandidate(data) {
        try {
            const candidate = new RTCIceCandidate({
                candidate: data.candidate,
                sdpMid: data.sdpMid,
                sdpMLineIndex: data.sdpMLineIndex
            });
            await this.peerConnection.addIceCandidate(candidate);
        } catch (error) {
            console.error('Error adding ICE candidate:', error);
        }
    }

    handleRemoteTrack(event) {
        const [stream] = event.streams;
        const trackId = event.track.id;

        // Создаем audio элемент для воспроизведения
        let audio = document.getElementById(`audio-${trackId}`);
        if (!audio) {
            audio = document.createElement('audio');
            audio.id = `audio-${trackId}`;
            audio.autoplay = true;
            audio.controls = false;
            this.elements.audioStreams.appendChild(audio);
        }

        audio.srcObject = stream;
        this.remoteStreams.set(trackId, stream);

        console.log(`Added remote track: ${trackId}`);
    }

    // Управление участниками
    handleUserJoined(userData) {
        console.log('User joined:', userData);
        this.sendWebSocketMessage('get_room_users', {});
    }

    handleUserLeft(userData) {
        console.log('User left:', userData);
        this.sendWebSocketMessage('get_room_users', {});
        
        // Удаляем аудио элемент пользователя
        const audio = document.getElementById(`audio-${userData.userId}`);
        if (audio) {
            audio.remove();
        }
        this.remoteStreams.delete(userData.userId);
    }

    handleMuteChanged(userData) {
        console.log('User mute changed:', userData);
        this.sendWebSocketMessage('get_room_users', {});
    }

    updateParticipantsList(users) {
        this.elements.participantsCount.textContent = users.length;
        
        this.elements.participantsList.innerHTML = users.map(user => {
            const initial = user.userName ? user.userName.charAt(0).toUpperCase() : '?';
            const isCurrentUser = user.id === this.clientId;
            const muteIcon = user.isMuted ? '🔇' : '🎤';
            const muteClass = user.isMuted ? 'muted' : '';
            
            return `
                <div class="participant ${muteClass}" id="participant-${user.id}">
                    <div class="participant-avatar">${initial}</div>
                    <div class="participant-info">
                        <div class="participant-name">
                            ${user.userName || 'Пользователь'} ${isCurrentUser ? '(Вы)' : ''}
                        </div>
                        <div class="participant-status">
                            ${user.isConnected ? 'Онлайн' : 'Подключается...'}
                        </div>
                    </div>
                    <span class="participant-mute-icon">${muteIcon}</span>
                </div>
            `;
        }).join('');
    }

    // Управление микрофоном
    async toggleMute() {
        this.isMuted = !this.isMuted;
        
        // Управляем локальным потоком
        if (this.localStream) {
            this.localStream.getAudioTracks().forEach(track => {
                track.enabled = !this.isMuted;
            });
        }

        // Обновляем UI
        this.updateMuteButton();
        
        // Уведомляем сервер
        this.sendWebSocketMessage('toggle_mute', { isMuted: this.isMuted });
    }

    updateMuteButton() {
        if (this.isMuted) {
            this.elements.muteBtn.classList.add('muted');
            this.elements.muteIcon.textContent = '🔇';
            this.elements.muteText.textContent = 'Микрофон выключен';
        } else {
            this.elements.muteBtn.classList.remove('muted');
            this.elements.muteIcon.textContent = '🎤';
            this.elements.muteText.textContent = 'Микрофон включен';
        }
    }
}

// Инициализация приложения
let app;
document.addEventListener('DOMContentLoaded', () => {
    app = new RoomSpeak();
}); 