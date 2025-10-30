export function toggleMute(isMuted, localStream, ws) {
    const newMutedState = !isMuted;
    localStream.getAudioTracks()[0].enabled = !newMutedState;

    ws.send(JSON.stringify({
        type: 'mute',
        data: {
            is_muted: newMutedState
        }
    }));

    const muteButton = document.getElementById('mute-btn');
    if (newMutedState) {
        muteButton.innerHTML = '<i class="fas fa-microphone-slash"></i>';
        muteButton.classList.remove('bg-gray-600');
        muteButton.classList.add('bg-red-600');
    } else {
        muteButton.innerHTML = '<i class="fas fa-microphone"></i>';
        muteButton.classList.remove('bg-red-600');
        muteButton.classList.add('bg-gray-600');
    }

    return newMutedState;
}
