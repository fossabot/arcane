# GORM to sqlc + Goose Migration (Completed)

## Summary

Arcane's backend data layer now uses:

- `sqlc` for typed query generation
- `Goose` for schema migrations (up/down/status/redo support)
- `Store` interfaces + sqlc-backed store implementations for all service DB access

GORM has been removed from the active data path.

## What Changed

1. Migration engine:
Each migration is now a single Goose file with `-- +goose Up` and `-- +goose Down`.
SQLite and PostgreSQL migrations live in:

- `backend/resources/migrations/goose_sqlite/`
- `backend/resources/migrations/goose_postgres/`

2. Query/model generation:
`sqlc` config is in `backend/sqlc.yaml`, generating engine-specific packages:

- `backend/internal/database/models/pgdb/`
- `backend/internal/database/models/sqlitedb/`

3. Service data access:
Services use store interfaces from `backend/internal/database/stores/` and sqlc-generated query code, not GORM models.

4. Shared types:
Shared domain types live in `types/<domain>/` packages and are imported directly.
`types/models` stubs are not used.

## Existing Database Compatibility

Existing databases are preserved by the Goose transition migration:

- `00033_golang_migrate_to_goose.sql`

This migration and bootstrap seeding logic copy legacy `schema_migrations` progress into `goose_db_version` so old installs do not replay historical migrations.

## Caution / Operational Notes

1. Always take a DB backup before first startup on a production instance after upgrading.
2. If a database is already `dirty` or has manually-edited migration history, resolve that first.
3. Keep SQLite and PostgreSQL migration numbers aligned.
4. Keep SQLite and PostgreSQL query behavior aligned unless a dialect difference is intentional.

## Contributor Workflow (Database Changes)

1. Schema changes:
Add matching Goose migration files for both engines with the same version number.

2. Query changes:
Update sqlc query files in both directories:

- `backend/internal/database/queries/postgres/`
- `backend/internal/database/queries/sqlite/`

3. Regenerate code:

```bash
just sqlc
```

4. Validate:

```bash
cd backend && go test ./...
just test e2e
```

5. Commit generated code:
Commit sqlc output under `backend/internal/database/models/` with your query/schema changes.

## Useful Commands

```bash
# Generate sqlc models/queries
just sqlc

# Migration status
cd backend && go run ./cmd/main.go migrate status

# Roll back one migration
cd backend && go run ./cmd/main.go migrate down 1

# Roll back to a specific version
cd backend && go run ./cmd/main.go migrate down-to 32

# Redo latest migration
cd backend && go run ./cmd/main.go migrate redo
```
