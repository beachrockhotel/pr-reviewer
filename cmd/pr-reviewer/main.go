package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/beachrockhotel/pr-reviewer/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx); err != nil {
		log.Println("app exited with error:", err)
	}
}
