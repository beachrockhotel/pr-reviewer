package usecase

import (
	"context"

	"github.com/beachrockhotel/pr-reviewer/internal/domain"
)

type TeamUsecase struct {
	teams TeamRepo
}

func NewTeamUsecase(teams TeamRepo) *TeamUsecase {
	return &TeamUsecase{teams: teams}
}

func (u *TeamUsecase) CreateTeam(ctx context.Context, teamName string, members []domain.User) (domain.Team, error) {
	if err := u.teams.CreateTeam(ctx, teamName); err != nil {
		return domain.Team{}, err
	}

	if err := u.teams.UpsertUsersToTeam(ctx, teamName, members); err != nil {
		return domain.Team{}, err
	}

	team, list, err := u.teams.GetTeamWithMembers(ctx, teamName)
	if err != nil {
		return domain.Team{}, err
	}
	team.Members = list

	return team, nil
}

func (u *TeamUsecase) GetTeam(ctx context.Context, teamName string) (domain.Team, error) {
	team, list, err := u.teams.GetTeamWithMembers(ctx, teamName)
	if err != nil {
		return domain.Team{}, err
	}
	team.Members = list
	return team, nil
}
