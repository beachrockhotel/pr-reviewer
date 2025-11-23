package usecase

import (
	"context"

	"github.com/beachrockhotel/pr-reviewer/internal/domain"
)

type TeamRepo interface {
	CreateTeam(ctx context.Context, teamName string) error
	GetTeamWithMembers(ctx context.Context, teamName string) (domain.Team, []domain.User, error)
	UpsertUsersToTeam(ctx context.Context, teamName string, users []domain.User) error
}

type UserRepo interface {
	GetByID(ctx context.Context, userID string) (domain.User, error)
	SetActive(ctx context.Context, userID string, isActive bool) (domain.User, error)
	ListActiveInTeamExcept(ctx context.Context, teamName string, excludeIDs []string, limit int) ([]domain.User, error)
}

type PRRepo interface {
	CreatePRWithReviewers(ctx context.Context, pr domain.PullRequest, reviewers []string) (domain.PullRequest, error)
	GetByIDForUpdate(ctx context.Context, prID string) (domain.PullRequest, error)
	GetAssignedReviewers(ctx context.Context, prID string) ([]string, error)
	ReplaceReviewer(ctx context.Context, prID, oldID, newID string) (domain.PullRequest, error)
	SetMerged(ctx context.Context, prID string) (domain.PullRequest, error)
	ListByReviewer(ctx context.Context, reviewerID string) ([]domain.PullRequestShort, error)
	StatsByStatus(ctx context.Context) (map[domain.PRStatus]int, error)
}
