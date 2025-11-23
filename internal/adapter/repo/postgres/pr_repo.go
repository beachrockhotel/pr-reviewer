package postgres

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/beachrockhotel/pr-reviewer/internal/domain"
)

type PRRepo struct{ pool *pgxpool.Pool }

func NewPRRepo(pool *pgxpool.Pool) *PRRepo { return &PRRepo{pool: pool} }

func (r *PRRepo) CreatePRWithReviewers(ctx context.Context, pr domain.PullRequest, reviewers []string) (domain.PullRequest, error) {
	var exists bool
	if err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM pull_requests WHERE pull_request_id=$1)`, pr.ID,
	).Scan(&exists); err != nil {
		return domain.PullRequest{}, err
	}
	if exists {
		return domain.PullRequest{}, domain.ErrPRExists
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.PullRequest{}, err
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			log.Printf("postgres: rollback failed in CreatePRWithReviewers: %v", err)
		}
	}()

	_, err = tx.Exec(ctx, `
	  INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status)
	  VALUES ($1,$2,$3,'OPEN')`,
		pr.ID, pr.Name, pr.AuthorID)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.PullRequest{}, domain.ErrPRExists
		}
		return domain.PullRequest{}, err
	}

	for _, rid := range reviewers {
		if _, err := tx.Exec(ctx,
			`INSERT INTO pr_reviewers (pull_request_id, reviewer_id) VALUES ($1,$2)`,
			pr.ID, rid,
		); err != nil {
			return domain.PullRequest{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.PullRequest{}, err
	}
	return r.getByID(ctx, pr.ID)
}

func (r *PRRepo) GetByIDForUpdate(ctx context.Context, id string) (domain.PullRequest, error) {
	return r.getByID(ctx, id)
}

func (r *PRRepo) getByID(ctx context.Context, id string) (domain.PullRequest, error) {
	var out domain.PullRequest
	err := r.pool.QueryRow(ctx, `
		SELECT pull_request_id, pull_request_name, author_id, status, created_at, merged_at
		FROM pull_requests WHERE pull_request_id=$1`, id).
		Scan(&out.ID, &out.Name, &out.AuthorID, &out.Status, &out.CreatedAt, &out.MergedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.PullRequest{}, domain.ErrNotFound
		}
		return domain.PullRequest{}, err
	}
	revs, err := r.GetAssignedReviewers(ctx, id)
	if err != nil {
		return domain.PullRequest{}, err
	}
	out.AssignedReviewers = revs
	return out, nil
}

func (r *PRRepo) GetAssignedReviewers(ctx context.Context, prID string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT reviewer_id FROM pr_reviewers WHERE pull_request_id=$1`, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (r *PRRepo) ReplaceReviewer(ctx context.Context, prID, oldID, newID string) (domain.PullRequest, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.PullRequest{}, err
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			log.Printf("postgres: rollback failed in ReplaceReviewer: %v", err)
		}
	}()

	if _, err := tx.Exec(ctx, `DELETE FROM pr_reviewers WHERE pull_request_id=$1 AND reviewer_id=$2`, prID, oldID); err != nil {
		return domain.PullRequest{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO pr_reviewers (pull_request_id, reviewer_id) VALUES ($1,$2)`, prID, newID); err != nil {
		return domain.PullRequest{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.PullRequest{}, err
	}
	return r.getByID(ctx, prID)
}

func (r *PRRepo) SetMerged(ctx context.Context, prID string) (domain.PullRequest, error) {
	_, err := r.pool.Exec(ctx, `
		UPDATE pull_requests
		SET status='MERGED', merged_at = COALESCE(merged_at, $2)
		WHERE pull_request_id=$1`, prID, time.Now().UTC())
	if err != nil {
		return domain.PullRequest{}, err
	}
	return r.getByID(ctx, prID)
}

func (r *PRRepo) ListByReviewer(ctx context.Context, reviewerID string) ([]domain.PullRequestShort, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status
		FROM pull_requests pr
		JOIN pr_reviewers r ON r.pull_request_id = pr.pull_request_id
		WHERE r.reviewer_id = $1
		ORDER BY pr.created_at DESC`, reviewerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.PullRequestShort
	for rows.Next() {
		var s domain.PullRequestShort
		if err := rows.Scan(&s.ID, &s.Name, &s.AuthorID, &s.Status); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *PRRepo) StatsByStatus(ctx context.Context) (map[domain.PRStatus]int, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT status, COUNT(*)
		FROM pull_requests
		GROUP BY status`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	res := map[domain.PRStatus]int{
		domain.StatusOpen:   0,
		domain.StatusMerged: 0,
	}

	for rows.Next() {
		var status domain.PRStatus
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		res[status] = count
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return res, nil
}
