package migration

// Status represents a database migration status entry.
type Status struct {
	Version   int64  `json:"version" doc:"Migration version"`
	State     string `json:"state" doc:"Current migration state"`
	AppliedAt string `json:"appliedAt,omitempty" doc:"Timestamp when the migration was applied"`
	Path      string `json:"path" doc:"Migration file path"`
}

// DownRequest represents a request to rollback migrations by a number of steps.
type DownRequest struct {
	Steps int `json:"steps" doc:"Number of migrations to rollback"`
}

// DownToRequest represents a request to rollback migrations to a specific version.
type DownToRequest struct {
	Version int64 `json:"version" doc:"Target migration version"`
}
