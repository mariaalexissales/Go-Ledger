package ops

import (
	"sync"
	"sync/atomic"
)

// subscriberBuffer is how far behind a single SSE client may fall before its
// events start being dropped.
const subscriberBuffer = 256

// Hub fans security events out to live subscribers (the SSE stream).
//
// Publish never blocks. A browser tab that stops reading must not be able to
// stall the request path, so a full subscriber buffer drops the event and
// increments that subscriber's counter instead of applying backpressure.
type Hub struct {
	mu   sync.RWMutex
	subs map[int64]*subscriber
	next int64
}

type subscriber struct {
	ch      chan SecurityEvent
	dropped atomic.Int64
}

func NewHub() *Hub {
	return &Hub{subs: make(map[int64]*subscriber)}
}

// Subscription is a live feed of security events. Dropped reports how many
// events were discarded because the consumer fell behind, so the UI can say so
// rather than silently showing an incomplete picture.
type Subscription struct {
	hub *Hub
	id  int64
	sub *subscriber
}

func (s *Subscription) Events() <-chan SecurityEvent { return s.sub.ch }
func (s *Subscription) Dropped() int64               { return s.sub.dropped.Load() }

func (s *Subscription) Close() {
	s.hub.mu.Lock()
	defer s.hub.mu.Unlock()

	if _, ok := s.hub.subs[s.id]; ok {
		delete(s.hub.subs, s.id)
		close(s.sub.ch)
	}
}

func (h *Hub) Subscribe() *Subscription {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.next++
	sub := &subscriber{ch: make(chan SecurityEvent, subscriberBuffer)}
	h.subs[h.next] = sub

	return &Subscription{hub: h, id: h.next, sub: sub}
}

func (h *Hub) Publish(event SecurityEvent) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, sub := range h.subs {
		select {
		case sub.ch <- event:
		default:
			sub.dropped.Add(1)
		}
	}
}

func (h *Hub) SubscriberCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subs)
}
