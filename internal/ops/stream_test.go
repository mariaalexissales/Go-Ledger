package ops

import (
	"net/http/httptest"
	"testing"
)

func TestParseEventFilterAndMatches(t *testing.T) {
	allowed := SecurityEvent{IPAddress: "192.0.2.1", FlagStatus: FlagAllowed}
	blocked := SecurityEvent{IPAddress: "203.0.113.9", FlagStatus: FlagBlocked}

	tests := []struct {
		name        string
		query       string
		wantAllowed bool
		wantBlocked bool
	}{
		{
			name:        "no filter matches everything",
			query:       "",
			wantAllowed: true,
			wantBlocked: true,
		},
		{
			name:        "flag_status selects one verdict",
			query:       "?flag_status=BLOCKED",
			wantBlocked: true,
		},
		{
			name:        "flag_status is compared exactly, not case-folded",
			query:       "?flag_status=blocked",
			wantAllowed: false,
			wantBlocked: false,
		},
		{
			name:        "a single ip_address selects that host",
			query:       "?ip_address=203.0.113.9",
			wantBlocked: true,
		},
		{
			name:        "a comma-separated list selects several",
			query:       "?ip_address=192.0.2.1,203.0.113.9",
			wantAllowed: true,
			wantBlocked: true,
		},
		{
			name:        "surrounding whitespace in the list is trimmed",
			query:       "?ip_address=%20192.0.2.1%20,%20203.0.113.9%20",
			wantAllowed: true,
			wantBlocked: true,
		},
		{
			name:        "an empty ip_address is not a filter",
			query:       "?ip_address=",
			wantAllowed: true,
			wantBlocked: true,
		},
		{
			name:        "a list of only separators is not a filter",
			query:       "?ip_address=,,",
			wantAllowed: true,
			wantBlocked: true,
		},
		{
			name:  "an unmatched ip excludes everything",
			query: "?ip_address=198.51.100.1",
		},
		{
			name:        "the two filters combine as AND",
			query:       "?flag_status=BLOCKED&ip_address=192.0.2.1",
			wantAllowed: false,
			wantBlocked: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := parseEventFilter(httptest.NewRequest("GET", "/ops/events/stream"+tt.query, nil))

			if got := f.matches(allowed); got != tt.wantAllowed {
				t.Errorf("matches(allowed event) = %v, want %v", got, tt.wantAllowed)
			}
			if got := f.matches(blocked); got != tt.wantBlocked {
				t.Errorf("matches(blocked event) = %v, want %v", got, tt.wantBlocked)
			}
		})
	}
}

func TestLastEventID(t *testing.T) {
	tests := []struct {
		name   string
		header string
		query  string
		want   int64
	}{
		{
			name: "nothing supplied starts from the beginning",
			want: 0,
		},
		{
			name:   "the Last-Event-ID header is used",
			header: "42",
			want:   42,
		},
		{
			name:  "since_id is the fallback when the header is absent",
			query: "?since_id=17",
			want:  17,
		},
		{
			name:   "the header wins over since_id",
			header: "42",
			query:  "?since_id=17",
			want:   42,
		},
		{
			name:   "a non-numeric header degrades to a full replay",
			header: "not-an-id",
			want:   0,
		},
		{
			name:   "a negative id degrades to a full replay",
			header: "-5",
			want:   0,
		},
		{
			name:   "an overflowing id degrades to a full replay",
			header: "99999999999999999999999",
			want:   0,
		},
		{
			name:   "zero is passed through",
			header: "0",
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/ops/events/stream"+tt.query, nil)
			if tt.header != "" {
				r.Header.Set("Last-Event-ID", tt.header)
			}

			if got := lastEventID(r); got != tt.want {
				t.Errorf("lastEventID() = %d, want %d", got, tt.want)
			}
		})
	}
}
