// Package httpx holds the JSON response helpers shared by the api and ops layers.
package httpx

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

type errorResponse struct {
	Error string `json:"error"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("httpx: encoding a %d response body failed: %v", status, err)
	}
}

func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, errorResponse{Error: msg})
}

func WriteServerError(w http.ResponseWriter, r *http.Request, msg string, err error) {
	log.Printf("500 %s %s [%s]: %s: %v",
		r.Method, r.URL.Path, middleware.GetReqID(r.Context()), msg, err)

	WriteError(w, http.StatusInternalServerError, msg)
}

type ListResponse[T any] struct {
	Data   []T `json:"data"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

func WriteList[T any](w http.ResponseWriter, items []T, total, limit, offset int) {
	if items == nil {
		items = []T{}
	}

	WriteJSON(w, http.StatusOK, ListResponse[T]{
		Data:   items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}
