package httpx

import (
	"encoding/json"
	"errors"
	"net/http"
)

const maxBodyBytes = 1 << 20

func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
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
