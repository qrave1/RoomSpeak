export function initializeWebSocket(onOpen, onMessage, onClose, onError) {
    const ws = new WebSocket(`${window.location.protocol === 'https:' ? 'wss' : 'ws'}://${window.location.host}/api/v1/ws`);

    ws.onopen = onOpen;
    ws.onmessage = onMessage;
    ws.onclose = onClose;
    ws.onerror = onError;

    return ws;
}

export async function handleWSMessage(event, pc, updateParticipants, updateDetailedParticipants, disconnect) {
    const message = JSON.parse(event.data);

    switch (message.type) {
        case 'offer':
            await pc.setRemoteDescription(message);
            const answer = await pc.createAnswer();
            await pc.setLocalDescription(answer);
            this.ws.send(JSON.stringify({
                type: 'offer',
                data: {
                    sdp: answer.sdp
                }
            }));
            break;

        case 'answer':
            await pc.setRemoteDescription(message);
            break;

        case 'candidate':
            try {
                await pc.addIceCandidate(message.candidate);
            } catch (err) {
                console.error('Error adding ICE candidate:', err);
            }
            break;
        case 'participants_detailed':
            updateDetailedParticipants(message.data.participants);
            break;
        case 'user_action':
            const participantIndex = this.participants.findIndex(p => p.startsWith(message.data.user_name));
            if (participantIndex !== -1) {
                if (message.data.is_muted) {
                    this.participants[participantIndex] = `${message.data.user_name} <i class="fas fa-microphone-slash text-red-500"></i>`;
                } else {
                    this.participants[participantIndex] = message.data.user_name;
                }
            }
            // Обновляем детальную информацию об участниках
            updateDetailedParticipants(this.detailedParticipants.map(p => {
                if (p.username === message.data.user_name) {
                    return {...p, is_muted: message.data.is_muted};
                }
                return p;
            }));
            break;
        case 'error':
            console.error('Server error:', message.message);
            alert('Server error: ' + message.message);
            disconnect();
            break;
        case 'pong':
            break;

        default:
            console.warn('Unknown message type:', message.type);
    }
}
