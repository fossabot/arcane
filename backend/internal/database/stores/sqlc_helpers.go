package stores

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/getarcaneapp/arcane/backend/internal/database/models/pgdb"
	"github.com/getarcaneapp/arcane/backend/internal/database/models/sqlitedb"
	"github.com/getarcaneapp/arcane/types/apikey"
	"github.com/getarcaneapp/arcane/types/base"
	"github.com/getarcaneapp/arcane/types/containerregistry"
	"github.com/getarcaneapp/arcane/types/environment"
	"github.com/getarcaneapp/arcane/types/event"
	"github.com/getarcaneapp/arcane/types/gitops"
	"github.com/getarcaneapp/arcane/types/imageupdate"
	"github.com/getarcaneapp/arcane/types/notification"
	"github.com/getarcaneapp/arcane/types/project"
	"github.com/getarcaneapp/arcane/types/template"
)

func isNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows) || errors.Is(err, pgx.ErrNoRows)
}

func boolFromPgBool(value pgtype.Bool) bool {
	if !value.Valid {
		return false
	}
	return value.Bool
}

func boolToPgBool(value bool) pgtype.Bool {
	return pgtype.Bool{Bool: value, Valid: true}
}

func stringFromPgText(value pgtype.Text) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func timeFromPgTimestamp(value pgtype.Timestamp) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time
}

func timeToPgTimestamp(value time.Time) pgtype.Timestamp {
	if value.IsZero() {
		return pgtype.Timestamp{}
	}
	return pgtype.Timestamp{Time: value, Valid: true}
}

func timeFromPgTimestamptz(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time
}

func timePtrFromPgTimestamptz(value pgtype.Timestamptz) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time
	return &t
}

func nullTimeOrZero(value sql.NullTime) time.Time {
	if value.Valid {
		return value.Time
	}
	return time.Time{}
}

func timePtrFromNull(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time
	return &t
}

func stringFromNull(value sql.NullString) string {
	if value.Valid {
		return value.String
	}
	return ""
}

func stringPtrFromNull(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	v := value.String
	return &v
}

func stringPtrFromPgText(value pgtype.Text) *string {
	if !value.Valid {
		return nil
	}
	v := value.String
	return &v
}

func nullableText(value string) pgtype.Text {
	if value == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: value, Valid: true}
}

func nullableTextPtr(value *string) pgtype.Text {
	if value == nil || *value == "" {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *value, Valid: true}
}

func nullableTextPtrKeepEmpty(value *string) pgtype.Text {
	if value == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *value, Valid: true}
}

func nullableString(value string) sql.NullString {
	if value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func nullableStringPtr(value *string) sql.NullString {
	if value == nil || *value == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: *value, Valid: true}
}

func nullableNullStringPtrKeepEmpty(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *value, Valid: true}
}

func nullableTimestamptzPtr(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: *value, Valid: true}
}

func nullableTimestamptz(value time.Time) pgtype.Timestamptz {
	if value.IsZero() {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{Time: value, Valid: true}
}

func nullableNullTimePtr(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *value, Valid: true}
}

func nullableNullTime(value time.Time) sql.NullTime {
	if value.IsZero() {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: value, Valid: true}
}

func boolToNullInt(value bool) sql.NullInt64 {
	if value {
		return sql.NullInt64{Int64: 1, Valid: true}
	}
	return sql.NullInt64{Int64: 0, Valid: true}
}

func nullIntToBool(value sql.NullInt64) bool {
	if !value.Valid {
		return false
	}
	return value.Int64 != 0
}

func boolToInt64(value bool) int64 {
	if value {
		return 1
	}
	return 0
}

func templateMetadataToColumns(metadata *template.ComposeTemplateMetadata) (version, author, tags, remoteURL, envURL, documentationURL *string) {
	if metadata == nil {
		return nil, nil, nil, nil, nil, nil
	}
	version = metadata.Version
	author = metadata.Author
	remoteURL = metadata.RemoteURL
	envURL = metadata.EnvURL
	documentationURL = metadata.DocumentationURL
	if len(metadata.Tags) > 0 {
		encoded, err := json.Marshal(metadata.Tags)
		if err == nil {
			value := string(encoded)
			tags = &value
		}
	}
	return version, author, tags, remoteURL, envURL, documentationURL
}

func parseTemplateTags(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	var tags []string
	if err := json.Unmarshal([]byte(trimmed), &tags); err == nil {
		return tags
	}
	parts := strings.Split(trimmed, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func composeTemplateMetadataFromColumns(version, author, tags, remoteURL, envURL, documentationURL *string) *template.ComposeTemplateMetadata {
	parsedTags := []string(nil)
	if tags != nil {
		parsedTags = parseTemplateTags(*tags)
	}
	if version == nil && author == nil && remoteURL == nil && envURL == nil && documentationURL == nil && len(parsedTags) == 0 {
		return nil
	}
	return &template.ComposeTemplateMetadata{
		Version:          version,
		Author:           author,
		Tags:             parsedTags,
		RemoteURL:        remoteURL,
		EnvURL:           envURL,
		DocumentationURL: documentationURL,
	}
}

func mapApiKeyFromPGValues(
	id string,
	name string,
	description pgtype.Text,
	keyHash string,
	keyPrefix string,
	userID string,
	environmentID pgtype.Text,
	expiresAt pgtype.Timestamptz,
	lastUsedAt pgtype.Timestamptz,
	createdAt pgtype.Timestamptz,
	updatedAt pgtype.Timestamptz,
) *apikey.ModelApiKey {
	return &apikey.ModelApiKey{
		BaseModel: base.BaseModel{
			ID:        id,
			CreatedAt: timeFromPgTimestamptz(createdAt),
			UpdatedAt: timePtrFromPgTimestamptz(updatedAt),
		},
		Name:          name,
		Description:   stringPtrFromPgText(description),
		KeyHash:       keyHash,
		KeyPrefix:     keyPrefix,
		UserID:        userID,
		EnvironmentID: stringPtrFromPgText(environmentID),
		ExpiresAt:     timePtrFromPgTimestamptz(expiresAt),
		LastUsedAt:    timePtrFromPgTimestamptz(lastUsedAt),
	}
}

func mapApiKeyFromSQLiteValues(
	id string,
	name string,
	description sql.NullString,
	keyHash string,
	keyPrefix string,
	userID string,
	environmentID sql.NullString,
	expiresAt sql.NullTime,
	lastUsedAt sql.NullTime,
	createdAt time.Time,
	updatedAt sql.NullTime,
) *apikey.ModelApiKey {
	return &apikey.ModelApiKey{
		BaseModel: base.BaseModel{
			ID:        id,
			CreatedAt: createdAt,
			UpdatedAt: timePtrFromNull(updatedAt),
		},
		Name:          name,
		Description:   stringPtrFromNull(description),
		KeyHash:       keyHash,
		KeyPrefix:     keyPrefix,
		UserID:        userID,
		EnvironmentID: stringPtrFromNull(environmentID),
		ExpiresAt:     timePtrFromNull(expiresAt),
		LastUsedAt:    timePtrFromNull(lastUsedAt),
	}
}

func mapContainerRegistryFromPG(row *pgdb.ContainerRegistry) *containerregistry.ModelContainerRegistry {
	if row == nil {
		return nil
	}
	return &containerregistry.ModelContainerRegistry{
		BaseModel: base.BaseModel{
			ID:        row.ID,
			CreatedAt: timeFromPgTimestamptz(row.CreatedAt),
			UpdatedAt: timePtrFromPgTimestamptz(row.UpdatedAt),
		},
		URL:         row.Url,
		Username:    row.Username,
		Token:       row.Token,
		Description: stringPtrFromPgText(row.Description),
		Insecure:    row.Insecure,
		Enabled:     row.Enabled,
		CreatedAt:   timeFromPgTimestamptz(row.CreatedAt),
		UpdatedAt:   timeFromPgTimestamptz(row.UpdatedAt),
	}
}

func mapContainerRegistryFromSQLite(row *sqlitedb.ContainerRegistry) *containerregistry.ModelContainerRegistry {
	if row == nil {
		return nil
	}
	updatedAt := row.CreatedAt
	if row.UpdatedAt.Valid {
		updatedAt = row.UpdatedAt.Time
	}
	return &containerregistry.ModelContainerRegistry{
		BaseModel: base.BaseModel{
			ID:        row.ID,
			CreatedAt: row.CreatedAt,
			UpdatedAt: timePtrFromNull(row.UpdatedAt),
		},
		URL:         row.Url,
		Username:    row.Username,
		Token:       row.Token,
		Description: stringPtrFromNull(row.Description),
		Insecure:    row.Insecure,
		Enabled:     row.Enabled,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   updatedAt,
	}
}

func mapEnvironmentFromPG(row *pgdb.Environment) *environment.ModelEnvironment {
	if row == nil {
		return nil
	}
	return &environment.ModelEnvironment{
		Name:        stringFromPgText(row.Name),
		ApiUrl:      row.ApiUrl,
		Status:      row.Status,
		Enabled:     row.Enabled,
		IsEdge:      row.IsEdge,
		LastSeen:    timePtrFromPgTimestamptz(row.LastSeen),
		AccessToken: stringPtrFromPgText(row.AccessToken),
		ApiKeyID:    stringPtrFromPgText(row.ApiKeyID),
		BaseModel: base.BaseModel{
			ID:        row.ID,
			CreatedAt: timeFromPgTimestamptz(row.CreatedAt),
			UpdatedAt: timePtrFromPgTimestamptz(row.UpdatedAt),
		},
	}
}

func mapEnvironmentFromSQLite(row *sqlitedb.Environment) *environment.ModelEnvironment {
	if row == nil {
		return nil
	}
	return &environment.ModelEnvironment{
		Name:        stringFromNull(row.Name),
		ApiUrl:      row.ApiUrl,
		Status:      row.Status,
		Enabled:     row.Enabled,
		IsEdge:      row.IsEdge != 0,
		LastSeen:    timePtrFromNull(row.LastSeen),
		AccessToken: stringPtrFromNull(row.AccessToken),
		ApiKeyID:    stringPtrFromNull(row.ApiKeyID),
		BaseModel: base.BaseModel{
			ID:        row.ID,
			CreatedAt: row.CreatedAt,
			UpdatedAt: timePtrFromNull(row.UpdatedAt),
		},
	}
}

func mapEventFromPG(row *pgdb.Event) *event.ModelEvent {
	if row == nil {
		return nil
	}
	return &event.ModelEvent{
		Type:          event.EventType(row.Type),
		Severity:      event.EventSeverity(row.Severity),
		Title:         row.Title,
		Description:   stringFromPgText(row.Description),
		ResourceType:  stringPtrFromPgText(row.ResourceType),
		ResourceID:    stringPtrFromPgText(row.ResourceID),
		ResourceName:  stringPtrFromPgText(row.ResourceName),
		UserID:        stringPtrFromPgText(row.UserID),
		Username:      stringPtrFromPgText(row.Username),
		EnvironmentID: stringPtrFromPgText(row.EnvironmentID),
		Metadata:      row.Metadata,
		Timestamp:     timeFromPgTimestamptz(row.Timestamp),
		BaseModel: base.BaseModel{
			ID:        row.ID,
			CreatedAt: timeFromPgTimestamptz(row.CreatedAt),
			UpdatedAt: timePtrFromPgTimestamptz(row.UpdatedAt),
		},
	}
}

func mapEventFromSQLite(row *sqlitedb.Event) *event.ModelEvent {
	if row == nil {
		return nil
	}
	return &event.ModelEvent{
		Type:          event.EventType(row.Type),
		Severity:      event.EventSeverity(row.Severity),
		Title:         row.Title,
		Description:   stringFromNull(row.Description),
		ResourceType:  stringPtrFromNull(row.ResourceType),
		ResourceID:    stringPtrFromNull(row.ResourceID),
		ResourceName:  stringPtrFromNull(row.ResourceName),
		UserID:        stringPtrFromNull(row.UserID),
		Username:      stringPtrFromNull(row.Username),
		EnvironmentID: stringPtrFromNull(row.EnvironmentID),
		Metadata:      row.Metadata,
		Timestamp:     row.Timestamp,
		BaseModel: base.BaseModel{
			ID:        row.ID,
			CreatedAt: row.CreatedAt,
			UpdatedAt: timePtrFromNull(row.UpdatedAt),
		},
	}
}

func mapImageUpdateFromPGValues(
	id string,
	repository string,
	tag string,
	hasUpdate bool,
	updateType pgtype.Text,
	currentVersion string,
	latestVersion pgtype.Text,
	currentDigest pgtype.Text,
	latestDigest pgtype.Text,
	checkTime pgtype.Timestamptz,
	responseTimeMs int32,
	lastError pgtype.Text,
	authMethod pgtype.Text,
	authUsername pgtype.Text,
	authRegistry pgtype.Text,
	usedCredential pgtype.Bool,
	notificationSent pgtype.Bool,
	createdAt pgtype.Timestamptz,
	updatedAt pgtype.Timestamptz,
) *imageupdate.ImageUpdateRecord {
	return &imageupdate.ImageUpdateRecord{
		ID:               id,
		Repository:       repository,
		Tag:              tag,
		HasUpdate:        hasUpdate,
		UpdateType:       stringFromPgText(updateType),
		CurrentVersion:   currentVersion,
		LatestVersion:    stringPtrFromPgText(latestVersion),
		CurrentDigest:    stringPtrFromPgText(currentDigest),
		LatestDigest:     stringPtrFromPgText(latestDigest),
		CheckTime:        timeFromPgTimestamptz(checkTime),
		ResponseTimeMs:   int(responseTimeMs),
		LastError:        stringPtrFromPgText(lastError),
		AuthMethod:       stringPtrFromPgText(authMethod),
		AuthUsername:     stringPtrFromPgText(authUsername),
		AuthRegistry:     stringPtrFromPgText(authRegistry),
		UsedCredential:   boolFromPgBool(usedCredential),
		NotificationSent: boolFromPgBool(notificationSent),
		BaseModel: base.BaseModel{
			ID:        id,
			CreatedAt: timeFromPgTimestamptz(createdAt),
			UpdatedAt: timePtrFromPgTimestamptz(updatedAt),
		},
	}
}

func mapImageUpdateFromSQLiteValues(
	id string,
	repository string,
	tag string,
	hasUpdate bool,
	updateType sql.NullString,
	currentVersion string,
	latestVersion sql.NullString,
	currentDigest sql.NullString,
	latestDigest sql.NullString,
	checkTime time.Time,
	responseTimeMs int64,
	lastError sql.NullString,
	authMethod sql.NullString,
	authUsername sql.NullString,
	authRegistry sql.NullString,
	usedCredential sql.NullInt64,
	notificationSent sql.NullInt64,
	createdAt time.Time,
	updatedAt sql.NullTime,
) *imageupdate.ImageUpdateRecord {
	return &imageupdate.ImageUpdateRecord{
		ID:               id,
		Repository:       repository,
		Tag:              tag,
		HasUpdate:        hasUpdate,
		UpdateType:       stringFromNull(updateType),
		CurrentVersion:   currentVersion,
		LatestVersion:    stringPtrFromNull(latestVersion),
		CurrentDigest:    stringPtrFromNull(currentDigest),
		LatestDigest:     stringPtrFromNull(latestDigest),
		CheckTime:        checkTime,
		ResponseTimeMs:   int(responseTimeMs),
		LastError:        stringPtrFromNull(lastError),
		AuthMethod:       stringPtrFromNull(authMethod),
		AuthUsername:     stringPtrFromNull(authUsername),
		AuthRegistry:     stringPtrFromNull(authRegistry),
		UsedCredential:   nullIntToBool(usedCredential),
		NotificationSent: nullIntToBool(notificationSent),
		BaseModel: base.BaseModel{
			ID:        id,
			CreatedAt: createdAt,
			UpdatedAt: timePtrFromNull(updatedAt),
		},
	}
}

func mapComposeTemplateFromPGValues(
	id string,
	name string,
	description pgtype.Text,
	content pgtype.Text,
	envContent pgtype.Text,
	isCustom bool,
	isRemote bool,
	registryID pgtype.Text,
	metaVersion pgtype.Text,
	metaAuthor pgtype.Text,
	metaTags pgtype.Text,
	metaRemoteURL pgtype.Text,
	metaEnvURL pgtype.Text,
	metaDocumentationURL pgtype.Text,
	createdAt pgtype.Timestamptz,
	updatedAt pgtype.Timestamptz,
) *template.ComposeTemplate {
	metadata := composeTemplateMetadataFromColumns(
		stringPtrFromPgText(metaVersion),
		stringPtrFromPgText(metaAuthor),
		stringPtrFromPgText(metaTags),
		stringPtrFromPgText(metaRemoteURL),
		stringPtrFromPgText(metaEnvURL),
		stringPtrFromPgText(metaDocumentationURL),
	)

	return &template.ComposeTemplate{
		Name:        name,
		Description: stringFromPgText(description),
		Content:     stringFromPgText(content),
		EnvContent:  stringPtrFromPgText(envContent),
		IsCustom:    isCustom,
		IsRemote:    isRemote,
		RegistryID:  stringPtrFromPgText(registryID),
		Metadata:    metadata,
		BaseModel: base.BaseModel{
			ID:        id,
			CreatedAt: timeFromPgTimestamptz(createdAt),
			UpdatedAt: timePtrFromPgTimestamptz(updatedAt),
		},
	}
}

func mapComposeTemplateFromSQLiteValues(
	id string,
	name string,
	description sql.NullString,
	content sql.NullString,
	envContent sql.NullString,
	isCustom bool,
	isRemote bool,
	registryID sql.NullString,
	metaVersion sql.NullString,
	metaAuthor sql.NullString,
	metaTags sql.NullString,
	metaRemoteURL sql.NullString,
	metaEnvURL sql.NullString,
	metaDocumentationURL sql.NullString,
	createdAt time.Time,
	updatedAt sql.NullTime,
) *template.ComposeTemplate {
	metadata := composeTemplateMetadataFromColumns(
		stringPtrFromNull(metaVersion),
		stringPtrFromNull(metaAuthor),
		stringPtrFromNull(metaTags),
		stringPtrFromNull(metaRemoteURL),
		stringPtrFromNull(metaEnvURL),
		stringPtrFromNull(metaDocumentationURL),
	)

	return &template.ComposeTemplate{
		Name:        name,
		Description: stringFromNull(description),
		Content:     stringFromNull(content),
		EnvContent:  stringPtrFromNull(envContent),
		IsCustom:    isCustom,
		IsRemote:    isRemote,
		RegistryID:  stringPtrFromNull(registryID),
		Metadata:    metadata,
		BaseModel: base.BaseModel{
			ID:        id,
			CreatedAt: createdAt,
			UpdatedAt: timePtrFromNull(updatedAt),
		},
	}
}

func mapTemplateRegistryFromPG(row *pgdb.TemplateRegistry) *template.ModelTemplateRegistry {
	if row == nil {
		return nil
	}
	return &template.ModelTemplateRegistry{
		Name:        row.Name,
		URL:         row.Url,
		Enabled:     row.Enabled,
		Description: stringFromPgText(row.Description),
		BaseModel: base.BaseModel{
			ID:        row.ID,
			CreatedAt: timeFromPgTimestamptz(row.CreatedAt),
			UpdatedAt: timePtrFromPgTimestamptz(row.UpdatedAt),
		},
	}
}

func mapTemplateRegistryFromSQLite(row *sqlitedb.TemplateRegistry) *template.ModelTemplateRegistry {
	if row == nil {
		return nil
	}
	return &template.ModelTemplateRegistry{
		Name:        row.Name,
		URL:         row.Url,
		Enabled:     row.Enabled,
		Description: stringFromNull(row.Description),
		BaseModel: base.BaseModel{
			ID:        row.ID,
			CreatedAt: row.CreatedAt,
			UpdatedAt: timePtrFromNull(row.UpdatedAt),
		},
	}
}

func mapProjectFromPG(row *pgdb.Project) *project.Project {
	if row == nil {
		return nil
	}
	return &project.Project{
		Name:            row.Name,
		DirName:         stringPtrFromPgText(row.DirName),
		Path:            row.Path,
		Status:          project.ProjectStatus(row.Status),
		StatusReason:    stringPtrFromPgText(row.StatusReason),
		ServiceCount:    int(row.ServiceCount),
		RunningCount:    int(row.RunningCount),
		GitOpsManagedBy: stringPtrFromPgText(row.GitopsManagedBy),
		BaseModel: base.BaseModel{
			ID:        row.ID,
			CreatedAt: timeFromPgTimestamptz(row.CreatedAt),
			UpdatedAt: timePtrFromPgTimestamptz(row.UpdatedAt),
		},
	}
}

func mapProjectFromSQLite(row *sqlitedb.Project) *project.Project {
	if row == nil {
		return nil
	}
	return &project.Project{
		Name:            row.Name,
		DirName:         stringPtrFromNull(row.DirName),
		Path:            row.Path,
		Status:          project.ProjectStatus(row.Status),
		StatusReason:    stringPtrFromNull(row.StatusReason),
		ServiceCount:    int(row.ServiceCount),
		RunningCount:    int(row.RunningCount),
		GitOpsManagedBy: stringPtrFromNull(row.GitopsManagedBy),
		BaseModel: base.BaseModel{
			ID:        row.ID,
			CreatedAt: row.CreatedAt,
			UpdatedAt: timePtrFromNull(row.UpdatedAt),
		},
	}
}

func mapGitRepositoryFromPG(row *pgdb.GitRepository) *gitops.ModelGitRepository {
	if row == nil {
		return nil
	}
	return &gitops.ModelGitRepository{
		Name:                   row.Name,
		URL:                    row.Url,
		AuthType:               row.AuthType,
		Username:               stringFromPgText(row.Username),
		Token:                  stringFromPgText(row.Token),
		SSHKey:                 stringFromPgText(row.SshKey),
		SSHHostKeyVerification: row.SshHostKeyVerification,
		Description:            stringPtrFromPgText(row.Description),
		Enabled:                row.Enabled,
		BaseModel: base.BaseModel{
			ID:        row.ID,
			CreatedAt: timeFromPgTimestamptz(row.CreatedAt),
			UpdatedAt: timePtrFromPgTimestamptz(row.UpdatedAt),
		},
	}
}

func mapGitRepositoryFromSQLite(row *sqlitedb.GitRepository) *gitops.ModelGitRepository {
	if row == nil {
		return nil
	}
	return &gitops.ModelGitRepository{
		Name:                   row.Name,
		URL:                    row.Url,
		AuthType:               row.AuthType,
		Username:               stringFromNull(row.Username),
		Token:                  stringFromNull(row.Token),
		SSHKey:                 stringFromNull(row.SshKey),
		SSHHostKeyVerification: row.SshHostKeyVerification,
		Description:            stringPtrFromNull(row.Description),
		Enabled:                row.Enabled != 0,
		BaseModel: base.BaseModel{
			ID:        row.ID,
			CreatedAt: row.CreatedAt,
			UpdatedAt: timePtrFromNull(row.UpdatedAt),
		},
	}
}

func mapGitOpsSyncFromPG(row *pgdb.GitopsSync) *gitops.ModelGitOpsSync {
	if row == nil {
		return nil
	}
	return &gitops.ModelGitOpsSync{
		Name:           row.Name,
		EnvironmentID:  row.EnvironmentID,
		RepositoryID:   row.RepositoryID,
		Branch:         row.Branch,
		ComposePath:    row.ComposePath,
		ProjectName:    row.ProjectName,
		ProjectID:      stringPtrFromPgText(row.ProjectID),
		AutoSync:       row.AutoSync,
		SyncInterval:   int(row.SyncInterval),
		LastSyncAt:     timePtrFromPgTimestamptz(row.LastSyncAt),
		LastSyncStatus: stringPtrFromPgText(row.LastSyncStatus),
		LastSyncError:  stringPtrFromPgText(row.LastSyncError),
		LastSyncCommit: stringPtrFromPgText(row.LastSyncCommit),
		BaseModel: base.BaseModel{
			ID:        row.ID,
			CreatedAt: timeFromPgTimestamptz(row.CreatedAt),
			UpdatedAt: timePtrFromPgTimestamptz(row.UpdatedAt),
		},
	}
}

func mapGitOpsSyncFromSQLite(row *sqlitedb.GitopsSync) *gitops.ModelGitOpsSync {
	if row == nil {
		return nil
	}
	updatedAt := row.UpdatedAt
	return &gitops.ModelGitOpsSync{
		Name:           row.Name,
		EnvironmentID:  row.EnvironmentID,
		RepositoryID:   row.RepositoryID,
		Branch:         row.Branch,
		ComposePath:    row.ComposePath,
		ProjectName:    row.ProjectName,
		ProjectID:      stringPtrFromNull(row.ProjectID),
		AutoSync:       row.AutoSync,
		SyncInterval:   int(row.SyncInterval),
		LastSyncAt:     timePtrFromNull(row.LastSyncAt),
		LastSyncStatus: stringPtrFromNull(row.LastSyncStatus),
		LastSyncError:  stringPtrFromNull(row.LastSyncError),
		LastSyncCommit: stringPtrFromNull(row.LastSyncCommit),
		BaseModel: base.BaseModel{
			ID:        row.ID,
			CreatedAt: row.CreatedAt,
			UpdatedAt: &updatedAt,
		},
	}
}

func mapAppriseSettingFromPG(row *pgdb.AppriseSetting) *notification.AppriseSettings {
	if row == nil {
		return nil
	}
	return &notification.AppriseSettings{
		ID:                 uint(row.ID),
		APIURL:             row.ApiUrl,
		Enabled:            boolFromPgBool(row.Enabled),
		ImageUpdateTag:     stringFromPgText(row.ImageUpdateTag),
		ContainerUpdateTag: stringFromPgText(row.ContainerUpdateTag),
		CreatedAt:          timeFromPgTimestamp(row.CreatedAt),
		UpdatedAt:          timeFromPgTimestamp(row.UpdatedAt),
	}
}

func mapAppriseSettingFromSQLite(row *sqlitedb.AppriseSetting) *notification.AppriseSettings {
	if row == nil {
		return nil
	}
	return &notification.AppriseSettings{
		ID:                 uint(row.ID),
		APIURL:             row.ApiUrl,
		Enabled:            nullIntToBool(row.Enabled),
		ImageUpdateTag:     stringFromNull(row.ImageUpdateTag),
		ContainerUpdateTag: stringFromNull(row.ContainerUpdateTag),
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
}

func mapNotificationSettingFromPG(row *pgdb.NotificationSetting) notification.NotificationSettings {
	if row == nil {
		return notification.NotificationSettings{}
	}
	return notification.NotificationSettings{
		ID:        uint(row.ID),
		Provider:  notification.NotificationProvider(row.Provider),
		Enabled:   boolFromPgBool(row.Enabled),
		Config:    row.Config,
		CreatedAt: timeFromPgTimestamp(row.CreatedAt),
		UpdatedAt: timeFromPgTimestamp(row.UpdatedAt),
	}
}

func mapNotificationSettingFromSQLite(row *sqlitedb.NotificationSetting) notification.NotificationSettings {
	if row == nil {
		return notification.NotificationSettings{}
	}
	return notification.NotificationSettings{
		ID:        uint(row.ID),
		Provider:  notification.NotificationProvider(row.Provider),
		Enabled:   row.Enabled.Valid && row.Enabled.Bool,
		Config:    row.Config,
		CreatedAt: nullTimeOrZero(row.CreatedAt),
		UpdatedAt: nullTimeOrZero(row.UpdatedAt),
	}
}
