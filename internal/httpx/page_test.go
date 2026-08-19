package httpx

import (
	"net/http/httptest"
	"testing"
)

// The ledger endpoints' real values. ParsePage takes them as arguments, so the
// cases below read the same way they did when this lived in internal/api.
const (
	testFallback = 25
	testMax      = 100
)

func TestParsePage(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantLimit  int
		wantOffset int
	}{
		{
			name:      "no query uses the defaults",
			query:     "",
			wantLimit: testFallback,
		},
		{
			name:      "an explicit limit is honored",
			query:     "?limit=10",
			wantLimit: 10,
		},
		{
			name:      "a limit above the cap is clamped rather than rejected",
			query:     "?limit=5000",
			wantLimit: testMax,
		},
		{
			name:      "exactly the cap is allowed",
			query:     "?limit=100",
			wantLimit: testMax,
		},
		{
			name:      "a non-numeric limit falls back to the default",
			query:     "?limit=abc",
			wantLimit: testFallback,
		},
		{
			name:      "zero is not a useful page size, so it falls back",
			query:     "?limit=0",
			wantLimit: testFallback,
		},
		{
			name:      "a negative limit falls back",
			query:     "?limit=-5",
			wantLimit: testFallback,
		},
		{
			name:       "offset is read independently of limit",
			query:      "?offset=40",
			wantLimit:  testFallback,
			wantOffset: 40,
		},
		{
			name:       "a negative offset clamps to the first page",
			query:      "?offset=-1",
			wantLimit:  testFallback,
			wantOffset: 0,
		},
		{
			name:       "a non-numeric offset clamps to the first page",
			query:      "?offset=later",
			wantLimit:  testFallback,
			wantOffset: 0,
		},
		{
			name:       "both together",
			query:      "?limit=50&offset=100",
			wantLimit:  50,
			wantOffset: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParsePage(httptest.NewRequest("GET", "/accounts"+tt.query, nil), testFallback, testMax)

			if got.Limit != tt.wantLimit {
				t.Errorf("Limit = %d, want %d", got.Limit, tt.wantLimit)
			}
			if got.Offset != tt.wantOffset {
				t.Errorf("Offset = %d, want %d", got.Offset, tt.wantOffset)
			}
		})
	}
}
