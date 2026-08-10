package ops

import (
	"context"
	"log"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	recorderQueueSize     = 4096
	recorderBatchSize     = 200
	recorderFlushInterval = 100 * time.Millisecond
	recorderFlushTimeout  = 5 * time.Second
)

// Recorder persists security events and then fans them out to live subscribers.
//
// Events are published to the hub only after the insert returns, so every
// streamed event carries a real database id. That id is what makes SSE
// reconnection via Last-Event-ID lossless.
type Recorder struct {
	db  *pgxpool.Pool
	hub *Hub

	queue   chan SecurityEvent
	done    chan struct{}
	dropped atomic.Int64
	failed  atomic.Int64
}

func NewRecorder(db *pgxpool.Pool, hub *Hub) *Recorder {
	return &Recorder{
		db:    db,
		hub:   hub,
		queue: make(chan SecurityEvent, recorderQueueSize),
		done:  make(chan struct{}),
	}
}

// Record enqueues an event. It never blocks: if the queue is saturated the
// event is dropped and counted.
func (r *Recorder) Record(ip, actionType, flagStatus string, at time.Time) {
	event := SecurityEvent{
		Timestamp:  at,
		IPAddress:  ip,
		ActionType: actionType,
		FlagStatus: flagStatus,
	}

	select {
	case r.queue <- event:
	default:
		r.dropped.Add(1)
	}
}

// Dropped counts events discarded because the queue was full.
func (r *Recorder) Dropped() int64 { return r.dropped.Load() }

// Failed counts events lost to database errors.
func (r *Recorder) Failed() int64 { return r.failed.Load() }

// Run drives the batching loop until ctx is cancelled, then drains whatever is
// still queued before signalling completion via Done.
func (r *Recorder) Run(ctx context.Context) {
	defer close(r.done)

	ticker := time.NewTicker(recorderFlushInterval)
	defer ticker.Stop()

	batch := make([]SecurityEvent, 0, recorderBatchSize)

	for {
		select {
		case <-ctx.Done():
			batch = r.drain(batch)
			r.flush(batch)
			return

		case event := <-r.queue:
			batch = append(batch, event)
			if len(batch) >= recorderBatchSize {
				r.flush(batch)
				batch = batch[:0]
			}

		case <-ticker.C:
			if len(batch) > 0 {
				r.flush(batch)
				batch = batch[:0]
			}
		}
	}
}

// Done closes once Run has flushed everything and returned.
func (r *Recorder) Done() <-chan struct{} { return r.done }

// drain pulls whatever is left in the queue without blocking.
func (r *Recorder) drain(batch []SecurityEvent) []SecurityEvent {
	for {
		select {
		case event := <-r.queue:
			batch = append(batch, event)
		default:
			return batch
		}
	}
}

func (r *Recorder) flush(batch []SecurityEvent) {
	if len(batch) == 0 {
		return
	}

	ips := make([]string, len(batch))
	actions := make([]string, len(batch))
	flags := make([]string, len(batch))
	times := make([]time.Time, len(batch))

	for i, event := range batch {
		ips[i] = event.IPAddress
		actions[i] = event.ActionType
		flags[i] = event.FlagStatus
		times[i] = event.Timestamp
	}

	// A fresh context: flush must still complete when Run was cancelled by
	// shutdown.
	ctx, cancel := context.WithTimeout(context.Background(), recorderFlushTimeout)
	defer cancel()

	rows, err := r.db.Query(ctx, `
		INSERT INTO security_events (ip_address, action_type, flag_status, timestamp)
		SELECT * FROM unnest($1::varchar[], $2::varchar[], $3::varchar[], $4::timestamptz[])
		RETURNING id, ip_address, action_type, flag_status, timestamp
	`, ips, actions, flags, times)

	if err != nil {
		r.failed.Add(int64(len(batch)))
		log.Printf("security event batch insert failed (%d events): %v", len(batch), err)
		return
	}
	defer rows.Close()

	stored := make([]SecurityEvent, 0, len(batch))
	for rows.Next() {
		var event SecurityEvent
		if err := rows.Scan(
			&event.ID,
			&event.IPAddress,
			&event.ActionType,
			&event.FlagStatus,
			&event.Timestamp,
		); err != nil {
			r.failed.Add(1)
			continue
		}
		stored = append(stored, event)
	}

	if err := rows.Err(); err != nil {
		log.Printf("security event batch read failed: %v", err)
	}

	for _, event := range stored {
		r.hub.Publish(event)
	}
}
