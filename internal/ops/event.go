package ops

import (
	"strings"
	"time"
)

const (
	FlagAllowed = "ALLOWED"
	FlagBlocked = "BLOCKED"
)

type SecurityEvent struct {
	ID         int64
	Timestamp  time.Time
	IPAddress  string
	ActionType string
	FlagStatus string
}

type EventDTO struct {
	ID         int64     `json:"id"`
	Timestamp  time.Time `json:"timestamp"`
	IPAddress  string    `json:"ip_address"`
	ActionType string    `json:"action_type"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	FlagStatus string    `json:"flag_status"`
	Blocked    bool      `json:"blocked"`
}

func NewEventDTO(e SecurityEvent) EventDTO {
	method, path, found := strings.Cut(e.ActionType, " ")
	if !found {
		method, path = "", e.ActionType
	}

	return EventDTO{
		ID:         e.ID,
		Timestamp:  e.Timestamp,
		IPAddress:  e.IPAddress,
		ActionType: e.ActionType,
		Method:     method,
		Path:       path,
		FlagStatus: e.FlagStatus,
		Blocked:    e.FlagStatus == FlagBlocked,
	}
}
