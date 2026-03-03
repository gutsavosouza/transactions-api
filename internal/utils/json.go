package utils

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

func RespondWithError(w http.ResponseWriter, statusCode int, message string) {
	if statusCode > 499 {
		slog.Error("5XX error: ", "error", message)
	}

	type errorResponse struct {
		Error string `json:"error"`
	}

	RespondWithJSON(w, statusCode, errorResponse{Error: message})
}

func RespondWithJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		slog.Error("Failed to marshal JSON. Data: %v", "error", payload)
		w.WriteHeader(500)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(data)
}

func ReadFromBody(r *http.Request, data any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	return decoder.Decode(data)
}
