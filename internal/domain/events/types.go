package events

import (
	"encoding/json"

	"github.com/pion/webrtc/v4"
)

// Message - общее событие
type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// JoinEvent - событие при подключении нового участника в комнату
type JoinEvent struct {
	ChannelID string `json:"channel_id"`
}

// SdpEvent - события связанные с SDP (offer, answer, ice)
type SdpEvent struct {
	SDP string `json:"sdp"`
}

// IceCandidateEvent - ICE кандидаты
type IceCandidateEvent struct {
	Candidate webrtc.ICECandidateInit `json:"candidate"`
}

// ParticipantListEvent - событие со списком активных участников комнаты
type ParticipantListEvent struct {
	List []string `json:"list"`
}

// UserActionEvent - событие, связанное с действием пользователя, например, отключение микрофона
type UserActionEvent struct {
	UserName string `json:"user_name"`
	IsMuted  bool   `json:"is_muted"`
}
