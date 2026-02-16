package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRespondWithGrpcError(t *testing.T) {
	tests := []struct {
		name           string
		grpcErr        error
		expectedStatus int
	}{
		{
			name:           "NotFound",
			grpcErr:        status.Error(codes.NotFound, "missing item"),
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Unavailable",
			grpcErr:        status.Error(codes.Unavailable, "server down"),
			expectedStatus: http.StatusServiceUnavailable,
		},
		{
			name:           "PermissionDenied",
			grpcErr:        status.Error(codes.PermissionDenied, "no access"),
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Unauthenticated",
			grpcErr:        status.Error(codes.Unauthenticated, "login required"),
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "InvalidArgument",
			grpcErr:        status.Error(codes.InvalidArgument, "bad ID"),
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "DeadlineExceeded",
			grpcErr:        status.Error(codes.DeadlineExceeded, "too slow"),
			expectedStatus: http.StatusGatewayTimeout,
		},
		{
			name:           "Unknown/Internal",
			grpcErr:        status.Error(codes.Unknown, "unknown error"),
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name:           "Non-gRPC Error",
			grpcErr:        errors.New("standard go error"),
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			RespondWithGrpcError(w, tt.grpcErr, "context message")

			if w.Code != tt.expectedStatus {
				t.Errorf("RespondWithGrpcError() status = %v, want %v", w.Code, tt.expectedStatus)
			}
		})
	}
}
