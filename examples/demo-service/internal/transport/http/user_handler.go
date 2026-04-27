package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/che1nov/gopulse/examples/demo-service/internal/domain"
	"github.com/che1nov/gopulse/examples/demo-service/internal/service"
)

type UserHandler struct {
	service service.UserService
	logger  *slog.Logger
}

type createUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type userResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func NewUserHandler(service service.UserService, logger *slog.Logger) UserHandler {
	return UserHandler{service: service, logger: logger}
}

func (h UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WarnContext(r.Context(), "decode request failed", "err", err, "operation", "create_user")
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	user, err := h.service.CreateUser(r.Context(), service.CreateUserInput{
		Name:  req.Name,
		Email: req.Email,
	})
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, domain.ErrInvalidUser) {
			status = http.StatusBadRequest
		}
		http.Error(w, err.Error(), status)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(userResponse{
		ID:    user.ID,
		Name:  user.Name,
		Email: user.Email,
	})
}
