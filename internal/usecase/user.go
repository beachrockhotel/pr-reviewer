package usecase

import (
	"context"

	"github.com/beachrockhotel/pr-reviewer/internal/domain"
)

type UserUsecase struct {
	users UserRepo
	prs   PRRepo
}

func NewUserUsecase(users UserRepo, prs PRRepo) *UserUsecase {
	return &UserUsecase{users: users, prs: prs}
}

func (u *UserUsecase) SetActive(ctx context.Context, id string, active bool) (domain.User, error) {
	return u.users.SetActive(ctx, id, active)
}

func (u *UserUsecase) GetReviews(ctx context.Context, userID string) ([]domain.PullRequestShort, error) {
	return u.prs.ListByReviewer(ctx, userID)
}
