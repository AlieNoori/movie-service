package memory

import (
	"context"
	"sync"

	"movieexample.com/auth/internal/repository"
	"movieexample.com/auth/pkg/model"
)

type Repository struct {
	sync.RWMutex
	userByID    map[string]*model.User
	userByEmail map[string]*model.User
}

func New() *Repository {
	return &Repository{
		userByID:    make(map[string]*model.User),
		userByEmail: make(map[string]*model.User),
	}
}

// GetByID retrieves a user by their unique ID.
func (r *Repository) GetByID(ctx context.Context, id string) (*model.User, error) {
	r.RLock()
	defer r.RUnlock()

	user, ok := r.userByID[id]
	if !ok {
		return nil, repository.ErrNotFound
	}

	return user, nil
}

// GetByEmail retrieves a user by their email address.
func (r *Repository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	r.RLock()
	defer r.RUnlock()

	user, ok := r.userByEmail[email]
	if !ok {
		return nil, repository.ErrNotFound
	}

	return user, nil
}

// Put adds or updates a user.
func (r *Repository) Put(_ context.Context, user *model.User) error {
	r.Lock()
	defer r.Unlock()

	r.userByEmail[user.Email] = user
	r.userByID[user.ID] = user

	return nil
}
