package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/beachrockhotel/pr-reviewer/internal/domain"
)

type UserRepo struct{ pool *pgxpool.Pool }

func NewUserRepo(pool *pgxpool.Pool) *UserRepo { return &UserRepo{pool: pool} }

func (r *UserRepo) GetByID(ctx context.Context, id string) (domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx, `
		SELECT user_id, username, team_name, is_active
		FROM users WHERE user_id=$1`, id).Scan(&u.UserID, &u.Username, &u.TeamName, &u.IsActive)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, domain.ErrNotFound
		}
		return domain.User{}, err
	}
	return u, nil
}

func (r *UserRepo) SetActive(ctx context.Context, id string, active bool) (domain.User, error) {
	ct, err := r.pool.Exec(ctx, `UPDATE users SET is_active=$2, updated_at=now() WHERE user_id=$1`, id, active)
	if err != nil {
		return domain.User{}, err
	}
	if ct.RowsAffected() == 0 {
		return domain.User{}, domain.ErrNotFound
	}
	return r.GetByID(ctx, id)
}

func (r *UserRepo) ListActiveInTeamExcept(ctx context.Context, teamName string, excludeIDs []string, limit int) ([]domain.User, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT user_id, username, team_name, is_active
		FROM users
		WHERE team_name=$1 AND is_active=TRUE
		  AND NOT (user_id = ANY($2))
		ORDER BY random()
		LIMIT $3
	`, teamName, excludeIDs, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.UserID, &u.Username, &u.TeamName, &u.IsActive); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}
