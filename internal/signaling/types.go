package signaling

import (
	"encoding/json"

	"github.com/pion/webrtc/v4"
)

type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type JoinEvent struct {
	Name   string `json:"name"`
	RoomID string `json:"room_id"`
}

type SdpEvent struct {
	SDP string `json:"sdp"`
}

type CandidateEvent struct {
	Candidate webrtc.ICECandidateInit `json:"candidate"`
}
