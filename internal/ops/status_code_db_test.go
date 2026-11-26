package ops

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"go-ledger/internal/db"
	"go-ledger/internal/httpx"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// testPool truncates security_events: point GO_LEDGER_TEST_DATABASE_URL at a throwaway database.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()

	url := os.Getenv("GO_LEDGER_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("GO_LEDGER_TEST_DATABASE_URL is not set")
	}

	if err := db.RunMigrations(url); err != nil {
		t.Fatalf("migrations: %v", err)
	}

	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	if _, err := pool.Exec(context.Background(), "TRUNCATE security_events RESTART IDENTITY"); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return pool
}

func recordAndWait(t *testing.T, pool *pgxpool.Pool, statuses ...int) []SecurityEvent {
	t.Helper()

	hub := NewHub()
	sub := hub.Subscribe()
	defer sub.Close()

	recorder := NewRecorder(pool, hub)
	ctx, cancel := context.WithCancel(context.Background())
	go recorder.Run(ctx)
	t.Cleanup(func() { cancel(); <-recorder.Done() })

	for i, status := range statuses {
		recorder.Record("203.0.113.66", "GET /api/accounts/"+string(rune('1'+i)), FlagAllowed, status, time.Now())
	}

	published := make([]SecurityEvent, 0, len(statuses))
	timeout := time.After(5 * time.Second)
	for len(published) < len(statuses) {
		select {
		case event := <-sub.Events():
			published = append(published, event)
		case <-timeout:
			t.Fatalf("published %d of %d events (failed inserts: %d)", len(published), len(statuses), recorder.Failed())
		}
	}
	return published
}

func listEvents(t *testing.T, pool *pgxpool.Pool, query string) []EventDTO {
	t.Helper()

	r := chi.NewRouter()
	NewConsole(pool, NewHub(), nil, false).Mount(r)

	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/events"+query, nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /events%s = %d: %s", query, rec.Code, rec.Body)
	}

	var page httpx.ListResponse[EventDTO]
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	return page.Data
}

func TestRecorderStoresAndPublishesStatusCode(t *testing.T) {
	pool := testPool(t)

	published := recordAndWait(t, pool, http.StatusOK, http.StatusNotFound)

	for i, want := range []int{http.StatusOK, http.StatusNotFound} {
		assertStatus(t, published[i], want)
	}
}

func TestListEventsFiltersByStatusCode(t *testing.T) {
	pool := testPool(t)
	recordAndWait(t, pool, http.StatusOK, http.StatusNotFound, http.StatusNotFound)

	got := listEvents(t, pool, "?status_code=404")

	if len(got) != 2 {
		t.Fatalf("got %d events, want the 2 404s", len(got))
	}
	for _, e := range got {
		if e.StatusCode == nil || *e.StatusCode != http.StatusNotFound {
			t.Errorf("event %d status = %v, want 404", e.ID, e.StatusCode)
		}
	}
}

func TestListEventsReturnsNullStatusForOldRows(t *testing.T) {
	pool := testPool(t)

	if _, err := pool.Exec(context.Background(), `
		INSERT INTO security_events (ip_address, action_type, flag_status) VALUES ('192.0.2.1', 'GET /api/accounts', 'ALLOWED')
	`); err != nil {
		t.Fatal(err)
	}

	got := listEvents(t, pool, "")

	if len(got) != 1 {
		t.Fatalf("got %d events, want 1", len(got))
	}
	if got[0].StatusCode != nil {
		t.Errorf("status = %d, want nil", *got[0].StatusCode)
	}
}

func TestStreamReplayIncludesStatusCode(t *testing.T) {
	pool := testPool(t)
	recordAndWait(t, pool, http.StatusNotFound)

	c := NewConsole(pool, NewHub(), nil, false)
	rec := httptest.NewRecorder()
	c.replay(rec, http.NewResponseController(rec), httptest.NewRequest(http.MethodGet, "/events/stream", nil), 0, eventFilter{})

	if !strings.Contains(rec.Body.String(), `"status_code":404`) {
		t.Errorf("replayed stream = %q, want it to carry status_code 404", rec.Body.String())
	}
}
