package main

import (
	"context"
	"database/sql"
	"time"

	"github.com/heroiclabs/nakama-common/runtime"
)

// These variables are populated via the Go linker. Set the initial value to unknown in case they are not set
var (
	// app metadata
	commitHash = "unknown"
	branch     = "unknown"
	buildTime  = "unknown"
)

func InitModule(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, initializer runtime.Initializer) error {
	// Initializing loggers
	logger.Info("Starting backend...")
	logger.Info("Current time in UTC: %s\n", time.Now().UTC())
	logger.Info("Git commit hash: %s\n", commitHash)
	logger.Info("Git branch: %s\n", branch)
	logger.Info("Build time in UTC: %s\n", buildTime)

	// Register as match handler, this call should be in InitModule.
	if err := initializer.RegisterMatch("match", func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule) (runtime.Match, error) {
		return &Match{}, nil
	}); err != nil {
		logger.Error("Unable to register: %v", err)
		return err
	}

	return nil
}
