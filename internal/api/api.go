package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/henvic/vio"
)

// cacheControl policy for geolocation.
var cacheControl = "max-age=3600, public"

// APIError response.
type APIError struct {
	HTTPCode int    `json:"http_code"`
	Message  string `json:"message"`
}

func (e APIError) Error() string {
	return e.Message
}

// lookupHandler handles the geolocation request to /v1/lookup.
func (s *Server) lookupHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Cache-control", cacheControl)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "\t")

	ip := r.URL.Query().Get("ip")
	if ip == "" {
		enc.Encode(APIError{
			HTTPCode: http.StatusBadRequest,
			Message:  "missing mandatory IP address query param",
		})
		return
	}

	location, err := s.service.LookupLocation(r.Context(), ip)
	switch {
	case err == context.Canceled, err == context.DeadlineExceeded:
		return
	case err == vio.ErrBadIPAddressFormat:
		w.WriteHeader(http.StatusBadRequest)
		enc.Encode(APIError{
			HTTPCode: http.StatusBadRequest,
			Message:  err.Error(),
		})
	case err != nil:
		w.WriteHeader(http.StatusInternalServerError)
		enc.Encode(APIError{
			HTTPCode: http.StatusInternalServerError,
			Message:  http.StatusText(http.StatusInternalServerError),
		})
		s.log.LogAttrs(r.Context(), slog.LevelError, "internal server error getting location", slog.Any("error", err))
	case location == nil:
		w.WriteHeader(http.StatusNotFound)
		enc.Encode(APIError{
			HTTPCode: http.StatusNotFound,
			Message:  "no location found for the given IP address",
		})
		return
	default:
		enc.Encode(location)
	}
}
