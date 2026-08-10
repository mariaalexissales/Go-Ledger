package ops

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	heartbeatInterval = 15 * time.Second
	streamWriteWait   = 10 * time.Second
	maxReplayEvents   = 500
)

// streamEvents serves the live security event feed as Server-Sent Events.
//
// Each frame carries the database id, so a browser reconnecting after an HMR
// reload or a laptop sleep sends Last-Event-ID automatically and gets the gap
// replayed rather than silently losing events.
func (c *Console) streamEvents(w http.ResponseWriter, r *http.Request) {
	rc := http.NewResponseController(w)
	filter := parseEventFilter(r)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	// Tells nginx and Vite's dev proxy not to buffer the response.
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	if err := rc.Flush(); err != nil {
		return
	}

	// Subscribe before replaying, so events created during the replay query
	// queue up rather than falling through the gap. Anything replayed is then
	// skipped by id when the live loop starts.
	sub := c.hub.Subscribe()
	defer sub.Close()

	replayedThrough := int64(0)
	if since := lastEventID(r); since > 0 {
		replayedThrough = c.replay(w, rc, r, since, filter)
	}

	heartbeat := time.NewTicker(heartbeatInterval)
	defer heartbeat.Stop()

	reportedDrops := int64(0)

	for {
		select {
		case <-r.Context().Done():
			return

		case event, ok := <-sub.Events():
			if !ok {
				return
			}
			if event.ID <= replayedThrough || !filter.matches(event) {
				continue
			}
			if err := writeEvent(w, rc, NewEventDTO(event)); err != nil {
				return
			}

		case <-heartbeat.C:
			// An SSE comment keeps proxies from timing the connection out
			// without firing onmessage on the client.
			if err := writeRaw(w, rc, ": ping\n\n"); err != nil {
				return
			}

			// Tell the client when it has fallen behind, so the UI can say the
			// feed is a live tail rather than a complete record.
			if dropped := sub.Dropped(); dropped > reportedDrops {
				reportedDrops = dropped
				if err := writeLag(w, rc, dropped); err != nil {
					return
				}
			}
		}
	}
}

// replay sends stored events newer than sinceID and returns the highest id sent.
func (c *Console) replay(w http.ResponseWriter, rc *http.ResponseController, r *http.Request, sinceID int64, filter eventFilter) int64 {
	rows, err := c.db.Query(r.Context(), `
		SELECT id, timestamp, ip_address, action_type, flag_status
		FROM security_events
		WHERE id > $1
		ORDER BY id
		LIMIT $2
	`, sinceID, maxReplayEvents)
	if err != nil {
		return sinceID
	}
	defer rows.Close()

	highest := sinceID
	for rows.Next() {
		var e SecurityEvent
		if err := rows.Scan(&e.ID, &e.Timestamp, &e.IPAddress, &e.ActionType, &e.FlagStatus); err != nil {
			break
		}

		highest = e.ID
		if !filter.matches(e) {
			continue
		}
		if err := writeEvent(w, rc, NewEventDTO(e)); err != nil {
			break
		}
	}

	return highest
}

type eventFilter struct {
	flagStatus string
	ips        map[string]struct{}
}

func parseEventFilter(r *http.Request) eventFilter {
	q := r.URL.Query()
	f := eventFilter{flagStatus: q.Get("flag_status")}

	for _, part := range strings.Split(q.Get("ip_address"), ",") {
		if part = strings.TrimSpace(part); part != "" {
			if f.ips == nil {
				f.ips = make(map[string]struct{})
			}
			f.ips[part] = struct{}{}
		}
	}

	return f
}

func (f eventFilter) matches(e SecurityEvent) bool {
	if f.flagStatus != "" && e.FlagStatus != f.flagStatus {
		return false
	}
	if f.ips != nil {
		if _, ok := f.ips[e.IPAddress]; !ok {
			return false
		}
	}
	return true
}

func lastEventID(r *http.Request) int64 {
	raw := r.Header.Get("Last-Event-ID")
	if raw == "" {
		raw = r.URL.Query().Get("since_id")
	}

	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id < 0 {
		return 0
	}
	return id
}

func writeEvent(w http.ResponseWriter, rc *http.ResponseController, dto EventDTO) error {
	payload, err := json.Marshal(dto)
	if err != nil {
		return err
	}

	return writeRaw(w, rc, fmt.Sprintf("id: %d\nevent: security_event\ndata: %s\n\n", dto.ID, payload))
}

func writeLag(w http.ResponseWriter, rc *http.ResponseController, dropped int64) error {
	return writeRaw(w, rc, fmt.Sprintf("event: lag\ndata: {\"dropped\":%d}\n\n", dropped))
}

// writeRaw applies a write deadline so a stalled client cannot pin the handler
// goroutine indefinitely.
func writeRaw(w http.ResponseWriter, rc *http.ResponseController, s string) error {
	_ = rc.SetWriteDeadline(time.Now().Add(streamWriteWait))

	if _, err := io.WriteString(w, s); err != nil {
		return err
	}
	return rc.Flush()
}
