package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/sourabhmandal/gosling/internal/domain"
	"github.com/sourabhmandal/gosling/internal/domain/entity"
	domainRepo "github.com/sourabhmandal/gosling/internal/domain/repository"
	"github.com/sourabhmandal/gosling/internal/infrastructure/database"
)

type userRepositoryPostgres struct {
	db *database.PostgresDB
}

// NewUserRepositoryPostgres creates a new PostgreSQL-backed UserRepository.
func NewUserRepositoryPostgres(db *database.PostgresDB) domainRepo.UserRepository {
	return &userRepositoryPostgres{db: db}
}

func (r *userRepositoryPostgres) Create(ctx context.Context, user *entity.User) error {
	query := `
		INSERT INTO users (id, email, name, avatar_url, google_id, created_at, updated_at, last_login_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`
	_, err := r.db.DB.ExecContext(
		ctx,
		query,
		user.ID,
		user.Email,
		user.Name,
		user.AvatarURL,
		user.GoogleID,
		user.CreatedAt,
		user.UpdatedAt,
		user.LastLoginAt,
	)
	if err != nil {
		return fmt.Errorf("failed to insert user into postgres: %w", err)
	}
	return nil
}

func (r *userRepositoryPostgres) GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error) {
	query := `
		SELECT id, email, name, avatar_url, google_id, created_at, updated_at, last_login_at
		FROM users
		WHERE id = $1
	`
	row := r.db.DB.QueryRowContext(ctx, query, id)
	return r.scanUser(row)
}

func (r *userRepositoryPostgres) GetByEmail(ctx context.Context, email string) (*entity.User, error) {
	query := `
		SELECT id, email, name, avatar_url, google_id, created_at, updated_at, last_login_at
		FROM users
		WHERE email = $1
	`
	row := r.db.DB.QueryRowContext(ctx, query, email)
	return r.scanUser(row)
}

func (r *userRepositoryPostgres) GetByGoogleID(ctx context.Context, googleID string) (*entity.User, error) {
	if googleID == "" {
		return nil, domain.ErrUserNotFound
	}
	query := `
		SELECT id, email, name, avatar_url, google_id, created_at, updated_at, last_login_at
		FROM users
		WHERE google_id = $1
	`
	row := r.db.DB.QueryRowContext(ctx, query, googleID)
	return r.scanUser(row)
}

func (r *userRepositoryPostgres) Update(ctx context.Context, user *entity.User) error {
	query := `
		UPDATE users
		SET email = $2, name = $3, avatar_url = $4, google_id = $5, updated_at = $6, last_login_at = $7
		WHERE id = $1
	`
	result, err := r.db.DB.ExecContext(
		ctx,
		query,
		user.ID,
		user.Email,
		user.Name,
		user.AvatarURL,
		user.GoogleID,
		user.UpdatedAt,
		user.LastLoginAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get affected rows: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrUserNotFound
	}

	return nil
}

func (r *userRepositoryPostgres) scanUser(row *sql.Row) (*entity.User, error) {
	user := &entity.User{}
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.Name,
		&user.AvatarURL,
		&user.GoogleID,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.LastLoginAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to scan user row: %w", err)
	}
	return user, nil
}
