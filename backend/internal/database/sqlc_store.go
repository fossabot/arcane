package database

import (
	"fmt"

	"github.com/getarcaneapp/arcane/backend/internal/database/stores"
)

type SqlcStore = stores.SqlcStore

func NewSqlcStore(db *DB) (*SqlcStore, error) {
	if db == nil {
		return nil, fmt.Errorf("database is nil")
	}
	provider, err := db.resolveProvider()
	if err != nil {
		return nil, err
	}
	return stores.NewSqlcStore(provider, db.pgPool, db.sqlDB)
}
