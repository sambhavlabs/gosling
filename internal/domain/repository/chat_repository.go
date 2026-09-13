package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/sourabhmandal/gosling/internal/domain/entity"
)

// ChatRepository defines the persistence port for Chat History.
type ChatRepository interface {
	Save(ctx context.Context, history *entity.ChatHistory) error
	GetHistoryByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*entity.ChatHistory, error)
}
