package ops

import "time"

type SecurityEvent struct {
	ID         int64     `db:"id"`
	Timestamp  time.Time `db:"timestamp"`
	IPAddress  string    `db:"ip_address"`
	ActionType string    `db:"action_type"`
	FlagStatus string    `db:"flag_status"`
}
