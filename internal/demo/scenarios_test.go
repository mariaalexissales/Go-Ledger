package demo

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"sync"
	"testing"

	"go-ledger/internal/ops"
)

func fakeLedger(t *testing.T, accounts, limit int) *httptest.Server {
	t.Helper()

	var (
		mu   sync.Mutex
		seen int
	)
	accountPath := regexp.MustCompile(`^/api/accounts/(\d+)$`)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen++
		over := seen > limit
		mu.Unlock()

		if over {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}

		if r.URL.Path == "/api/accounts" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(w, `{"data":[{"id":1}],"total":%d,"limit":1,"offset":0}`, accounts)
			return
		}

		m := accountPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			t.Errorf("unexpected request %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		if id, _ := strconv.Atoi(m[1]); id > accounts {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestEnumerationReachesMissingIDsBeforeTheLimit(t *testing.T) {
	const accounts, limit = 50, 30

	c := newClient(fakeLedger(t, accounts, limit).URL, "", ops.Policy{Limit: limit})
	if err := runEnumeration(context.Background(), c); err != nil {
		t.Fatal(err)
	}

	var scan []Step
	for _, s := range c.Steps() {
		if accountPathID(s.Path) > 0 {
			scan = append(scan, s)
		}
	}
	if len(scan) == 0 {
		t.Fatal("no account IDs were scanned")
	}

	for i := 1; i < len(scan); i++ {
		if accountPathID(scan[i].Path) != accountPathID(scan[i-1].Path)+1 {
			t.Fatalf("scan is not sequential at step %d: %s after %s", i, scan[i].Path, scan[i-1].Path)
		}
	}

	phase := map[int]int{http.StatusOK: 0, http.StatusNotFound: 1, http.StatusTooManyRequests: 2}
	counts := map[int]int{}
	last := 0
	for _, s := range scan {
		p, ok := phase[s.Status]
		if !ok {
			t.Fatalf("%s got %d", s.Path, s.Status)
		}
		if p < last {
			t.Fatalf("%s got %d after a later phase had started", s.Path, s.Status)
		}
		last = p
		counts[s.Status]++
	}
	for _, status := range []int{http.StatusOK, http.StatusNotFound, http.StatusTooManyRequests} {
		if counts[status] == 0 {
			t.Errorf("no %d in the scan (counts: %v)", status, counts)
		}
	}
}

func accountPathID(path string) int {
	m := regexp.MustCompile(`^/api/accounts/(\d+)$`).FindStringSubmatch(path)
	if m == nil {
		return 0
	}
	id, _ := strconv.Atoi(m[1])
	return id
}
