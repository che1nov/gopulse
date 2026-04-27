package http

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/che1nov/gopulse/examples/demo-service/internal/repository"
	"github.com/che1nov/gopulse/examples/demo-service/internal/service"
)

func BenchmarkCreateUserHandler(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := service.NewUserService(repository.NewMemoryUserRepository(), logger)
	handler := NewUserHandler(svc, logger)
	body := []byte(`{"name":"Alan Turing","email":"alan@example.com"}`)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
		rec := httptest.NewRecorder()

		handler.CreateUser(rec, req)
		if rec.Code != http.StatusCreated {
			b.Fatalf("status = %d", rec.Code)
		}
	}
}
