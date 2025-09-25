export async function initializeWebRTC(ws, localStream, selectedInputDevice, selectedOutputDevice, remoteAudioElements) {
    const iceReq = await fetch('/api/v1/ice');
    const iceServersResponse = await iceReq.json();

    const iceServers = [
        {urls: 'stun:stun.l.google.com:19302'},
        {
            urls: iceServersResponse.urls,
            username: iceServersResponse.username,
            credential: iceServersResponse.credential,
        }
    ];

    const pc = new RTCPeerConnection({
        iceServers: iceServers
    });

    pc.onicecandidate = (e) => {
        if (e.candidate) {
            ws.send(JSON.stringify({
                type: 'candidate',
                data: {
                    candidate: e.candidate
                }
            }));
        }
    };

    const constraints = {
        audio: {
            deviceId: selectedInputDevice ? {exact: selectedInputDevice} : undefined
        }
    };

    localStream = await navigator.mediaDevices.getUserMedia(constraints);

    localStream.getTracks().forEach(track => {
        pc.addTrack(track, localStream);
    });

    pc.ontrack = (event) => {
        if (event.track.kind === 'audio') {
            const audio = new Audio();
            audio.srcObject = event.streams[0];
            audio.autoplay = true;
            document.body.appendChild(audio);

            if (selectedOutputDevice && 'setSinkId' in audio) {
                audio.setSinkId(selectedOutputDevice)
                    .catch(err => console.error('Error setting audio output:', err));
            }

            remoteAudioElements.push(audio);

            audio.play().catch(err => console.error('Failed to play remote audio:', err));
        }
    };

    return {pc, localStream};
}

export async function createOffer(pc, ws) {
    const offer = await pc.createOffer();
    await pc.setLocalDescription(offer);
    ws.send(JSON.stringify({
        type: 'offer',
        data: {
            sdp: offer.sdp
        }
    }));
}
