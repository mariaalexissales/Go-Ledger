package api

import (
	"net/http/httptest"
	"testing"
)

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
