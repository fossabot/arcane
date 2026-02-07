-- +goose Up
-- +goose StatementBegin
DO $$
BEGIN
  IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'schema_migrations') THEN
    WITH latest AS (
      SELECT COALESCE(MAX(version), 0) AS version
      FROM schema_migrations
      WHERE dirty = false
    ),
    series AS (
      SELECT generate_series(1, (SELECT version FROM latest)) AS version_id
    )
    INSERT INTO goose_db_version (version_id, is_applied)
    SELECT series.version_id, true
    FROM series
    WHERE NOT EXISTS (
      SELECT 1 FROM goose_db_version g WHERE g.version_id = series.version_id
    );

    DROP TABLE schema_migrations;
  END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS goose_db_version;
CREATE TABLE IF NOT EXISTS schema_migrations (
    version bigint NOT NULL,
    dirty boolean NOT NULL
);
-- +goose StatementEnd
