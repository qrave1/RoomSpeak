export async function getAudioDevices() {
    try {
        const stream = await navigator.mediaDevices.getUserMedia({audio: true});
        stream.getTracks().forEach(track => track.stop());
    } catch (err) {
        console.error('Error getting permissions:', err);
    }

    const devices = await navigator.mediaDevices.enumerateDevices();
    const audioInputDevices = devices.filter(d => d.kind === 'audioinput');
    const audioOutputDevices = devices.filter(d => d.kind === 'audiooutput');

    return {audioInputDevices, audioOutputDevices};
}

export async function updateAudioDevices(localStream, selectedInputDevice) {
    if (localStream) {
        localStream.getTracks().forEach(track => track.stop());
    }

    const constraints = {
        audio: {
            deviceId: selectedInputDevice ? {exact: selectedInputDevice} : undefined
        }
    };

    const newStream = await navigator.mediaDevices.getUserMedia(constraints);

    return newStream;
}

export function updateOutputDevice(remoteAudioElements, selectedOutputDevice) {
    if (selectedOutputDevice) {
        for (const audio of remoteAudioElements) {
            if ('setSinkId' in audio) {
                audio.setSinkId(selectedOutputDevice)
                    .catch(err => console.error('Error setting audio output:', err));
            }
        }
    }
}
