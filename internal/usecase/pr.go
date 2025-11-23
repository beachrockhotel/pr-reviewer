package usecase

import (
	"context"
	"errors"

	"github.com/beachrockhotel/pr-reviewer/internal/domain"
)

type PRUsecase struct {
	users UserRepo
	prs   PRRepo
}

func NewPRUsecase(users UserRepo, prs PRRepo) *PRUsecase {
	return &PRUsecase{users: users, prs: prs}
}

func (u *PRUsecase) CreatePR(ctx context.Context, prID, name, authorID string) (domain.PullRequest, error) {
	author, err := u.users.GetByID(ctx, authorID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.PullRequest{}, domain.ErrNotFound
		}
		return domain.PullRequest{}, err
	}

	exclude := []string{author.UserID}
	cands, err := u.users.ListActiveInTeamExcept(ctx, author.TeamName, exclude, 2)
	if err != nil {
		return domain.PullRequest{}, err
	}

	revs := make([]string, 0, len(cands))
	for _, c := range cands {
		revs = append(revs, c.UserID)
	}

	pr := domain.PullRequest{
		ID:       prID,
		Name:     name,
		AuthorID: authorID,
		Status:   domain.StatusOpen,
	}

	return u.prs.CreatePRWithReviewers(ctx, pr, revs)
}

func (u *PRUsecase) Reassign(ctx context.Context, prID, oldUserID string) (domain.PullRequest, string, error) {
	pr, err := u.prs.GetByIDForUpdate(ctx, prID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.PullRequest{}, "", domain.ErrNotFound
		}
		return domain.PullRequest{}, "", err
	}

	if pr.Status == domain.StatusMerged {
		return domain.PullRequest{}, "", domain.ErrPRMerged
	}

	assigned, err := u.prs.GetAssignedReviewers(ctx, prID)
	if err != nil {
		return domain.PullRequest{}, "", err
	}

	found := false
	for _, r := range assigned {
		if r == oldUserID {
			found = true
			break
		}
	}
	if !found {
		return domain.PullRequest{}, "", domain.ErrNotAssigned
	}

	oldUser, err := u.users.GetByID(ctx, oldUserID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return domain.PullRequest{}, "", domain.ErrNotFound
		}
		return domain.PullRequest{}, "", err
	}

	exclude := append(append([]string{}, assigned...), pr.AuthorID)

	cands, err := u.users.ListActiveInTeamExcept(ctx, oldUser.TeamName, exclude, 20)
	if err != nil {
		return domain.PullRequest{}, "", err
	}

	next := ""
	for _, c := range cands {
		if c.UserID == oldUserID {
			continue
		}

		taken := false
		for _, a := range assigned {
			if a == c.UserID {
				taken = true
				break
			}
		}
		if !taken {
			next = c.UserID
			break
		}
	}

	if next == "" {
		return domain.PullRequest{}, "", domain.ErrNoCandidate
	}

	updated, err := u.prs.ReplaceReviewer(ctx, prID, oldUserID, next)
	return updated, next, err
}

func (u *PRUsecase) Merge(ctx context.Context, prID string) (domain.PullRequest, error) {
	return u.prs.SetMerged(ctx, prID)
}

func (u *PRUsecase) StatsByStatus(ctx context.Context) (map[domain.PRStatus]int, error) {
	return u.prs.StatsByStatus(ctx)
}
