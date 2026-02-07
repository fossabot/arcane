package stores

import (
	"context"
	"fmt"
	"time"

	"github.com/getarcaneapp/arcane/backend/internal/database/models/pgdb"
	"github.com/getarcaneapp/arcane/backend/internal/database/models/sqlitedb"
	"github.com/getarcaneapp/arcane/types/project"
)

func (s *SqlcStore) CreateProject(ctx context.Context, project project.Project) (*project.Project, error) {
	createdAt := project.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now()
	}
	updatedAt := project.UpdatedAt
	if updatedAt == nil {
		updatedAt = &createdAt
	}

	switch s.driver {
	case "postgres":
		row, err := s.pg.CreateProject(ctx, pgdb.CreateProjectParams{
			ID:              project.ID,
			Name:            project.Name,
			DirName:         nullableTextPtrKeepEmpty(project.DirName),
			Path:            project.Path,
			Status:          string(project.Status),
			ServiceCount:    int32(project.ServiceCount),
			RunningCount:    int32(project.RunningCount),
			StatusReason:    nullableTextPtrKeepEmpty(project.StatusReason),
			GitopsManagedBy: nullableTextPtrKeepEmpty(project.GitOpsManagedBy),
			CreatedAt:       nullableTimestamptz(createdAt),
			UpdatedAt:       nullableTimestamptzPtr(updatedAt),
		})
		if err != nil {
			return nil, err
		}
		return mapProjectFromPG(row), nil
	case "sqlite":
		row, err := s.sqlite.CreateProject(ctx, sqlitedb.CreateProjectParams{
			ID:              project.ID,
			Name:            project.Name,
			DirName:         nullableNullStringPtrKeepEmpty(project.DirName),
			Path:            project.Path,
			Status:          string(project.Status),
			ServiceCount:    int64(project.ServiceCount),
			RunningCount:    int64(project.RunningCount),
			StatusReason:    nullableNullStringPtrKeepEmpty(project.StatusReason),
			GitopsManagedBy: nullableNullStringPtrKeepEmpty(project.GitOpsManagedBy),
			CreatedAt:       createdAt,
			UpdatedAt:       nullableNullTimePtr(updatedAt),
		})
		if err != nil {
			return nil, err
		}
		return mapProjectFromSQLite(row), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) GetProjectByID(ctx context.Context, id string) (*project.Project, error) {
	switch s.driver {
	case "postgres":
		row, err := s.pg.GetProjectByID(ctx, id)
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapProjectFromPG(row), nil
	case "sqlite":
		row, err := s.sqlite.GetProjectByID(ctx, id)
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapProjectFromSQLite(row), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) GetProjectByPathOrDir(ctx context.Context, path string, dirName string) (*project.Project, error) {
	switch s.driver {
	case "postgres":
		row, err := s.pg.GetProjectByPathOrDir(ctx, pgdb.GetProjectByPathOrDirParams{
			Path:    path,
			DirName: nullableText(dirName),
		})
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapProjectFromPG(row), nil
	case "sqlite":
		row, err := s.sqlite.GetProjectByPathOrDir(ctx, sqlitedb.GetProjectByPathOrDirParams{
			Path:    path,
			DirName: nullableString(dirName),
		})
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapProjectFromSQLite(row), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) ListProjects(ctx context.Context) ([]project.Project, error) {
	switch s.driver {
	case "postgres":
		rows, err := s.pg.ListProjects(ctx)
		if err != nil {
			return nil, err
		}
		items := make([]project.Project, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapProjectFromPG(row))
		}
		return items, nil
	case "sqlite":
		rows, err := s.sqlite.ListProjects(ctx)
		if err != nil {
			return nil, err
		}
		items := make([]project.Project, 0, len(rows))
		for _, row := range rows {
			items = append(items, *mapProjectFromSQLite(row))
		}
		return items, nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) SaveProject(ctx context.Context, project project.Project) (*project.Project, error) {
	updatedAt := project.UpdatedAt
	if updatedAt == nil {
		now := time.Now()
		updatedAt = &now
	}

	switch s.driver {
	case "postgres":
		row, err := s.pg.SaveProject(ctx, pgdb.SaveProjectParams{
			Name:            project.Name,
			DirName:         nullableTextPtrKeepEmpty(project.DirName),
			Path:            project.Path,
			Status:          string(project.Status),
			ServiceCount:    int32(project.ServiceCount),
			RunningCount:    int32(project.RunningCount),
			StatusReason:    nullableTextPtrKeepEmpty(project.StatusReason),
			GitopsManagedBy: nullableTextPtrKeepEmpty(project.GitOpsManagedBy),
			UpdatedAt:       nullableTimestamptzPtr(updatedAt),
			ID:              project.ID,
		})
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapProjectFromPG(row), nil
	case "sqlite":
		row, err := s.sqlite.SaveProject(ctx, sqlitedb.SaveProjectParams{
			Name:            project.Name,
			DirName:         nullableNullStringPtrKeepEmpty(project.DirName),
			Path:            project.Path,
			Status:          string(project.Status),
			ServiceCount:    int64(project.ServiceCount),
			RunningCount:    int64(project.RunningCount),
			StatusReason:    nullableNullStringPtrKeepEmpty(project.StatusReason),
			GitopsManagedBy: nullableNullStringPtrKeepEmpty(project.GitOpsManagedBy),
			UpdatedAt:       nullableNullTimePtr(updatedAt),
			ID:              project.ID,
		})
		if err != nil {
			if isNotFound(err) {
				return nil, nil
			}
			return nil, err
		}
		return mapProjectFromSQLite(row), nil
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) DeleteProjectByID(ctx context.Context, id string) (bool, error) {
	switch s.driver {
	case "postgres":
		rowsAffected, err := s.pg.DeleteProjectByID(ctx, id)
		if err != nil {
			return false, err
		}
		return rowsAffected > 0, nil
	case "sqlite":
		rowsAffected, err := s.sqlite.DeleteProjectByID(ctx, id)
		if err != nil {
			return false, err
		}
		return rowsAffected > 0, nil
	default:
		return false, fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) UpdateProjectStatus(ctx context.Context, id string, status project.ProjectStatus, updatedAt time.Time) error {
	switch s.driver {
	case "postgres":
		return s.pg.UpdateProjectStatus(ctx, pgdb.UpdateProjectStatusParams{
			Status:    string(status),
			UpdatedAt: nullableTimestamptz(updatedAt),
			ID:        id,
		})
	case "sqlite":
		return s.sqlite.UpdateProjectStatus(ctx, sqlitedb.UpdateProjectStatusParams{
			Status:    string(status),
			UpdatedAt: nullableNullTime(updatedAt),
			ID:        id,
		})
	default:
		return fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) UpdateProjectStatusAndCounts(ctx context.Context, id string, status project.ProjectStatus, serviceCount int, runningCount int, updatedAt time.Time) error {
	switch s.driver {
	case "postgres":
		return s.pg.UpdateProjectStatusAndCounts(ctx, pgdb.UpdateProjectStatusAndCountsParams{
			Status:       string(status),
			ServiceCount: int32(serviceCount),
			RunningCount: int32(runningCount),
			UpdatedAt:    nullableTimestamptz(updatedAt),
			ID:           id,
		})
	case "sqlite":
		return s.sqlite.UpdateProjectStatusAndCounts(ctx, sqlitedb.UpdateProjectStatusAndCountsParams{
			Status:       string(status),
			ServiceCount: int64(serviceCount),
			RunningCount: int64(runningCount),
			UpdatedAt:    nullableNullTime(updatedAt),
			ID:           id,
		})
	default:
		return fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}

func (s *SqlcStore) UpdateProjectServiceCount(ctx context.Context, id string, serviceCount int) error {
	switch s.driver {
	case "postgres":
		return s.pg.UpdateProjectServiceCount(ctx, pgdb.UpdateProjectServiceCountParams{
			ServiceCount: int32(serviceCount),
			ID:           id,
		})
	case "sqlite":
		return s.sqlite.UpdateProjectServiceCount(ctx, sqlitedb.UpdateProjectServiceCountParams{
			ServiceCount: int64(serviceCount),
			ID:           id,
		})
	default:
		return fmt.Errorf("unsupported database provider: %s", s.driver)
	}
}
