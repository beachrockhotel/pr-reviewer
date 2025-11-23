package app

import (
	"context"
	"net/http"

	oapiadapter "github.com/beachrockhotel/pr-reviewer/internal/adapter/oapi"
	"github.com/beachrockhotel/pr-reviewer/internal/adapter/repo/postgres"
	"github.com/beachrockhotel/pr-reviewer/internal/platform/config"
	"github.com/beachrockhotel/pr-reviewer/internal/platform/httpserver"
	"github.com/beachrockhotel/pr-reviewer/internal/platform/log"
	"github.com/beachrockhotel/pr-reviewer/internal/usecase"
	prapi "github.com/beachrockhotel/pr-reviewer/shared/pkg/openapi/pr/v1"
)

func Run(ctx context.Context) error {
	cfg := config.Load()
	logger := log.New(cfg.LogLevel)

	pool, err := postgres.Connect(ctx, cfg.DB.DSN)
	if err != nil {
		return err
	}
	defer pool.Close()

	teamRepo := postgres.NewTeamRepo(pool)
	userRepo := postgres.NewUserRepo(pool)
	prRepo := postgres.NewPRRepo(pool)

	teamUC := usecase.NewTeamUsecase(teamRepo)
	userUC := usecase.NewUserUsecase(userRepo, prRepo)
	prUC := usecase.NewPRUsecase(userRepo, prRepo)

	h := oapiadapter.NewHandler(teamUC, userUC, prUC, logger)

	apiSrv, err := prapi.NewServer(h)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.Handle("/stats", http.HandlerFunc(h.StatsHTTP))
	mux.Handle("/", apiSrv)

	return httpserver.New(cfg.HTTPPort, mux, logger).Run(ctx)
}
