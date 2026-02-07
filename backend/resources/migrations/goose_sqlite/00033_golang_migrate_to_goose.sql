-- +goose Up
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER NOT NULL,
    dirty BOOLEAN NOT NULL
);
WITH RECURSIVE
  max_version AS (
    SELECT COALESCE(MAX(version), 0) AS v
    FROM schema_migrations
    WHERE dirty = 0
  ),
  seq(x) AS (
    SELECT 1 FROM max_version WHERE v >= 1
    UNION ALL
    SELECT x + 1 FROM seq, max_version WHERE x < v
  )
INSERT INTO goose_db_version (version_id, is_applied)
SELECT x, 1
FROM seq
WHERE NOT EXISTS (
  SELECT 1 FROM goose_db_version g WHERE g.version_id = x
);
DROP TABLE IF EXISTS schema_migrations;

-- +goose Down
DROP TABLE IF EXISTS goose_db_version;
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER NOT NULL,
    dirty BOOLEAN NOT NULL
);
