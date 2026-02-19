package handlers

import (
	"log/slog"
	"net/http"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RespondWithGrpcError maps a gRPC error to a standard HTTP response.
func RespondWithGrpcError(w http.ResponseWriter, err error, message string) {
	st, ok := status.FromError(err)
	if !ok {
		// This is not a gRPC error (e.g. context cancellation from net/http)
		slog.Error("internal_error", "error", err, "context", message)
		http.Error(w, message, http.StatusInternalServerError)
		return
	}

	slog.Error("grpc_error", 
		"code", st.Code().String(), 
		"message", st.Message(), 
		"context", message,
	)

	var statusCode int
	var publicMessage string

	switch st.Code() {
	case codes.InvalidArgument:
		statusCode = http.StatusBadRequest
		publicMessage = "Invalid request arguments"
	case codes.NotFound:
		statusCode = http.StatusNotFound
		publicMessage = "Resource not found"
	case codes.Unauthenticated:
		statusCode = http.StatusUnauthorized
		publicMessage = "Authentication required"
	case codes.PermissionDenied:
		statusCode = http.StatusForbidden
		publicMessage = "Permission denied"
	case codes.Unavailable:
		statusCode = http.StatusServiceUnavailable
		publicMessage = "Internal service temporarily unavailable"
	case codes.DeadlineExceeded:
		statusCode = http.StatusGatewayTimeout
		publicMessage = "Service request timed out"
	default:
		statusCode = http.StatusInternalServerError
		publicMessage = "Internal server error"
	}

	// Always hide the raw gRPC details from the end user
	http.Error(w, publicMessage, statusCode)
}
