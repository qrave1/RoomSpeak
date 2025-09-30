export async function getChannels() {
    try {
        const response = await fetch('/api/v1/channels');
        if (!response.ok) {
            throw new Error('Network response was not ok');
        }
        const data = await response.json();
        console.log('Channels data from backend:', data);
        return data.channels;
    } catch (err) {
        console.error('Error getting channels:', err);
        return [];
    }
}

export async function createChannel(newChannelName, newChannelIsPublic) {
    if (!newChannelName.trim()) {
        return false;
    }

    try {
        await fetch('/api/v1/channels', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                name: newChannelName,
                is_public: newChannelIsPublic
            })
        });
        return true;
    } catch (err) {
        console.error('Error creating channel:', err);
        return false;
    }
}

export async function deleteChannel(channelID) {
    try {
        await fetch(`/api/v1/channels/${channelID}`, {
            method: 'DELETE'
        });
        return true;
    } catch (err) {
        console.error('Error deleting channel:', err);
        return false;
    }
}
