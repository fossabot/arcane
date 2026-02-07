package base

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

// BaseModel contains common fields for persisted entities.
type BaseModel struct {
	ID        string     `json:"id"`
	CreatedAt time.Time  `json:"createdAt" sortable:"true"`
	UpdatedAt *time.Time `json:"updatedAt,omitempty"`
}

// JSON is the canonical JSON object type for persisted models.
type JSON = JsonObject

// StringSlice is a JSON-encoded string array type for database columns.
//
// nolint:recvcheck
type StringSlice []string

func (s StringSlice) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, s)
	case string:
		return json.Unmarshal([]byte(v), s)
	default:
		return json.Unmarshal(nil, s)
	}
}
