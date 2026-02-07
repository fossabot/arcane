-- +goose Up
BEGIN;
INSERT INTO settings (key, value)
SELECT 'projectsDirectory', value
FROM settings
WHERE key = 'stacksDirectory'
ON CONFLICT (key) DO NOTHING;

DELETE FROM settings WHERE key = 'stacksDirectory';
COMMIT;
-- +goose Down
BEGIN;
INSERT INTO settings (key, value)
SELECT 'stacksDirectory', value
FROM settings
WHERE key = 'projectsDirectory'
ON CONFLICT (key) DO NOTHING;

DELETE FROM settings WHERE key = 'projectsDirectory';
COMMIT;