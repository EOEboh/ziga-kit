package respond

import (
	"encoding/json"
	"net/http"
)

// JSON serialises v as JSON and writes it with the given status code.
// It always sets Content-Type: application/json.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Error writes a standardised error envelope.
//
//	{ "error": "message here" }
func Error(w http.ResponseWriter, status int, message string) {
	JSON(w, status, map[string]string{"error": message})
}

// NoContent writes a 204 with no body.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}
