package ops

import (
	"sync"
	"sync/atomic"
)

const subscriberBuffer = 256

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
