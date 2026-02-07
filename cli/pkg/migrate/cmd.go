package migrate

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/getarcaneapp/arcane/cli/internal/client"
	"github.com/getarcaneapp/arcane/cli/internal/output"
	"github.com/getarcaneapp/arcane/cli/internal/types"
	"github.com/getarcaneapp/arcane/types/base"
	"github.com/getarcaneapp/arcane/types/migration"
	"github.com/spf13/cobra"
)

var jsonOutput bool

// MigrateCmd is the parent command for migration operations.
var MigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Database migration commands",
}

var statusCmd = &cobra.Command{
	Use:          "status",
	Short:        "Show migration status",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.NewFromConfig()
		if err != nil {
			return err
		}

		resp, err := c.Get(cmd.Context(), types.Endpoints.MigrateStatus(c.EnvID()))
		if err != nil {
			return fmt.Errorf("failed to get migration status: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		var result base.ApiResponse[[]migration.Status]
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if jsonOutput {
			resultBytes, err := json.MarshalIndent(result.Data, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(resultBytes))
			return nil
		}

		headers := []string{"VERSION", "STATE", "APPLIED_AT", "PATH"}
		rows := make([][]string, len(result.Data))
		for i, status := range result.Data {
			rows[i] = []string{
				fmt.Sprintf("%d", status.Version),
				status.State,
				status.AppliedAt,
				status.Path,
			}
		}

		output.Table(headers, rows)
		fmt.Printf("\nTotal: %d migrations\n", len(result.Data))
		return nil
	},
}

var downCmd = &cobra.Command{
	Use:          "down [steps]",
	Short:        "Rollback migrations by a number of steps",
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		steps := 1
		if len(args) == 1 {
			parsed, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid steps: %w", err)
			}
			steps = parsed
		}

		c, err := client.NewFromConfig()
		if err != nil {
			return err
		}

		resp, err := c.Post(cmd.Context(), types.Endpoints.MigrateDown(c.EnvID()), migration.DownRequest{Steps: steps})
		if err != nil {
			return fmt.Errorf("failed to rollback migrations: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		var result base.ApiResponse[base.MessageResponse]
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if jsonOutput {
			resultBytes, err := json.MarshalIndent(result.Data, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(resultBytes))
			return nil
		}

		message := result.Data.Message
		if message == "" {
			message = fmt.Sprintf("Rolled back %d migration(s)", steps)
		}
		output.Success("%s", message)
		return nil
	},
}

var downToCmd = &cobra.Command{
	Use:          "down-to <version>",
	Short:        "Rollback migrations down to a specific version",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		version, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid version: %w", err)
		}

		c, err := client.NewFromConfig()
		if err != nil {
			return err
		}

		resp, err := c.Post(cmd.Context(), types.Endpoints.MigrateDownTo(c.EnvID()), migration.DownToRequest{Version: version})
		if err != nil {
			return fmt.Errorf("failed to rollback migrations: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		var result base.ApiResponse[base.MessageResponse]
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if jsonOutput {
			resultBytes, err := json.MarshalIndent(result.Data, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(resultBytes))
			return nil
		}

		message := result.Data.Message
		if message == "" {
			message = fmt.Sprintf("Rolled back migrations to version %d", version)
		}
		output.Success("%s", message)
		return nil
	},
}

var redoCmd = &cobra.Command{
	Use:          "redo",
	Short:        "Rollback and re-apply the most recent migration",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.NewFromConfig()
		if err != nil {
			return err
		}

		resp, err := c.Post(cmd.Context(), types.Endpoints.MigrateRedo(c.EnvID()), nil)
		if err != nil {
			return fmt.Errorf("failed to redo migration: %w", err)
		}
		defer func() { _ = resp.Body.Close() }()

		var result base.ApiResponse[base.MessageResponse]
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}

		if jsonOutput {
			resultBytes, err := json.MarshalIndent(result.Data, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal JSON: %w", err)
			}
			fmt.Println(string(resultBytes))
			return nil
		}

		message := result.Data.Message
		if message == "" {
			message = "Redo completed successfully"
		}
		output.Success("%s", message)
		return nil
	},
}

func init() {
	MigrateCmd.AddCommand(statusCmd)
	MigrateCmd.AddCommand(downCmd)
	MigrateCmd.AddCommand(downToCmd)
	MigrateCmd.AddCommand(redoCmd)

	statusCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	downCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	downToCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	redoCmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
}
