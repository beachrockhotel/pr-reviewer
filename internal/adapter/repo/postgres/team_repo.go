package postgres

import (
	"context"
	"errors"
	"log"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/beachrockhotel/pr-reviewer/internal/domain"
)

type TeamRepo struct {
	pool *pgxpool.Pool
}

func NewTeamRepo(pool *pgxpool.Pool) *TeamRepo {
	return &TeamRepo{pool: pool}
}

func (r *TeamRepo) CreateTeam(ctx context.Context, teamName string) error {
	var exists bool
	if err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM teams WHERE team_name=$1)`, teamName,
	).Scan(&exists); err != nil {
		return err
	}
	if exists {
		return domain.ErrTeamExists
	}

	_, err := r.pool.Exec(ctx, `INSERT INTO teams (team_name) VALUES ($1)`, teamName)
	if isUniqueViolation(err) {
		return domain.ErrTeamExists
	}
	return err
}

func (r *TeamRepo) GetTeamWithMembers(ctx context.Context, teamName string) (domain.Team, []domain.User, error) {
	var exists bool
	if err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM teams WHERE team_name=$1)`, teamName,
	).Scan(&exists); err != nil {
		return domain.Team{}, nil, err
	}
	if !exists {
		return domain.Team{}, nil, domain.ErrNotFound
	}

	rows, err := r.pool.Query(ctx, `
		SELECT user_id, username, is_active
		FROM users
		WHERE team_name = $1
		ORDER BY user_id`, teamName)
	if err != nil {
		return domain.Team{}, nil, err
	}
	defer rows.Close()

	var members []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.UserID, &u.Username, &u.IsActive); err != nil {
			return domain.Team{}, nil, err
		}
		u.TeamName = teamName
		members = append(members, u)
	}
	if err := rows.Err(); err != nil {
		return domain.Team{}, nil, err
	}

	return domain.Team{TeamName: teamName, Members: nil}, members, nil
}

func (r *TeamRepo) UpsertUsersToTeam(ctx context.Context, teamName string, users []domain.User) error {
	if len(users) == 0 {
		return nil
	}

	b := &pgx.Batch{}
	for _, u := range users {
		b.Queue(`
			INSERT INTO users (user_id, username, team_name, is_active)
			VALUES ($1,$2,$3,$4)
			ON CONFLICT (user_id) DO UPDATE
			  SET username=EXCLUDED.username,
			      team_name=EXCLUDED.team_name,
			      is_active=EXCLUDED.is_active,
			      updated_at=now()
		`, u.UserID, u.Username, teamName, u.IsActive)
	}

	br := r.pool.SendBatch(ctx, b)
	defer func() {
		if err := br.Close(); err != nil {
			log.Printf("postgres: batch close failed in UpsertUsersToTeam: %v", err)
		}
	}()

	for range users {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}

	return nil
}

func isUniqueViolation(err error) bool {
	var pgerr *pgconn.PgError
	if errors.As(err, &pgerr) && pgerr.Code == "23505" {
		return true
	}
	return false
}
