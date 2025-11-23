package oapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/beachrockhotel/pr-reviewer/internal/domain"
	"github.com/beachrockhotel/pr-reviewer/internal/usecase"
	pr "github.com/beachrockhotel/pr-reviewer/shared/pkg/openapi/pr/v1"
)

type Handler struct {
	pr.UnimplementedHandler

	team *usecase.TeamUsecase
	user *usecase.UserUsecase
	prUC *usecase.PRUsecase
	log  *slog.Logger
}

func NewHandler(team *usecase.TeamUsecase, user *usecase.UserUsecase, prUC *usecase.PRUsecase, logger *slog.Logger) *Handler {
	return &Handler{
		team: team,
		user: user,
		prUC: prUC,
		log:  logger,
	}
}

func makeError(code pr.ErrorResponseErrorCode, msg string) pr.ErrorResponse {
	return pr.ErrorResponse{
		Error: pr.ErrorResponseError{
			Code:    code,
			Message: msg,
		},
	}
}

func notFoundError() pr.ErrorResponse {
	return makeError(pr.ErrorResponseErrorCodeNOTFOUND, "resource not found")
}

func mapPRToSchema(p domain.PullRequest) pr.PullRequest {
	revs := make([]string, len(p.AssignedReviewers))
	copy(revs, p.AssignedReviewers)

	var created pr.OptNilDateTime
	if p.CreatedAt != nil {
		created.SetTo(*p.CreatedAt)
	}

	var merged pr.OptNilDateTime
	if p.MergedAt != nil {
		merged.SetTo(*p.MergedAt)
	}

	return pr.PullRequest{
		PullRequestID:     p.ID,
		PullRequestName:   p.Name,
		AuthorID:          p.AuthorID,
		Status:            pr.PullRequestStatus(p.Status),
		AssignedReviewers: revs,
		CreatedAt:         created,
		MergedAt:          merged,
	}
}

func (h *Handler) TeamAddPost(ctx context.Context, req *pr.Team) (pr.TeamAddPostRes, error) {
	members := make([]domain.User, 0, len(req.Members))
	for _, m := range req.Members {
		members = append(members, domain.User{
			UserID:   m.UserID,
			Username: m.Username,
			TeamName: req.TeamName,
			IsActive: m.IsActive,
		})
	}

	team, err := h.team.CreateTeam(ctx, req.TeamName, members)
	if err != nil {
		if errors.Is(err, domain.ErrTeamExists) {
			er := makeError(pr.ErrorResponseErrorCodeTEAMEXISTS, "team_name already exists")
			return &er, nil
		}
		return nil, err
	}

	outMembers := make([]pr.TeamMember, 0, len(team.Members))
	for _, u := range team.Members {
		outMembers = append(outMembers, pr.TeamMember{
			UserID:   u.UserID,
			Username: u.Username,
			IsActive: u.IsActive,
		})
	}

	teamSchema := pr.Team{
		TeamName: team.TeamName,
		Members:  outMembers,
	}

	return &pr.TeamAddPostCreated{
		Team: pr.NewOptTeam(teamSchema),
	}, nil
}

func (h *Handler) TeamGetGet(ctx context.Context, params pr.TeamGetGetParams) (pr.TeamGetGetRes, error) {
	team, err := h.team.GetTeam(ctx, params.TeamName)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			er := notFoundError()
			return &er, nil
		}
		return nil, err
	}

	members := make([]pr.TeamMember, 0, len(team.Members))
	for _, u := range team.Members {
		members = append(members, pr.TeamMember{
			UserID:   u.UserID,
			Username: u.Username,
			IsActive: u.IsActive,
		})
	}

	return &pr.Team{
		TeamName: team.TeamName,
		Members:  members,
	}, nil
}

func (h *Handler) UsersSetIsActivePost(ctx context.Context, req *pr.UsersSetIsActivePostReq) (pr.UsersSetIsActivePostRes, error) {
	u, err := h.user.SetActive(ctx, req.UserID, req.IsActive)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			er := notFoundError()
			return &er, nil
		}
		return nil, err
	}

	userSchema := pr.User{
		UserID:   u.UserID,
		Username: u.Username,
		TeamName: u.TeamName,
		IsActive: u.IsActive,
	}

	return &pr.UsersSetIsActivePostOK{
		User: pr.NewOptUser(userSchema),
	}, nil
}

func (h *Handler) PullRequestCreatePost(ctx context.Context, req *pr.PullRequestCreatePostReq) (pr.PullRequestCreatePostRes, error) {
	created, err := h.prUC.CreatePR(ctx, req.PullRequestID, req.PullRequestName, req.AuthorID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			e := notFoundError()
			nf := pr.PullRequestCreatePostNotFound(e)
			return &nf, nil
		case errors.Is(err, domain.ErrPRExists):
			e := makeError(pr.ErrorResponseErrorCodePREXISTS, "PR id already exists")
			cf := pr.PullRequestCreatePostConflict(e)
			return &cf, nil
		default:
			return nil, err
		}
	}

	prSchema := mapPRToSchema(created)

	return &pr.PullRequestCreatePostCreated{
		Pr: pr.NewOptPullRequest(prSchema),
	}, nil
}

func (h *Handler) PullRequestMergePost(ctx context.Context, req *pr.PullRequestMergePostReq) (pr.PullRequestMergePostRes, error) {
	merged, err := h.prUC.Merge(ctx, req.PullRequestID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			e := notFoundError()
			return &e, nil
		}
		return nil, err
	}

	prSchema := mapPRToSchema(merged)

	return &pr.PullRequestMergePostOK{
		Pr: pr.NewOptPullRequest(prSchema),
	}, nil
}

func (h *Handler) PullRequestReassignPost(ctx context.Context, req *pr.PullRequestReassignPostReq) (pr.PullRequestReassignPostRes, error) {
	updated, replacedBy, err := h.prUC.Reassign(ctx, req.PullRequestID, req.OldUserID)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrPRMerged):
			e := makeError(pr.ErrorResponseErrorCodePRMERGED, "cannot reassign on merged PR")
			cf := pr.PullRequestReassignPostConflict(e)
			return &cf, nil
		case errors.Is(err, domain.ErrNotAssigned):
			e := makeError(pr.ErrorResponseErrorCodeNOTASSIGNED, "reviewer is not assigned to this PR")
			cf := pr.PullRequestReassignPostConflict(e)
			return &cf, nil
		case errors.Is(err, domain.ErrNoCandidate):
			e := makeError(pr.ErrorResponseErrorCodeNOCANDIDATE, "no active replacement candidate in team")
			cf := pr.PullRequestReassignPostConflict(e)
			return &cf, nil
		case errors.Is(err, domain.ErrNotFound):
			e := notFoundError()
			nf := pr.PullRequestReassignPostNotFound(e)
			return &nf, nil
		default:
			return nil, err
		}
	}

	prSchema := mapPRToSchema(updated)

	return &pr.PullRequestReassignPostOK{
		Pr:         prSchema,
		ReplacedBy: replacedBy,
	}, nil
}

func (h *Handler) UsersGetReviewGet(ctx context.Context, params pr.UsersGetReviewGetParams) (*pr.UsersGetReviewGetOK, error) {
	list, err := h.user.GetReviews(ctx, params.UserID)
	if err != nil {
		return nil, err
	}

	prs := make([]pr.PullRequestShort, 0, len(list))
	for _, s := range list {
		prs = append(prs, pr.PullRequestShort{
			PullRequestID:   s.ID,
			PullRequestName: s.Name,
			AuthorID:        s.AuthorID,
			Status:          pr.PullRequestShortStatus(s.Status),
		})
	}

	return &pr.UsersGetReviewGetOK{
		UserID:       params.UserID,
		PullRequests: prs,
	}, nil
}

type StatsResponse struct {
	Open   int `json:"open"`
	Merged int `json:"merged"`
}

func (h *Handler) StatsHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stats, err := h.prUC.StatsByStatus(ctx)
	if err != nil {
		h.log.Error("stats: failed to get stats", "err", err)
		w.WriteHeader(http.StatusInternalServerError)

		if err := json.NewEncoder(w).Encode(map[string]string{"error": "internal error"}); err != nil {
			h.log.Error("stats: failed to write error response", "err", err)
		}

		return
	}

	resp := StatsResponse{
		Open:   stats[domain.StatusOpen],
		Merged: stats[domain.StatusMerged],
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.log.Error("stats: failed to write response", "err", err)
	}
}
