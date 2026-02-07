package migrate

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/getarcaneapp/arcane/backend/internal/config"
	"github.com/getarcaneapp/arcane/backend/internal/database"
)

var MigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Database migration commands",
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show migration status",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		db, err := connectDB(ctx)
		if err != nil {
			return err
		}
		defer db.Close()

		statuses, err := db.MigrateStatus(ctx)
		if err != nil {
			return err
		}

		for _, status := range statuses {
			appliedAt := ""
			if !status.AppliedAt.IsZero() {
				appliedAt = status.AppliedAt.Format("2006-01-02 15:04:05")
			}
			cmd.Printf("%d\t%s\t%s\t%s\n", status.Source.Version, status.State, appliedAt, status.Source.Path)
		}

		return nil
	},
}

var downCmd = &cobra.Command{
	Use:   "down [steps]",
	Short: "Rollback migrations by a number of steps",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		steps := 1
		if len(args) == 1 {
			parsed, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid steps: %w", err)
			}
			steps = parsed
		}

		ctx := cmd.Context()
		db, err := connectDB(ctx)
		if err != nil {
			return err
		}
		defer db.Close()

		return db.MigrateDown(ctx, steps)
	},
}

var downToCmd = &cobra.Command{
	Use:   "down-to <version>",
	Short: "Rollback migrations down to a specific version",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		version, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid version: %w", err)
		}

		ctx := cmd.Context()
		db, err := connectDB(ctx)
		if err != nil {
			return err
		}
		defer db.Close()

		return db.MigrateDownTo(ctx, version)
	},
}

var redoCmd = &cobra.Command{
	Use:   "redo",
	Short: "Rollback and re-apply the most recent migration",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		db, err := connectDB(ctx)
		if err != nil {
			return err
		}
		defer db.Close()

		return db.MigrateRedo(ctx)
	},
}

func init() {
	MigrateCmd.AddCommand(statusCmd)
	MigrateCmd.AddCommand(downCmd)
	MigrateCmd.AddCommand(downToCmd)
	MigrateCmd.AddCommand(redoCmd)
}

func connectDB(ctx context.Context) (*database.DB, error) {
	cfg := config.Load()
	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		return nil, err
	}
	return db, nil
}
