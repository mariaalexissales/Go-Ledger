package ops

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func serveGuarded(t *testing.T, limit int, handler http.HandlerFunc) (*httptest.ResponseRecorder, SecurityEvent) {
	t.Helper()

	limiter := NewRateLimiter(limit, time.Minute, time.Minute)
	t.Cleanup(limiter.Close)

	recorder := NewRecorder(nil, nil)
	guard := NewSecurityGuard(limiter, recorder, NewResolver(ModeRemoteAddr, ""))

	rec := httptest.NewRecorder()
	guard.SecurityLogger(handler).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/accounts/7", nil))

	select {
	case event := <-recorder.queue:
		return rec, event
	default:
		t.Fatal("no event was recorded")
		return nil, SecurityEvent{}
	}
}

func assertStatus(t *testing.T, event SecurityEvent, want int) {
	t.Helper()
	if event.StatusCode == nil {
		t.Fatalf("recorded status = nil, want %d", want)
	}
	if *event.StatusCode != want {
		t.Errorf("recorded status = %d, want %d", *event.StatusCode, want)
	}
}

func TestSecurityLoggerRecordsA404(t *testing.T) {
	_, event := serveGuarded(t, 10, func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	assertStatus(t, event, http.StatusNotFound)
	if event.FlagStatus != FlagAllowed {
		t.Errorf("recorded flag = %q, want %q", event.FlagStatus, FlagAllowed)
	}
}

func TestSecurityLoggerRecordsAnImplicit200(t *testing.T) {
	_, event := serveGuarded(t, 10, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	assertStatus(t, event, http.StatusOK)
}

func TestSecurityLoggerRecordsAnEmptyResponseAs200(t *testing.T) {
	_, event := serveGuarded(t, 10, func(w http.ResponseWriter, r *http.Request) {})

	assertStatus(t, event, http.StatusOK)
}

func TestSecurityLoggerRecordsOnlyTheFirstWriteHeader(t *testing.T) {
	_, event := serveGuarded(t, 10, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		w.WriteHeader(http.StatusInternalServerError)
	})

	assertStatus(t, event, http.StatusNoContent)
}

func TestSecurityLoggerRecordsABlockedRequestAs429(t *testing.T) {
	rec, event := serveGuarded(t, 0, func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler ran for a blocked request")
	})

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("response status = %d, want 429", rec.Code)
	}
	assertStatus(t, event, http.StatusTooManyRequests)
	if event.FlagStatus != FlagBlocked {
		t.Errorf("recorded flag = %q, want %q", event.FlagStatus, FlagBlocked)
	}
}

func TestSecurityLoggerRecordsAPanicAs500(t *testing.T) {
	limiter := NewRateLimiter(10, time.Minute, time.Minute)
	t.Cleanup(limiter.Close)

	recorder := NewRecorder(nil, nil)
	guard := NewSecurityGuard(limiter, recorder, NewResolver(ModeRemoteAddr, ""))
	h := guard.SecurityLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))

	func() {
		defer func() {
			if recover() == nil {
				t.Error("the panic was swallowed; it must reach Recoverer")
			}
		}()
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/accounts", nil))
	}()

	select {
	case event := <-recorder.queue:
		assertStatus(t, event, http.StatusInternalServerError)
	default:
		t.Fatal("no event was recorded for the panicking request")
	}
}

func TestStatusWriterFlushesThroughToTheRealWriter(t *testing.T) {
	inner := httptest.NewRecorder()

	if err := http.NewResponseController(&statusWriter{ResponseWriter: inner}).Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if !inner.Flushed {
		t.Error("Flush did not reach the underlying writer")
	}
}
