package stores

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/getarcaneapp/arcane/backend/internal/database/models/pgdb"
	"github.com/getarcaneapp/arcane/backend/internal/database/models/sqlitedb"
	"github.com/getarcaneapp/arcane/types/template"
)

func (s *SqlcStore) GetComposeTemplateByID(ctx context.Context, id string) (*template.ComposeTemplate, error) {
	switch s.driver {
	case "postgres":
		row, err := s.pg.GetComposeTemplateByID(ctx, id)
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapComposeTemplateFromPGValues(
			row.ID, row.Name, row.Description, row.Content, row.EnvContent, row.IsCustom, row.IsRemote, row.RegistryID,
			row.MetaVersion, row.MetaAuthor, row.MetaTags, row.MetaRemoteUrl, row.MetaEnvUrl, row.MetaDocumentationUrl,
			row.CreatedAt, row.UpdatedAt,
		), nil
	case "sqlite":
		row, err := s.sqlite.GetComposeTemplateByID(ctx, id)
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapComposeTemplateFromSQLiteValues(
			row.ID, row.Name, row.Description, row.Content, row.EnvContent, row.IsCustom, row.IsRemote, row.RegistryID,
			row.MetaVersion, row.MetaAuthor, row.MetaTags, row.MetaRemoteUrl, row.MetaEnvUrl, row.MetaDocumentationUrl,
			row.CreatedAt, row.UpdatedAt,
		), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) ListComposeTemplates(ctx context.Context) ([]template.ComposeTemplate, error) {
	switch s.driver {
	case "postgres":
		rows, err := s.pg.ListComposeTemplates(ctx)
		if err != nil {
			return nil, err
		}
		items := make([]template.ComposeTemplate, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapComposeTemplateFromPGValues(
				row.ID, row.Name, row.Description, row.Content, row.EnvContent, row.IsCustom, row.IsRemote, row.RegistryID,
				row.MetaVersion, row.MetaAuthor, row.MetaTags, row.MetaRemoteUrl, row.MetaEnvUrl, row.MetaDocumentationUrl,
				row.CreatedAt, row.UpdatedAt,
			))
		}
		return items, nil
	case "sqlite":
		rows, err := s.sqlite.ListComposeTemplates(ctx)
		if err != nil {
			return nil, err
		}
		items := make([]template.ComposeTemplate, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapComposeTemplateFromSQLiteValues(
				row.ID, row.Name, row.Description, row.Content, row.EnvContent, row.IsCustom, row.IsRemote, row.RegistryID,
				row.MetaVersion, row.MetaAuthor, row.MetaTags, row.MetaRemoteUrl, row.MetaEnvUrl, row.MetaDocumentationUrl,
				row.CreatedAt, row.UpdatedAt,
			))
		}
		return items, nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) FindLocalComposeTemplateByDescriptionOrName(ctx context.Context, description string, name string) (*template.ComposeTemplate, error) {
	switch s.driver {
	case "postgres":
		row, err := s.pg.FindLocalComposeTemplateByDescriptionOrName(ctx, pgdb.FindLocalComposeTemplateByDescriptionOrNameParams{
			Description: pgtype.Text{String: description, Valid: true},
			Name:        name,
		})
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapComposeTemplateFromPGValues(
			row.ID, row.Name, row.Description, row.Content, row.EnvContent, row.IsCustom, row.IsRemote, row.RegistryID,
			row.MetaVersion, row.MetaAuthor, row.MetaTags, row.MetaRemoteUrl, row.MetaEnvUrl, row.MetaDocumentationUrl,
			row.CreatedAt, row.UpdatedAt,
		), nil
	case "sqlite":
		row, err := s.sqlite.FindLocalComposeTemplateByDescriptionOrName(ctx, sqlitedb.FindLocalComposeTemplateByDescriptionOrNameParams{
			Description: sql.NullString{String: description, Valid: true},
			Name:        name,
		})
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapComposeTemplateFromSQLiteValues(
			row.ID, row.Name, row.Description, row.Content, row.EnvContent, row.IsCustom, row.IsRemote, row.RegistryID,
			row.MetaVersion, row.MetaAuthor, row.MetaTags, row.MetaRemoteUrl, row.MetaEnvUrl, row.MetaDocumentationUrl,
			row.CreatedAt, row.UpdatedAt,
		), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) FindLocalComposeTemplateByDescription(ctx context.Context, description string) (*template.ComposeTemplate, error) {
	switch s.driver {
	case "postgres":
		row, err := s.pg.FindLocalComposeTemplateByDescription(ctx, pgtype.Text{String: description, Valid: true})
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapComposeTemplateFromPGValues(
			row.ID, row.Name, row.Description, row.Content, row.EnvContent, row.IsCustom, row.IsRemote, row.RegistryID,
			row.MetaVersion, row.MetaAuthor, row.MetaTags, row.MetaRemoteUrl, row.MetaEnvUrl, row.MetaDocumentationUrl,
			row.CreatedAt, row.UpdatedAt,
		), nil
	case "sqlite":
		row, err := s.sqlite.FindLocalComposeTemplateByDescription(ctx, sql.NullString{String: description, Valid: true})
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapComposeTemplateFromSQLiteValues(
			row.ID, row.Name, row.Description, row.Content, row.EnvContent, row.IsCustom, row.IsRemote, row.RegistryID,
			row.MetaVersion, row.MetaAuthor, row.MetaTags, row.MetaRemoteUrl, row.MetaEnvUrl, row.MetaDocumentationUrl,
			row.CreatedAt, row.UpdatedAt,
		), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) CreateComposeTemplate(ctx context.Context, template template.ComposeTemplate) (*template.ComposeTemplate, error) {
	createdAt := template.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	updatedAt := template.UpdatedAt
	if updatedAt == nil {
		updatedAt = &createdAt
	}
	metaVersion, metaAuthor, metaTags, metaRemoteURL, metaEnvURL, metaDocumentationURL := templateMetadataToColumns(template.Metadata)

	switch s.driver {
	case "postgres":
		row, err := s.pg.CreateComposeTemplate(ctx, pgdb.CreateComposeTemplateParams{
			ID:                   template.ID,
			Name:                 template.Name,
			Description:          nullableTextPtrKeepEmpty(&template.Description),
			Content:              nullableTextPtrKeepEmpty(&template.Content),
			EnvContent:           nullableTextPtrKeepEmpty(template.EnvContent),
			IsCustom:             template.IsCustom,
			IsRemote:             template.IsRemote,
			RegistryID:           nullableTextPtrKeepEmpty(template.RegistryID),
			MetaVersion:          nullableTextPtrKeepEmpty(metaVersion),
			MetaAuthor:           nullableTextPtrKeepEmpty(metaAuthor),
			MetaTags:             nullableTextPtrKeepEmpty(metaTags),
			MetaRemoteUrl:        nullableTextPtrKeepEmpty(metaRemoteURL),
			MetaEnvUrl:           nullableTextPtrKeepEmpty(metaEnvURL),
			MetaDocumentationUrl: nullableTextPtrKeepEmpty(metaDocumentationURL),
			CreatedAt:            nullableTimestamptz(createdAt),
			UpdatedAt:            nullableTimestamptzPtr(updatedAt),
		})
		if err != nil {
			return nil, err
		}
		return mapComposeTemplateFromPGValues(
			row.ID, row.Name, row.Description, row.Content, row.EnvContent, row.IsCustom, row.IsRemote, row.RegistryID,
			row.MetaVersion, row.MetaAuthor, row.MetaTags, row.MetaRemoteUrl, row.MetaEnvUrl, row.MetaDocumentationUrl,
			row.CreatedAt, row.UpdatedAt,
		), nil
	case "sqlite":
		row, err := s.sqlite.CreateComposeTemplate(ctx, sqlitedb.CreateComposeTemplateParams{
			ID:                   template.ID,
			Name:                 template.Name,
			Description:          sql.NullString{String: template.Description, Valid: true},
			Content:              sql.NullString{String: template.Content, Valid: true},
			EnvContent:           nullableNullStringPtrKeepEmpty(template.EnvContent),
			IsCustom:             template.IsCustom,
			IsRemote:             template.IsRemote,
			RegistryID:           nullableNullStringPtrKeepEmpty(template.RegistryID),
			MetaVersion:          nullableNullStringPtrKeepEmpty(metaVersion),
			MetaAuthor:           nullableNullStringPtrKeepEmpty(metaAuthor),
			MetaTags:             nullableNullStringPtrKeepEmpty(metaTags),
			MetaRemoteUrl:        nullableNullStringPtrKeepEmpty(metaRemoteURL),
			MetaEnvUrl:           nullableNullStringPtrKeepEmpty(metaEnvURL),
			MetaDocumentationUrl: nullableNullStringPtrKeepEmpty(metaDocumentationURL),
			CreatedAt:            createdAt,
			UpdatedAt:            nullableNullTimePtr(updatedAt),
		})
		if err != nil {
			return nil, err
		}
		return mapComposeTemplateFromSQLiteValues(
			row.ID, row.Name, row.Description, row.Content, row.EnvContent, row.IsCustom, row.IsRemote, row.RegistryID,
			row.MetaVersion, row.MetaAuthor, row.MetaTags, row.MetaRemoteUrl, row.MetaEnvUrl, row.MetaDocumentationUrl,
			row.CreatedAt, row.UpdatedAt,
		), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) SaveComposeTemplate(ctx context.Context, template template.ComposeTemplate) (*template.ComposeTemplate, error) {
	updatedAt := template.UpdatedAt
	if updatedAt == nil {
		now := time.Now()
		updatedAt = &now
	}
	metaVersion, metaAuthor, metaTags, metaRemoteURL, metaEnvURL, metaDocumentationURL := templateMetadataToColumns(template.Metadata)

	switch s.driver {
	case "postgres":
		row, err := s.pg.SaveComposeTemplate(ctx, pgdb.SaveComposeTemplateParams{
			Name:                 template.Name,
			Description:          nullableTextPtrKeepEmpty(&template.Description),
			Content:              nullableTextPtrKeepEmpty(&template.Content),
			EnvContent:           nullableTextPtrKeepEmpty(template.EnvContent),
			IsCustom:             template.IsCustom,
			IsRemote:             template.IsRemote,
			RegistryID:           nullableTextPtrKeepEmpty(template.RegistryID),
			MetaVersion:          nullableTextPtrKeepEmpty(metaVersion),
			MetaAuthor:           nullableTextPtrKeepEmpty(metaAuthor),
			MetaTags:             nullableTextPtrKeepEmpty(metaTags),
			MetaRemoteUrl:        nullableTextPtrKeepEmpty(metaRemoteURL),
			MetaEnvUrl:           nullableTextPtrKeepEmpty(metaEnvURL),
			MetaDocumentationUrl: nullableTextPtrKeepEmpty(metaDocumentationURL),
			UpdatedAt:            nullableTimestamptzPtr(updatedAt),
			ID:                   template.ID,
		})
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapComposeTemplateFromPGValues(
			row.ID, row.Name, row.Description, row.Content, row.EnvContent, row.IsCustom, row.IsRemote, row.RegistryID,
			row.MetaVersion, row.MetaAuthor, row.MetaTags, row.MetaRemoteUrl, row.MetaEnvUrl, row.MetaDocumentationUrl,
			row.CreatedAt, row.UpdatedAt,
		), nil
	case "sqlite":
		row, err := s.sqlite.SaveComposeTemplate(ctx, sqlitedb.SaveComposeTemplateParams{
			Name:                 template.Name,
			Description:          sql.NullString{String: template.Description, Valid: true},
			Content:              sql.NullString{String: template.Content, Valid: true},
			EnvContent:           nullableNullStringPtrKeepEmpty(template.EnvContent),
			IsCustom:             template.IsCustom,
			IsRemote:             template.IsRemote,
			RegistryID:           nullableNullStringPtrKeepEmpty(template.RegistryID),
			MetaVersion:          nullableNullStringPtrKeepEmpty(metaVersion),
			MetaAuthor:           nullableNullStringPtrKeepEmpty(metaAuthor),
			MetaTags:             nullableNullStringPtrKeepEmpty(metaTags),
			MetaRemoteUrl:        nullableNullStringPtrKeepEmpty(metaRemoteURL),
			MetaEnvUrl:           nullableNullStringPtrKeepEmpty(metaEnvURL),
			MetaDocumentationUrl: nullableNullStringPtrKeepEmpty(metaDocumentationURL),
			UpdatedAt:            nullableNullTimePtr(updatedAt),
			ID:                   template.ID,
		})
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapComposeTemplateFromSQLiteValues(
			row.ID, row.Name, row.Description, row.Content, row.EnvContent, row.IsCustom, row.IsRemote, row.RegistryID,
			row.MetaVersion, row.MetaAuthor, row.MetaTags, row.MetaRemoteUrl, row.MetaEnvUrl, row.MetaDocumentationUrl,
			row.CreatedAt, row.UpdatedAt,
		), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) DeleteComposeTemplateByID(ctx context.Context, id string) (bool, error) {
	switch s.driver {
	case "postgres":
		rowsAffected, err := s.pg.DeleteComposeTemplateByID(ctx, id)
		if err != nil {
			return false, err
		}
		return rowsAffected > 0, nil
	case "sqlite":
		rowsAffected, err := s.sqlite.DeleteComposeTemplateByID(ctx, id)
		if err != nil {
			return false, err
		}
		return rowsAffected > 0, nil
	default:
		return false, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) ListTemplateRegistries(ctx context.Context) ([]template.ModelTemplateRegistry, error) {
	switch s.driver {
	case "postgres":
		rows, err := s.pg.ListTemplateRegistries(ctx)
		if err != nil {
			return nil, err
		}
		items := make([]template.ModelTemplateRegistry, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapTemplateRegistryFromPG(row))
		}
		return items, nil
	case "sqlite":
		rows, err := s.sqlite.ListTemplateRegistries(ctx)
		if err != nil {
			return nil, err
		}
		items := make([]template.ModelTemplateRegistry, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapTemplateRegistryFromSQLite(row))
		}
		return items, nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) GetTemplateRegistryByID(ctx context.Context, id string) (*template.ModelTemplateRegistry, error) {
	switch s.driver {
	case "postgres":
		row, err := s.pg.GetTemplateRegistryByID(ctx, id)
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapTemplateRegistryFromPG(row), nil
	case "sqlite":
		row, err := s.sqlite.GetTemplateRegistryByID(ctx, id)
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapTemplateRegistryFromSQLite(row), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) CreateTemplateRegistry(ctx context.Context, registry template.ModelTemplateRegistry) (*template.ModelTemplateRegistry, error) {
	createdAt := registry.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	updatedAt := registry.UpdatedAt
	if updatedAt == nil {
		updatedAt = &createdAt
	}

	switch s.driver {
	case "postgres":
		row, err := s.pg.CreateTemplateRegistry(ctx, pgdb.CreateTemplateRegistryParams{
			ID:          registry.ID,
			Name:        registry.Name,
			Url:         registry.URL,
			Enabled:     registry.Enabled,
			Description: nullableTextPtrKeepEmpty(&registry.Description),
			CreatedAt:   nullableTimestamptz(createdAt),
			UpdatedAt:   nullableTimestamptzPtr(updatedAt),
		})
		if err != nil {
			return nil, err
		}
		return mapTemplateRegistryFromPG(row), nil
	case "sqlite":
		row, err := s.sqlite.CreateTemplateRegistry(ctx, sqlitedb.CreateTemplateRegistryParams{
			ID:          registry.ID,
			Name:        registry.Name,
			Url:         registry.URL,
			Enabled:     registry.Enabled,
			Description: sql.NullString{String: registry.Description, Valid: true},
			CreatedAt:   createdAt,
			UpdatedAt:   nullableNullTimePtr(updatedAt),
		})
		if err != nil {
			return nil, err
		}
		return mapTemplateRegistryFromSQLite(row), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) SaveTemplateRegistry(ctx context.Context, registry template.ModelTemplateRegistry) (*template.ModelTemplateRegistry, error) {
	updatedAt := registry.UpdatedAt
	if updatedAt == nil {
		now := time.Now()
		updatedAt = &now
	}

	switch s.driver {
	case "postgres":
		row, err := s.pg.SaveTemplateRegistry(ctx, pgdb.SaveTemplateRegistryParams{
			Name:        registry.Name,
			Url:         registry.URL,
			Enabled:     registry.Enabled,
			Description: nullableTextPtrKeepEmpty(&registry.Description),
			UpdatedAt:   nullableTimestamptzPtr(updatedAt),
			ID:          registry.ID,
		})
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapTemplateRegistryFromPG(row), nil
	case "sqlite":
		row, err := s.sqlite.SaveTemplateRegistry(ctx, sqlitedb.SaveTemplateRegistryParams{
			Name:        registry.Name,
			Url:         registry.URL,
			Enabled:     registry.Enabled,
			Description: sql.NullString{String: registry.Description, Valid: true},
			UpdatedAt:   nullableNullTimePtr(updatedAt),
			ID:          registry.ID,
		})
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapTemplateRegistryFromSQLite(row), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) DeleteTemplateRegistryByID(ctx context.Context, id string) (bool, error) {
	switch s.driver {
	case "postgres":
		rowsAffected, err := s.pg.DeleteTemplateRegistryByID(ctx, id)
		if err != nil {
			return false, err
		}
		return rowsAffected > 0, nil
	case "sqlite":
		rowsAffected, err := s.sqlite.DeleteTemplateRegistryByID(ctx, id)
		if err != nil {
			return false, err
		}
		return rowsAffected > 0, nil
	default:
		return false, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}
