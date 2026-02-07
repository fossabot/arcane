package stores

import (
	"database/sql"

	"github.com/getarcaneapp/arcane/backend/internal/database/models/pgdb"
	"github.com/getarcaneapp/arcane/backend/internal/database/models/sqlitedb"
	"github.com/jackc/pgx/v5"
)

func (s *SqlcStore) withPgTx(tx pgx.Tx) *SqlcStore {
	return &SqlcStore{
		driver: s.driver,
		pg:     pgdb.New(pgDebugDBTX{inner: tx}),
		pgPool: s.pgPool,
		sqlDB:  s.sqlDB,
	}
}

func (s *SqlcStore) withSQLiteTx(tx *sql.Tx) *SqlcStore {
	return &SqlcStore{
		driver: s.driver,
		sqlite: sqlitedb.New(sqliteDebugDBTX{inner: tx}),
		pgPool: s.pgPool,
		sqlDB:  s.sqlDB,
	}
}
