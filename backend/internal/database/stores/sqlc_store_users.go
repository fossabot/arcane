package stores

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/getarcaneapp/arcane/backend/internal/database/models/pgdb"
	"github.com/getarcaneapp/arcane/backend/internal/database/models/sqlitedb"
	"github.com/getarcaneapp/arcane/types/base"
	"github.com/getarcaneapp/arcane/types/user"
)

func (s *SqlcStore) CreateUser(ctx context.Context, user user.ModelUser) (*user.ModelUser, error) {
	if user.ID == "" {
		user.ID = uuid.NewString()
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now().UTC()
	}
	if user.Roles == nil {
		user.Roles = base.StringSlice{}
	}

	switch s.driver {
	case "postgres":
		row, err := s.pg.CreateUser(ctx, pgdb.CreateUserParams{
			ID:                       user.ID,
			Username:                 user.Username,
			PasswordHash:             user.PasswordHash,
			DisplayName:              nullableTextPtrKeepEmpty(user.DisplayName),
			Email:                    nullableTextPtrKeepEmpty(user.Email),
			Roles:                    user.Roles,
			RequirePasswordChange:    user.RequiresPasswordChange,
			OidcSubjectID:            nullableTextPtrKeepEmpty(user.OidcSubjectId),
			LastLogin:                nullableTimestamptzPtr(user.LastLogin),
			CreatedAt:                nullableTimestamptz(user.CreatedAt),
			UpdatedAt:                nullableTimestamptzPtr(user.UpdatedAt),
			OidcAccessToken:          nullableTextPtrKeepEmpty(user.OidcAccessToken),
			OidcRefreshToken:         nullableTextPtrKeepEmpty(user.OidcRefreshToken),
			OidcAccessTokenExpiresAt: nullableTimestamptzPtr(user.OidcAccessTokenExpiresAt),
			Locale:                   nullableTextPtrKeepEmpty(user.Locale),
			RequiresPasswordChange:   user.RequiresPasswordChange,
		})
		if err != nil {
			return nil, err
		}
		return mapUserFromPG(row), nil
	case "sqlite":
		row, err := s.sqlite.CreateUser(ctx, sqlitedb.CreateUserParams{
			ID:                       user.ID,
			Username:                 user.Username,
			PasswordHash:             user.PasswordHash,
			DisplayName:              nullableNullStringPtrKeepEmpty(user.DisplayName),
			Email:                    nullableNullStringPtrKeepEmpty(user.Email),
			Roles:                    user.Roles,
			RequirePasswordChange:    user.RequiresPasswordChange,
			OidcSubjectID:            nullableNullStringPtrKeepEmpty(user.OidcSubjectId),
			LastLogin:                nullableNullTimePtr(user.LastLogin),
			CreatedAt:                user.CreatedAt,
			UpdatedAt:                nullableNullTimePtr(user.UpdatedAt),
			OidcAccessToken:          nullableNullStringPtrKeepEmpty(user.OidcAccessToken),
			OidcRefreshToken:         nullableNullStringPtrKeepEmpty(user.OidcRefreshToken),
			OidcAccessTokenExpiresAt: nullableNullTimePtr(user.OidcAccessTokenExpiresAt),
			Locale:                   nullableNullStringPtrKeepEmpty(user.Locale),
			RequiresPasswordChange:   user.RequiresPasswordChange,
		})
		if err != nil {
			return nil, err
		}
		return mapUserFromSQLite(row), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) GetUserByUsername(ctx context.Context, username string) (*user.ModelUser, error) {
	switch s.driver {
	case "postgres":
		row, err := s.pg.GetUserByUsername(ctx, username)
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapUserFromPG(row), nil
	case "sqlite":
		row, err := s.sqlite.GetUserByUsername(ctx, username)
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapUserFromSQLite(row), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) GetUserByID(ctx context.Context, id string) (*user.ModelUser, error) {
	switch s.driver {
	case "postgres":
		row, err := s.pg.GetUserByID(ctx, id)
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapUserFromPG(row), nil
	case "sqlite":
		row, err := s.sqlite.GetUserByID(ctx, id)
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapUserFromSQLite(row), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) GetUserByOidcSubjectID(ctx context.Context, subjectID string) (*user.ModelUser, error) {
	switch s.driver {
	case "postgres":
		row, err := s.pg.GetUserByOidcSubjectID(ctx, nullableText(subjectID))
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapUserFromPG(row), nil
	case "sqlite":
		row, err := s.sqlite.GetUserByOidcSubjectID(ctx, nullableString(subjectID))
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapUserFromSQLite(row), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) GetUserByEmail(ctx context.Context, email string) (*user.ModelUser, error) {
	switch s.driver {
	case "postgres":
		row, err := s.pg.GetUserByEmail(ctx, nullableText(email))
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapUserFromPG(row), nil
	case "sqlite":
		row, err := s.sqlite.GetUserByEmail(ctx, nullableString(email))
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapUserFromSQLite(row), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) SaveUser(ctx context.Context, user user.ModelUser) (*user.ModelUser, error) {
	if user.ID == "" {
		return nil, fmt.Errorf("user id is required")
	}
	if user.Roles == nil {
		user.Roles = base.StringSlice{}
	}
	if user.UpdatedAt == nil {
		now := time.Now().UTC()
		user.UpdatedAt = &now
	}

	switch s.driver {
	case "postgres":
		row, err := s.pg.UpdateUser(ctx, pgdb.UpdateUserParams{
			ID:                       user.ID,
			Username:                 user.Username,
			PasswordHash:             user.PasswordHash,
			DisplayName:              nullableTextPtrKeepEmpty(user.DisplayName),
			Email:                    nullableTextPtrKeepEmpty(user.Email),
			Roles:                    user.Roles,
			RequirePasswordChange:    user.RequiresPasswordChange,
			OidcSubjectID:            nullableTextPtrKeepEmpty(user.OidcSubjectId),
			LastLogin:                nullableTimestamptzPtr(user.LastLogin),
			UpdatedAt:                nullableTimestamptzPtr(user.UpdatedAt),
			OidcAccessToken:          nullableTextPtrKeepEmpty(user.OidcAccessToken),
			OidcRefreshToken:         nullableTextPtrKeepEmpty(user.OidcRefreshToken),
			OidcAccessTokenExpiresAt: nullableTimestamptzPtr(user.OidcAccessTokenExpiresAt),
			Locale:                   nullableTextPtrKeepEmpty(user.Locale),
			RequiresPasswordChange:   user.RequiresPasswordChange,
		})
		if err != nil {
			return nil, err
		}
		return mapUserFromPG(row), nil
	case "sqlite":
		row, err := s.sqlite.UpdateUser(ctx, sqlitedb.UpdateUserParams{
			Username:                 user.Username,
			PasswordHash:             user.PasswordHash,
			DisplayName:              nullableNullStringPtrKeepEmpty(user.DisplayName),
			Email:                    nullableNullStringPtrKeepEmpty(user.Email),
			Roles:                    user.Roles,
			RequirePasswordChange:    user.RequiresPasswordChange,
			OidcSubjectID:            nullableNullStringPtrKeepEmpty(user.OidcSubjectId),
			LastLogin:                nullableNullTimePtr(user.LastLogin),
			UpdatedAt:                nullableNullTimePtr(user.UpdatedAt),
			OidcAccessToken:          nullableNullStringPtrKeepEmpty(user.OidcAccessToken),
			OidcRefreshToken:         nullableNullStringPtrKeepEmpty(user.OidcRefreshToken),
			OidcAccessTokenExpiresAt: nullableNullTimePtr(user.OidcAccessTokenExpiresAt),
			Locale:                   nullableNullStringPtrKeepEmpty(user.Locale),
			RequiresPasswordChange:   user.RequiresPasswordChange,
			ID:                       user.ID,
		})
		if err != nil {
			return nil, err
		}
		return mapUserFromSQLite(row), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) AttachOidcSubjectTransactional(ctx context.Context, userID string, subject string, updateFn func(u *user.ModelUser)) (*user.ModelUser, error) {
	switch s.driver {
	case "postgres":
		if slog.Default().Enabled(ctx, slog.LevelDebug) {
			slog.DebugContext(ctx, "Database transaction begin", "driver", "postgres", "operation", "AttachOidcSubjectTransactional")
		}
		tx, err := s.pgPool.Begin(ctx)
		if err != nil {
			return nil, err
		}
		rolledBack := false
		defer func() {
			if !rolledBack {
				return
			}
			if slog.Default().Enabled(ctx, slog.LevelDebug) {
				slog.DebugContext(ctx, "Database transaction rollback (defer)", "driver", "postgres", "operation", "AttachOidcSubjectTransactional")
			}
			_ = tx.Rollback(ctx)
		}()

		txStore := s.withPgTx(tx)
		row, err := txStore.pg.GetUserByIDForUpdate(ctx, userID)
		if err != nil {
			rolledBack = true
			if isNotFound(err) {
				return nil, nil
			}
			return nil, fmt.Errorf("failed to load user for OIDC merge: %w", err)
		}

		user := mapUserFromPG(row)
		if user.OidcSubjectId != nil && *user.OidcSubjectId != "" && *user.OidcSubjectId != subject {
			rolledBack = true
			return nil, fmt.Errorf("user already linked to another OIDC subject")
		}
		user.OidcSubjectId = &subject
		if updateFn != nil {
			updateFn(user)
		}

		if user.Roles == nil {
			user.Roles = base.StringSlice{}
		}
		now := time.Now().UTC()
		user.UpdatedAt = &now

		updated, err := txStore.pg.UpdateUser(ctx, pgdb.UpdateUserParams{
			ID:                       user.ID,
			Username:                 user.Username,
			PasswordHash:             user.PasswordHash,
			DisplayName:              nullableTextPtrKeepEmpty(user.DisplayName),
			Email:                    nullableTextPtrKeepEmpty(user.Email),
			Roles:                    user.Roles,
			RequirePasswordChange:    user.RequiresPasswordChange,
			OidcSubjectID:            nullableTextPtrKeepEmpty(user.OidcSubjectId),
			LastLogin:                nullableTimestamptzPtr(user.LastLogin),
			UpdatedAt:                nullableTimestamptz(now),
			OidcAccessToken:          nullableTextPtrKeepEmpty(user.OidcAccessToken),
			OidcRefreshToken:         nullableTextPtrKeepEmpty(user.OidcRefreshToken),
			OidcAccessTokenExpiresAt: nullableTimestamptzPtr(user.OidcAccessTokenExpiresAt),
			Locale:                   nullableTextPtrKeepEmpty(user.Locale),
			RequiresPasswordChange:   user.RequiresPasswordChange,
		})
		if err != nil {
			rolledBack = true
			if strings.Contains(strings.ToLower(err.Error()), "unique") || strings.Contains(strings.ToLower(err.Error()), "duplicate key") {
				return nil, fmt.Errorf("oidc subject is already linked to another user: %w", err)
			}
			return nil, fmt.Errorf("failed to persist OIDC merge: %w", err)
		}

		if err := tx.Commit(ctx); err != nil {
			rolledBack = true
			if slog.Default().Enabled(ctx, slog.LevelDebug) {
				slog.DebugContext(ctx, "Database transaction commit failed", "driver", "postgres", "operation", "AttachOidcSubjectTransactional", "error", err)
			}
			return nil, err
		}
		if slog.Default().Enabled(ctx, slog.LevelDebug) {
			slog.DebugContext(ctx, "Database transaction commit", "driver", "postgres", "operation", "AttachOidcSubjectTransactional")
		}
		return mapUserFromPG(updated), nil

	case "sqlite":
		if slog.Default().Enabled(ctx, slog.LevelDebug) {
			slog.DebugContext(ctx, "Database transaction begin", "driver", "sqlite", "operation", "AttachOidcSubjectTransactional")
		}
		tx, err := s.sqlDB.BeginTx(ctx, nil)
		if err != nil {
			return nil, err
		}
		rolledBack := false
		defer func() {
			if !rolledBack {
				return
			}
			if slog.Default().Enabled(ctx, slog.LevelDebug) {
				slog.DebugContext(ctx, "Database transaction rollback (defer)", "driver", "sqlite", "operation", "AttachOidcSubjectTransactional")
			}
			_ = tx.Rollback()
		}()

		txStore := s.withSQLiteTx(tx)
		row, err := txStore.sqlite.GetUserByIDForUpdate(ctx, userID)
		if err != nil {
			rolledBack = true
			if isNotFound(err) {
				return nil, nil
			}
			return nil, fmt.Errorf("failed to load user for OIDC merge: %w", err)
		}

		user := mapUserFromSQLite(row)
		if user.OidcSubjectId != nil && *user.OidcSubjectId != "" && *user.OidcSubjectId != subject {
			rolledBack = true
			return nil, fmt.Errorf("user already linked to another OIDC subject")
		}
		user.OidcSubjectId = &subject
		if updateFn != nil {
			updateFn(user)
		}

		if user.Roles == nil {
			user.Roles = base.StringSlice{}
		}
		now := time.Now().UTC()
		user.UpdatedAt = &now

		updated, err := txStore.sqlite.UpdateUser(ctx, sqlitedb.UpdateUserParams{
			Username:                 user.Username,
			PasswordHash:             user.PasswordHash,
			DisplayName:              nullableNullStringPtrKeepEmpty(user.DisplayName),
			Email:                    nullableNullStringPtrKeepEmpty(user.Email),
			Roles:                    user.Roles,
			RequirePasswordChange:    user.RequiresPasswordChange,
			OidcSubjectID:            nullableNullStringPtrKeepEmpty(user.OidcSubjectId),
			LastLogin:                nullableNullTimePtr(user.LastLogin),
			UpdatedAt:                nullableNullTime(now),
			OidcAccessToken:          nullableNullStringPtrKeepEmpty(user.OidcAccessToken),
			OidcRefreshToken:         nullableNullStringPtrKeepEmpty(user.OidcRefreshToken),
			OidcAccessTokenExpiresAt: nullableNullTimePtr(user.OidcAccessTokenExpiresAt),
			Locale:                   nullableNullStringPtrKeepEmpty(user.Locale),
			RequiresPasswordChange:   user.RequiresPasswordChange,
			ID:                       user.ID,
		})
		if err != nil {
			rolledBack = true
			if strings.Contains(strings.ToLower(err.Error()), "unique") || strings.Contains(strings.ToLower(err.Error()), "duplicate key") {
				return nil, fmt.Errorf("oidc subject is already linked to another user: %w", err)
			}
			return nil, fmt.Errorf("failed to persist OIDC merge: %w", err)
		}

		if err := tx.Commit(); err != nil {
			rolledBack = true
			if slog.Default().Enabled(ctx, slog.LevelDebug) {
				slog.DebugContext(ctx, "Database transaction commit failed", "driver", "sqlite", "operation", "AttachOidcSubjectTransactional", "error", err)
			}
			return nil, err
		}
		if slog.Default().Enabled(ctx, slog.LevelDebug) {
			slog.DebugContext(ctx, "Database transaction commit", "driver", "sqlite", "operation", "AttachOidcSubjectTransactional")
		}
		return mapUserFromSQLite(updated), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) CountUsers(ctx context.Context) (int64, error) {
	switch s.driver {
	case "postgres":
		return s.pg.CountUsers(ctx)
	case "sqlite":
		return s.sqlite.CountUsers(ctx)
	default:
		return 0, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) DeleteUserByID(ctx context.Context, id string) (bool, error) {
	switch s.driver {
	case "postgres":
		rows, err := s.pg.DeleteUserByID(ctx, id)
		if err != nil {
			return false, err
		}
		return rows > 0, nil
	case "sqlite":
		rows, err := s.sqlite.DeleteUserByID(ctx, id)
		if err != nil {
			return false, err
		}
		return rows > 0, nil
	default:
		return false, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) UpdateUserPasswordHash(ctx context.Context, id string, passwordHash string, updatedAt time.Time) error {
	switch s.driver {
	case "postgres":
		return s.pg.UpdateUserPasswordHash(ctx, pgdb.UpdateUserPasswordHashParams{
			ID:           id,
			PasswordHash: passwordHash,
			UpdatedAt:    nullableTimestamptz(updatedAt),
		})
	case "sqlite":
		return s.sqlite.UpdateUserPasswordHash(ctx, sqlitedb.UpdateUserPasswordHashParams{
			PasswordHash: passwordHash,
			UpdatedAt:    nullableNullTime(updatedAt),
			ID:           id,
		})
	default:
		return fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) ListUsers(ctx context.Context) ([]user.ModelUser, error) {
	switch s.driver {
	case "postgres":
		rows, err := s.pg.ListUsers(ctx)
		if err != nil {
			return nil, err
		}
		users := make([]user.ModelUser, 0, len(rows))
		for _, row := range rows {
			users = append(users, *mapUserFromPG(row))
		}
		return users, nil
	case "sqlite":
		rows, err := s.sqlite.ListUsers(ctx)
		if err != nil {
			return nil, err
		}
		users := make([]user.ModelUser, 0, len(rows))
		for _, row := range rows {
			users = append(users, *mapUserFromSQLite(row))
		}
		return users, nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func mapUserFromPG(row *pgdb.User) *user.ModelUser {
	if row == nil {
		return nil
	}
	requiresPasswordChange := row.RequiresPasswordChange || row.RequirePasswordChange
	return &user.ModelUser{
		BaseModel: base.BaseModel{
			ID:        row.ID,
			CreatedAt: timeFromPgTimestamptz(row.CreatedAt),
			UpdatedAt: timePtrFromPgTimestamptz(row.UpdatedAt),
		},
		Username:                 row.Username,
		PasswordHash:             row.PasswordHash,
		DisplayName:              stringPtrFromPgText(row.DisplayName),
		Email:                    stringPtrFromPgText(row.Email),
		Roles:                    row.Roles,
		OidcSubjectId:            stringPtrFromPgText(row.OidcSubjectID),
		LastLogin:                timePtrFromPgTimestamptz(row.LastLogin),
		OidcAccessToken:          stringPtrFromPgText(row.OidcAccessToken),
		OidcRefreshToken:         stringPtrFromPgText(row.OidcRefreshToken),
		OidcAccessTokenExpiresAt: timePtrFromPgTimestamptz(row.OidcAccessTokenExpiresAt),
		Locale:                   stringPtrFromPgText(row.Locale),
		RequiresPasswordChange:   requiresPasswordChange,
	}
}

func mapUserFromSQLite(row *sqlitedb.User) *user.ModelUser {
	if row == nil {
		return nil
	}
	requiresPasswordChange := row.RequiresPasswordChange || row.RequirePasswordChange
	return &user.ModelUser{
		BaseModel: base.BaseModel{
			ID:        row.ID,
			CreatedAt: row.CreatedAt,
			UpdatedAt: timePtrFromNull(row.UpdatedAt),
		},
		Username:                 row.Username,
		PasswordHash:             row.PasswordHash,
		DisplayName:              stringPtrFromNull(row.DisplayName),
		Email:                    stringPtrFromNull(row.Email),
		Roles:                    row.Roles,
		OidcSubjectId:            stringPtrFromNull(row.OidcSubjectID),
		LastLogin:                timePtrFromNull(row.LastLogin),
		OidcAccessToken:          stringPtrFromNull(row.OidcAccessToken),
		OidcRefreshToken:         stringPtrFromNull(row.OidcRefreshToken),
		OidcAccessTokenExpiresAt: timePtrFromNull(row.OidcAccessTokenExpiresAt),
		Locale:                   stringPtrFromNull(row.Locale),
		RequiresPasswordChange:   requiresPasswordChange,
	}
}
