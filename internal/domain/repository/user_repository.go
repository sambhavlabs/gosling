package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/sourabhmandal/gosling/internal/domain/entity"
)

// UserRepository defines the persistence port for User domain entities.
type UserRepository interface {
	Create(ctx context.Context, user *entity.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
	GetByEmail(ctx context.Context, email string) (*entity.User, error)
	GetByGoogleID(ctx context.Context, googleID string) (*entity.User, error)
	Update(ctx context.Context, user *entity.User) error
}
