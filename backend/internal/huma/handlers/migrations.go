package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/getarcaneapp/arcane/backend/internal/services"
	"github.com/getarcaneapp/arcane/types/base"
	"github.com/getarcaneapp/arcane/types/migration"
)

// MigrationHandler handles database migration endpoints.
type MigrationHandler struct {
	migrationService *services.MigrationService
}

// --- Input/Output Types ---

type MigrationStatusInput struct {
	EnvironmentID string `path:"id" doc:"Environment ID"`
}

type MigrationStatusOutput struct {
	Body base.ApiResponse[[]migration.Status]
}

type MigrationDownInput struct {
	EnvironmentID string               `path:"id" doc:"Environment ID"`
	Body          migration.DownRequest `doc:"Migration rollback request"`
}

type MigrationDownOutput struct {
	Body base.ApiResponse[base.MessageResponse]
}

type MigrationDownToInput struct {
	EnvironmentID string                 `path:"id" doc:"Environment ID"`
	Body          migration.DownToRequest `doc:"Migration rollback request"`
}

type MigrationDownToOutput struct {
	Body base.ApiResponse[base.MessageResponse]
}

type MigrationRedoInput struct {
	EnvironmentID string `path:"id" doc:"Environment ID"`
}

type MigrationRedoOutput struct {
	Body base.ApiResponse[base.MessageResponse]
}

// RegisterMigrations registers migration endpoints using Huma.
func RegisterMigrations(api huma.API, migrationService *services.MigrationService) {
	h := &MigrationHandler{migrationService: migrationService}

	huma.Register(api, huma.Operation{
		OperationID: "migration-status",
		Method:      http.MethodGet,
		Path:        "/environments/{id}/migrate/status",
		Summary:     "Get migration status",
		Description: "List database migration status entries",
		Tags:        []string{"Migrations"},
		Security: []map[string][]string{
			{"BearerAuth": {}},
			{"ApiKeyAuth": {}},
		},
	}, h.Status)

	huma.Register(api, huma.Operation{
		OperationID: "migration-down",
		Method:      http.MethodPost,
		Path:        "/environments/{id}/migrate/down",
		Summary:     "Rollback migrations",
		Description: "Rollback migrations by a number of steps",
		Tags:        []string{"Migrations"},
		Security: []map[string][]string{
			{"BearerAuth": {}},
			{"ApiKeyAuth": {}},
		},
	}, h.Down)

	huma.Register(api, huma.Operation{
		OperationID: "migration-down-to",
		Method:      http.MethodPost,
		Path:        "/environments/{id}/migrate/down-to",
		Summary:     "Rollback migrations to version",
		Description: "Rollback migrations down to a specific version",
		Tags:        []string{"Migrations"},
		Security: []map[string][]string{
			{"BearerAuth": {}},
			{"ApiKeyAuth": {}},
		},
	}, h.DownTo)

	huma.Register(api, huma.Operation{
		OperationID: "migration-redo",
		Method:      http.MethodPost,
		Path:        "/environments/{id}/migrate/redo",
		Summary:     "Redo latest migration",
		Description: "Rollback and re-apply the most recent migration",
		Tags:        []string{"Migrations"},
		Security: []map[string][]string{
			{"BearerAuth": {}},
			{"ApiKeyAuth": {}},
		},
	}, h.Redo)
}

// Status returns the database migration status list.
func (h *MigrationHandler) Status(ctx context.Context, input *MigrationStatusInput) (*MigrationStatusOutput, error) {
	if h.migrationService == nil {
		return nil, huma.Error500InternalServerError("service not available")
	}

	if err := checkAdmin(ctx); err != nil {
		return nil, err
	}

	statuses, err := h.migrationService.Status(ctx)
	if err != nil {
		return nil, huma.Error500InternalServerError(fmt.Sprintf("failed to get migration status: %v", err))
	}

	return &MigrationStatusOutput{
		Body: base.ApiResponse[[]migration.Status]{
			Success: true,
			Data:    statuses,
		},
	}, nil
}

// Down rolls back migrations by a number of steps.
func (h *MigrationHandler) Down(ctx context.Context, input *MigrationDownInput) (*MigrationDownOutput, error) {
	if h.migrationService == nil {
		return nil, huma.Error500InternalServerError("service not available")
	}

	if err := checkAdmin(ctx); err != nil {
		return nil, err
	}

	if input.Body.Steps <= 0 {
		return nil, huma.Error400BadRequest("steps must be greater than 0")
	}

	if err := h.migrationService.Down(ctx, input.Body.Steps); err != nil {
		return nil, huma.Error500InternalServerError(fmt.Sprintf("failed to rollback migrations: %v", err))
	}

	return &MigrationDownOutput{
		Body: base.ApiResponse[base.MessageResponse]{
			Success: true,
			Data: base.MessageResponse{
				Message: fmt.Sprintf("Rolled back %d migration(s)", input.Body.Steps),
			},
		},
	}, nil
}

// DownTo rolls back migrations down to a specific version.
func (h *MigrationHandler) DownTo(ctx context.Context, input *MigrationDownToInput) (*MigrationDownToOutput, error) {
	if h.migrationService == nil {
		return nil, huma.Error500InternalServerError("service not available")
	}

	if err := checkAdmin(ctx); err != nil {
		return nil, err
	}

	if input.Body.Version <= 0 {
		return nil, huma.Error400BadRequest("version must be greater than 0")
	}

	if err := h.migrationService.DownTo(ctx, input.Body.Version); err != nil {
		return nil, huma.Error500InternalServerError(fmt.Sprintf("failed to rollback migrations to version %d: %v", input.Body.Version, err))
	}

	return &MigrationDownToOutput{
		Body: base.ApiResponse[base.MessageResponse]{
			Success: true,
			Data: base.MessageResponse{
				Message: fmt.Sprintf("Rolled back migrations to version %d", input.Body.Version),
			},
		},
	}, nil
}

// Redo rolls back and re-applies the most recent migration.
func (h *MigrationHandler) Redo(ctx context.Context, input *MigrationRedoInput) (*MigrationRedoOutput, error) {
	if h.migrationService == nil {
		return nil, huma.Error500InternalServerError("service not available")
	}

	if err := checkAdmin(ctx); err != nil {
		return nil, err
	}

	if err := h.migrationService.Redo(ctx); err != nil {
		return nil, huma.Error500InternalServerError(fmt.Sprintf("failed to redo latest migration: %v", err))
	}

	return &MigrationRedoOutput{
		Body: base.ApiResponse[base.MessageResponse]{
			Success: true,
			Data: base.MessageResponse{
				Message: "Redo completed successfully",
			},
		},
	}, nil
}
