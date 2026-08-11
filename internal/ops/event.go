package ops

import (
	"strings"
	"time"
)

// Flag values recorded against every request the security guard sees.
const (
	FlagAllowed = "ALLOWED"
	FlagBlocked = "BLOCKED"
)

// SecurityEvent is the stored shape, scanned positionally. It never reaches the
// wire -- EventDTO below is what clients see -- so it carries no struct tags.
type SecurityEvent struct {
	ID         int64
	Timestamp  time.Time
	IPAddress  string
	ActionType string
	FlagStatus string
}

// EventDTO is the wire shape. The database stores action_type as a single
// "GET /api/accounts" string; splitting it here saves every client from doing
// the same parsing, without changing the schema.
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
