package api

import (
	"net/http/httptest"
	"testing"
)

func TestParsePageParams(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantLimit  int
		wantOffset int
	}{
		{
			name:      "no query uses the defaults",
			query:     "",
			wantLimit: defaultLimit,
		},
		{
			name:      "an explicit limit is honored",
			query:     "?limit=10",
			wantLimit: 10,
		},
		{
			name:      "a limit above the cap is clamped rather than rejected",
			query:     "?limit=5000",
			wantLimit: maxLimit,
		},
		{
			name:      "exactly the cap is allowed",
			query:     "?limit=100",
			wantLimit: maxLimit,
		},
		{
			name:      "a non-numeric limit falls back to the default",
			query:     "?limit=abc",
			wantLimit: defaultLimit,
		},
		{
			name:      "zero is not a useful page size, so it falls back",
			query:     "?limit=0",
			wantLimit: defaultLimit,
		},
		{
			name:      "a negative limit falls back",
			query:     "?limit=-5",
			wantLimit: defaultLimit,
		},
		{
			name:       "offset is read independently of limit",
			query:      "?offset=40",
			wantLimit:  defaultLimit,
			wantOffset: 40,
		},
		{
			name:       "a negative offset clamps to the first page",
			query:      "?offset=-1",
			wantLimit:  defaultLimit,
			wantOffset: 0,
		},
		{
			name:       "a non-numeric offset clamps to the first page",
			query:      "?offset=later",
			wantLimit:  defaultLimit,
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
			got := parsePageParams(httptest.NewRequest("GET", "/accounts"+tt.query, nil))

			if got.Limit != tt.wantLimit {
				t.Errorf("Limit = %d, want %d", got.Limit, tt.wantLimit)
			}
			if got.Offset != tt.wantOffset {
				t.Errorf("Offset = %d, want %d", got.Offset, tt.wantOffset)
			}
		})
	}
}

func TestOptionalIntParam(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  *int
	}{
		{
			name:  "absent yields nil",
			query: "",
		},
		{
			name:  "empty yields nil",
			query: "?account_id=",
		},
		{
			name:  "non-numeric yields nil rather than an error",
			query: "?account_id=seven",
		},
		{
			name:  "a number is returned",
			query: "?account_id=7",
			want:  ptr(7),
		},
		{
			name:  "zero is a real value, not an absence",
			query: "?account_id=0",
			want:  ptr(0),
		},
		{
			name:  "negative is passed through for the query to find nothing",
			query: "?account_id=-3",
			want:  ptr(-3),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := optionalIntParam(httptest.NewRequest("GET", "/transactions"+tt.query, nil), "account_id")

			switch {
			case tt.want == nil && got != nil:
				t.Errorf("got %d, want nil", *got)
			case tt.want != nil && got == nil:
				t.Errorf("got nil, want %d", *tt.want)
			case tt.want != nil && *got != *tt.want:
				t.Errorf("got %d, want %d", *got, *tt.want)
			}
		})
	}
}

func ptr(n int) *int { return &n }
