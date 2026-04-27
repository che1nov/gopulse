package service

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"log/slog"
	"strconv"
	"time"

	"github.com/che1nov/gopulse/examples/demo-service/internal/domain"
)

type UserRepository interface {
	Save(ctx context.Context, user domain.User) error
	Count(ctx context.Context) int
}

type UserService struct {
	repo   UserRepository
	logger *slog.Logger
}

type CreateUserInput struct {
	Name  string
	Email string
}

func NewUserService(repo UserRepository, logger *slog.Logger) UserService {
	return UserService{repo: repo, logger: logger}
}

func (s UserService) CreateUser(ctx context.Context, input CreateUserInput) (domain.User, error) {
	s.logger.InfoContext(ctx, "create user", "operation", "create_user")

	now := time.Now()
	user, err := domain.NewUser(makeUserID(input.Email, now), input.Name, input.Email, now)
	if err != nil {
		s.logger.WarnContext(ctx, "invalid user", "err", err, "operation", "create_user")
		return domain.User{}, err
	}

	if err := s.repo.Save(ctx, user); err != nil {
		s.logger.ErrorContext(ctx, "save user failed", "err", err, "operation", "create_user")
		return domain.User{}, err
	}
	return user, nil
}

func (s UserService) Count(ctx context.Context) int {
	return s.repo.Count(ctx)
}

func makeUserID(email string, now time.Time) string {
	sum := sha1.Sum([]byte(email + ":" + strconv.FormatInt(now.UnixNano(), 10)))
	return hex.EncodeToString(sum[:8])
}
