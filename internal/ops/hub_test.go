package ops

import (
	"sync"
	"testing"
	"time"
)

func TestHubPublishReachesEverySubscriber(t *testing.T) {
	hub := NewHub()

	first := hub.Subscribe()
	defer first.Close()
	second := hub.Subscribe()
	defer second.Close()

	if got := hub.SubscriberCount(); got != 2 {
		t.Fatalf("SubscriberCount() = %d, want 2", got)
	}

	hub.Publish(SecurityEvent{ID: 7, IPAddress: "192.0.2.1", FlagStatus: FlagBlocked})

	for i, sub := range []*Subscription{first, second} {
		select {
		case got := <-sub.Events():
			if got.ID != 7 {
				t.Errorf("subscriber %d received event ID %d, want 7", i, got.ID)
			}
		default:
			t.Errorf("subscriber %d received nothing", i)
		}
	}
}

func TestHubPublishNeverBlocksOnAFullSubscriber(t *testing.T) {
	hub := NewHub()
	sub := hub.Subscribe()
	defer sub.Close()

	const overshoot = 50
	done := make(chan struct{})

	go func() {
		defer close(done)
		for i := range subscriberBuffer + overshoot {
			hub.Publish(SecurityEvent{ID: int64(i)})
		}
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Publish blocked once the subscriber buffer filled; it must drop instead")
	}

	if got := sub.Dropped(); got != overshoot {
		t.Errorf("Dropped() = %d, want %d", got, overshoot)
	}

	if got := <-sub.Events(); got.ID != 0 {
		t.Errorf("first buffered event ID = %d, want 0 (dropping must discard the newest, not the queue)", got.ID)
	}
}

func TestHubCloseUnsubscribes(t *testing.T) {
	hub := NewHub()
	sub := hub.Subscribe()

	sub.Close()

	if got := hub.SubscriberCount(); got != 0 {
		t.Errorf("SubscriberCount() after Close = %d, want 0", got)
	}

	if _, open := <-sub.Events(); open {
		t.Error("Events() channel is still open after Close")
	}

	hub.Publish(SecurityEvent{ID: 1})
}

func TestHubCloseIsIdempotent(t *testing.T) {
	hub := NewHub()
	sub := hub.Subscribe()

	sub.Close()
	sub.Close() // Must not panic by closing an already-closed channel.
}

func TestHubConcurrentPublishAndSubscribe(t *testing.T) {
	hub := NewHub()

	var wg sync.WaitGroup

	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 200 {
				hub.Publish(SecurityEvent{ID: int64(i)})
			}
		}()
	}

	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sub := hub.Subscribe()
			defer sub.Close()

			for range 20 {
				select {
				case <-sub.Events():
				default:
				}
			}
			_ = sub.Dropped()
			_ = hub.SubscriberCount()
		}()
	}

	wg.Wait()

	if got := hub.SubscriberCount(); got != 0 {
		t.Errorf("SubscriberCount() = %d after every subscriber closed, want 0", got)
	}
}
