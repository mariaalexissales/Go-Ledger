package ops

import (
	"encoding/json"
	"net/http"

	"go-ledger/internal/httpx"
)

// decodeJSON reads a JSON body, writing a 400 and reporting false on failure.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}
