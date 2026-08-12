package httpx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSON(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
	}

	tests := []struct {
		name       string
		body       string
		wantOK     bool
		wantStatus int
		wantError  string
	}{
		{
			name:       "valid object",
			body:       `{"name":"checking"}`,
			wantOK:     true,
			wantStatus: http.StatusOK,
		},
		{
			name:       "malformed json",
			body:       `{"name":`,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid request body",
		},
		{
			name:       "empty body",
			body:       ``,
			wantStatus: http.StatusBadRequest,
			wantError:  "invalid request body",
		},
		{
			name:       "body over the cap",
			body:       `{"name":"` + strings.Repeat("x", maxBodyBytes) + `"}`,
			wantStatus: http.StatusRequestEntityTooLarge,
			wantError:  "request body too large",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			w := httptest.NewRecorder()

			var dst payload
			ok := DecodeJSON(w, r, &dst)

			if ok != tt.wantOK {
				t.Fatalf("DecodeJSON() = %v, want %v", ok, tt.wantOK)
			}
			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantError == "" {
				return
			}

			var got struct {
				Error string `json:"error"`
			}
			if err := json.NewDecoder(w.Body).Decode(&got); err != nil {
				t.Fatalf("decoding error envelope: %v", err)
			}
			if got.Error != tt.wantError {
				t.Errorf("error = %q, want %q", got.Error, tt.wantError)
			}
		})
	}
}

func TestDecodeJSONAtCap(t *testing.T) {
	var dst struct {
		Name string `json:"name"`
	}

	body := `{"name":"` + strings.Repeat("x", maxBodyBytes-11) + `"}`
	if len(body) != maxBodyBytes {
		t.Fatalf("test setup: body is %d bytes, want %d", len(body), maxBodyBytes)
	}

	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	w := httptest.NewRecorder()

	if !DecodeJSON(w, r, &dst) {
		t.Fatalf("DecodeJSON() = false for a body at exactly the cap, status %d", w.Code)
	}
}
