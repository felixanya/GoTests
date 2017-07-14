package gameserver

import (
	"encoding/json"
)

const (
	CLIENT_STATUS_IN_GAME = 0
	CLIENT_STATUS_FAIL    = 1
	CLIENT_STATUS_WIN     = 2
)

// ServerClient state structure
type ServerClientState struct {
	Type        string `json:"type"`
	ID          uint32 `json:"id"`
	X           int16  `json:"x"`
	Y           int16  `json:"y"`
	Status      int8   `json:"status"`
	VisualState uint8   `json:"visualState"`
}

func NewServerClientState(id uint32) ServerClientState {
	state := ServerClientState{
		Type: "ClientState",
		ID:   id,
	}
	return state
}

func (client *ServerClientState) ToBytes() ([]byte, error) {
	return json.Marshal(client)
}
