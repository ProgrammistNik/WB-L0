package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status string `json:"status"`
	Time   string `json:"time"`
}

// handleGetOrder handles GET /order/{order_uid} requests
func (s *Server) handleGetOrder(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	orderUID := r.PathValue("order_uid")
	if orderUID == "" {
		s.writeErrorResponse(w, http.StatusBadRequest, "Order UID is required", "")
		return
	}

	orderUID = strings.TrimSpace(orderUID)
	if len(orderUID) == 0 {
		s.writeErrorResponse(w, http.StatusBadRequest, "Order UID cannot be empty", "")
		return
	}

	result, err := s.service.GetOrder(r.Context(), orderUID)

	duration := time.Since(start)

	if err != nil {
		s.logger.Error().
			Err(err).
			Str("order_uid", orderUID).
			Str("remote_addr", r.RemoteAddr).
			Dur("duration", duration).
			Msg("Failed to get order")

		if isNotFoundError(err) {
			s.writeErrorResponse(w, http.StatusNotFound, "Order not found", orderUID)
			return
		}

		s.writeErrorResponse(w, http.StatusInternalServerError, "Internal server error", "")
		return
	}

	order := result
	if order == nil {
		s.writeErrorResponse(w, http.StatusNotFound, "Order not found", orderUID)
		return
	}

	s.writeJSONResponse(w, http.StatusOK, order)
}

// handleHealth handles GET /health requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status: "ok",
		Time:   time.Now().UTC().Format(time.RFC3339),
	}

	s.writeJSONResponse(w, http.StatusOK, response)
}

// writeJSONResponse writes a JSON response
func (s *Server) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		s.logger.Error().Err(err).Msg("Failed to encode JSON response")
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
	}
}

// writeErrorResponse writes an error response in JSON format
func (s *Server) writeErrorResponse(w http.ResponseWriter, statusCode int, message, details string) {
	errorResp := ErrorResponse{
		Error:   message,
		Message: details,
	}

	s.writeJSONResponse(w, statusCode, errorResp)
}

// isNotFoundError checks if an error indicates that a resource was not found
func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())
	return strings.Contains(errMsg, "not found") ||
		strings.Contains(errMsg, "no rows") ||
		errors.Is(err, ErrOrderNotFound)
}

// ErrOrderNotFound is a sentinel error for order not found cases
var ErrOrderNotFound = errors.New("order not found")