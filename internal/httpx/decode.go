package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
)

// maxBodyBytes caps how much of a request body will be read. Every write
// endpoint here takes a small flat object, so a megabyte is orders of magnitude
// more than any legitimate request needs. Without a cap, json.Decoder reads
// until the client stops sending, which is a trivial way to make the server
// allocate without limit.
const maxBodyBytes = 1 << 20

// DecodeJSON reads a JSON request body into dst. On failure it writes the error
// response itself and reports false, so handlers can `if !DecodeJSON(...) {
// return }` rather than plumbing an error back.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		// MaxBytesReader surfaces its own error type, and reporting it as a
		// malformed body would send the client looking for a syntax error.
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			WriteError(w, http.StatusRequestEntityTooLarge, "request body too large")
			return false
		}

		WriteError(w, http.StatusBadRequest, "invalid request body")
		return false
	}

	return true
}
