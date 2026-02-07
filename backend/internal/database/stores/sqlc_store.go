package stores

import (
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/getarcaneapp/arcane/backend/internal/database/models/pgdb"
	"github.com/getarcaneapp/arcane/backend/internal/database/models/sqlitedb"
)

type SqlcStore struct {
	driver string
	pg     *pgdb.Queries
	sqlite *sqlitedb.Queries
	pgPool *pgxpool.Pool
	sqlDB  *sql.DB
}

func NewSqlcStore(driver string, pgPool *pgxpool.Pool, sqlDB *sql.DB) (*SqlcStore, error) {
	if driver == "" {
		return nil, fmt.Errorf("database driver is empty")
	}

	store := &SqlcStore{driver: driver, pgPool: pgPool, sqlDB: sqlDB}
	switch driver {
	case "postgres":
		if pgPool == nil {
			return nil, fmt.Errorf("pgx pool is nil")
		}
		store.pg = pgdb.New(pgDebugDBTX{inner: pgPool})
	case "sqlite":
		if sqlDB == nil {
			return nil, fmt.Errorf("sql.DB is nil")
		}
		store.sqlite = sqlitedb.New(sqliteDebugDBTX{inner: sqlDB})
	default:
		return nil, fmt.Errorf("unsupported database provider: %s", driver)
	}

	return store, nil
}
