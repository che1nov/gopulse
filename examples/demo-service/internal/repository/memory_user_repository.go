package repository

import (
	"context"
	"sync"

	"github.com/che1nov/gopulse/examples/demo-service/internal/domain"
)

type MemoryUserRepository struct {
	mu    sync.RWMutex
	users map[string]domain.User
}

func NewMemoryUserRepository() *MemoryUserRepository {
	return &MemoryUserRepository{users: make(map[string]domain.User)}
}

func (r *MemoryUserRepository) Save(_ context.Context, user domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.users[user.ID] = user
	return nil
}

func (r *MemoryUserRepository) Count(_ context.Context) int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.users)
}
