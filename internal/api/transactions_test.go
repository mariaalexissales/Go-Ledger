package api

import (
	"encoding/json"
	"testing"
)

func TestValidateCreateTransaction(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"valid positive amount", `{"account_id":1,"amount":100.50}`, ""},
		{"valid negative amount", `{"account_id":1,"amount":-42}`, ""},
		{"absent amount", `{"account_id":1}`, "amount is required"},
		{"null amount", `{"account_id":1,"amount":null}`, "amount is required"},
		{"NaN amount", `{"account_id":1,"amount":"NaN"}`, "amount must be a number"},
		{"zero amount", `{"account_id":1,"amount":0}`, "amount cannot be zero"},
		{"zero amount with decimals", `{"account_id":1,"amount":0.00}`, "amount cannot be zero"},
		{"absent account_id", `{"amount":10}`, "account_id must be a positive integer"},
		{"zero account_id", `{"account_id":0,"amount":10}`, "account_id must be a positive integer"},
		{"negative account_id", `{"account_id":-3,"amount":10}`, "account_id must be a positive integer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req createTransactionRequest
			if err := json.Unmarshal([]byte(tt.body), &req); err != nil {
				t.Fatalf("decoding %s: %v", tt.body, err)
			}

			msg, ok := validateCreateTransaction(req)

			if tt.want == "" {
				if !ok {
					t.Fatalf("rejected a valid body with %q", msg)
				}
				return
			}

			if ok {
				t.Fatalf("accepted %s, want rejection with %q", tt.body, tt.want)
			}
			if msg != tt.want {
				t.Errorf("message = %q, want %q", msg, tt.want)
			}
		})
	}
}
