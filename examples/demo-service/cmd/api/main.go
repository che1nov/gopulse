package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/che1nov/gopulse/examples/demo-service/internal/repository"
	"github.com/che1nov/gopulse/examples/demo-service/internal/service"
	userhttp "github.com/che1nov/gopulse/examples/demo-service/internal/transport/http"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	repo := repository.NewMemoryUserRepository()
	userService := service.NewUserService(repo, logger)
	handler := userhttp.NewUserHandler(userService, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /users", handler.CreateUser)

	logger.Info("service started", "addr", ":8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		logger.Error("service stopped", "err", err)
		os.Exit(1)
	}
}
