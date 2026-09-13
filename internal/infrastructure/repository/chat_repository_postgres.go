package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/sourabhmandal/gosling/internal/domain/entity"
	domainRepo "github.com/sourabhmandal/gosling/internal/domain/repository"
	"github.com/sourabhmandal/gosling/internal/infrastructure/database"
)

type chatRepositoryPostgres struct {
	db *database.PostgresDB
}

// NewChatRepositoryPostgres creates a new PostgreSQL-backed ChatRepository.
func NewChatRepositoryPostgres(db *database.PostgresDB) domainRepo.ChatRepository {
	return &chatRepositoryPostgres{db: db}
}

func (r *chatRepositoryPostgres) Save(ctx context.Context, history *entity.ChatHistory) error {
	query := `
		INSERT INTO chat_histories (id, user_id, model, prompt, response, prompt_tokens, output_tokens, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.DB.ExecContext(
		ctx,
		query,
		history.ID,
		history.UserID,
		history.Model,
		history.Prompt,
		history.Response,
		history.PromptTokens,
		history.OutputTokens,
		history.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save chat history: %w", err)
	}
	return nil
}

func (r *chatRepositoryPostgres) GetHistoryByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*entity.ChatHistory, error) {
	query := `
		SELECT id, user_id, model, prompt, response, prompt_tokens, output_tokens, created_at
		FROM chat_histories
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	rows, err := r.db.DB.QueryContext(ctx, query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query chat history: %w", err)
	}
	defer rows.Close()

	var histories []*entity.ChatHistory
	for rows.Next() {
		h := &entity.ChatHistory{}
		if err := rows.Scan(
			&h.ID,
			&h.UserID,
			&h.Model,
			&h.Prompt,
			&h.Response,
			&h.PromptTokens,
			&h.OutputTokens,
			&h.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan chat history: %w", err)
		}
		histories = append(histories, h)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading chat history rows: %w", err)
	}

	return histories, nil
}
