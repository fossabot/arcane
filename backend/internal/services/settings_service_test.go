package services

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/getarcaneapp/arcane/backend/internal/config"
	"github.com/getarcaneapp/arcane/backend/internal/database"
	"github.com/getarcaneapp/arcane/types/settings"
)

func setupSettingsTestStore(t *testing.T) (database.SettingsStore, func()) {
	t.Helper()
	ctx := context.Background()
	db, err := database.Initialize(ctx, testSettingsSQLiteDSN(t))
	require.NoError(t, err)
	store, err := database.NewSqlcStore(db)
	require.NoError(t, err)
	return store, func() { _ = db.Close() }
}

func testSettingsSQLiteDSN(t *testing.T) string {
	t.Helper()
	name := strings.ReplaceAll(t.Name(), "/", "_")
	return fmt.Sprintf("file:%s?mode=memory&cache=shared", name)
}

func setupSettingsService(t *testing.T) (context.Context, database.SettingsStore, *SettingsService, func()) {
	t.Helper()
	ctx := context.Background()
	store, cleanup := setupSettingsTestStore(t)
	svc, err := NewSettingsService(ctx, store)
	require.NoError(t, err)
	return ctx, store, svc, cleanup
}

func TestSettingsService_EnsureDefaultSettings_Idempotent(t *testing.T) {
	ctx := context.Background()
	store, cleanup := setupSettingsTestStore(t)
	defer cleanup()
	svc, err := NewSettingsService(ctx, store)
	require.NoError(t, err)

	require.NoError(t, svc.EnsureDefaultSettings(ctx))

	settings1, err := store.ListSettings(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, settings1)

	require.NoError(t, svc.EnsureDefaultSettings(ctx))

	settings2, err := store.ListSettings(ctx)
	require.NoError(t, err)
	require.Equal(t, len(settings1), len(settings2))

	// Spot-check a couple keys exist
	for _, key := range []string{"authLocalEnabled", "projectsDirectory"} {
		sv, err := store.GetSetting(ctx, key)
		require.NoErrorf(t, err, "missing default key %s", key)
		require.NotNil(t, sv)
	}
}

func TestSettingsService_GetSettings_UnknownKeysIgnored(t *testing.T) {
	ctx := context.Background()
	store, cleanup := setupSettingsTestStore(t)
	defer cleanup()
	svc, err := NewSettingsService(ctx, store)
	require.NoError(t, err)

	require.NoError(t, store.UpsertSetting(ctx, "someUnknownKey", "x"))

	_, err = svc.GetSettings(ctx)
	require.NoError(t, err)
}

func TestSettingsService_PruneUnknownSettings_RemovesStaleKeys(t *testing.T) {
	ctx := context.Background()
	store, cleanup := setupSettingsTestStore(t)
	defer cleanup()
	svc, err := NewSettingsService(ctx, store)
	require.NoError(t, err)

	require.NoError(t, svc.UpdateSetting(ctx, "projectsDirectory", "/tmp/projects"))
	require.NoError(t, svc.UpdateSetting(ctx, "encryptionKey", "test-encryption-key"))
	require.NoError(t, svc.UpdateSetting(ctx, "unknownKey", "value"))

	require.NoError(t, svc.PruneUnknownSettings(ctx))

	sv, err := store.GetSetting(ctx, "unknownKey")
	require.NoError(t, err)
	require.Nil(t, sv)

	sv2, err := store.GetSetting(ctx, "projectsDirectory")
	require.NoError(t, err)
	require.NotNil(t, sv2)

	sv3, err := store.GetSetting(ctx, "encryptionKey")
	require.NoError(t, err)
	require.NotNil(t, sv3)
}

func TestSettingsService_GetSettings_EnvOverride_OidcMergeAccounts(t *testing.T) {
	ctx := context.Background()
	store, cleanup := setupSettingsTestStore(t)
	defer cleanup()

	svc, err := NewSettingsService(ctx, store)
	require.NoError(t, err)
	require.NoError(t, svc.EnsureDefaultSettings(ctx))

	// Default in DB is false
	settings1, err := svc.GetSettings(ctx)
	require.NoError(t, err)
	require.False(t, settings1.OidcMergeAccounts.IsTrue())

	// Env override should take precedence
	t.Setenv("OIDC_MERGE_ACCOUNTS", "true")
	settings2, err := svc.GetSettings(ctx)
	require.NoError(t, err)
	require.True(t, settings2.OidcMergeAccounts.IsTrue())
}

func TestSettingsService_GetSetHelpers(t *testing.T) {
	ctx, _, svc, cleanup := setupSettingsService(t)
	defer cleanup()

	// Defaults for missing keys
	require.True(t, svc.GetBoolSetting(ctx, "nonexistentBool", true))
	require.Equal(t, 42, svc.GetIntSetting(ctx, "nonexistentInt", 42))
	require.Equal(t, "def", svc.GetStringSetting(ctx, "nonexistentStr", "def"))

	// Set and read back
	require.NoError(t, svc.SetBoolSetting(ctx, "enableGravatar", true))
	require.True(t, svc.GetBoolSetting(ctx, "enableGravatar", false))

	require.NoError(t, svc.SetIntSetting(ctx, "authSessionTimeout", 123))
	require.Equal(t, 123, svc.GetIntSetting(ctx, "authSessionTimeout", 0))

	require.NoError(t, svc.SetStringSetting(ctx, "baseServerUrl", "http://localhost"))
	require.Equal(t, "http://localhost", svc.GetStringSetting(ctx, "baseServerUrl", ""))
}

func TestSettingsService_UpdateSetting(t *testing.T) {
	ctx, store, svc, cleanup := setupSettingsService(t)
	defer cleanup()

	// Use an existing key ("pruneMode") instead of a non-existent one
	require.NoError(t, svc.UpdateSetting(ctx, "pruneMode", "all"))

	sv, err := store.GetSetting(ctx, "pruneMode")
	require.NoError(t, err)
	require.NotNil(t, sv)
	require.Equal(t, "all", sv.Value)
}

func TestSettingsService_EnsureEncryptionKey(t *testing.T) {
	ctx, store, svc, cleanup := setupSettingsService(t)
	defer cleanup()

	k1, err := svc.EnsureEncryptionKey(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, k1)

	k2, err := svc.EnsureEncryptionKey(ctx)
	require.NoError(t, err)
	require.Equal(t, k1, k2, "encryption key should be stable between calls")

	sv, err := store.GetSetting(ctx, "encryptionKey")
	require.NoError(t, err)
	require.NotNil(t, sv)
	require.Equal(t, k1, sv.Value)
}

func TestSettingsService_UpdateSettings_MergeOidcSecret(t *testing.T) {
	ctx, store, svc, cleanup := setupSettingsService(t)
	defer cleanup()

	// Seed existing OIDC config with a secret
	existing := settings.OidcConfig{
		ClientID:     "old",
		ClientSecret: "keep-this",
		IssuerURL:    "https://issuer",
	}
	b, err := json.Marshal(existing)
	require.NoError(t, err)
	require.NoError(t, svc.UpdateSetting(ctx, "authOidcConfig", string(b)))

	// Incoming update missing clientSecret should preserve existing one
	incoming := settings.OidcConfig{
		ClientID:  "new",
		IssuerURL: "https://issuer",
	}
	nb, err := json.Marshal(incoming)
	require.NoError(t, err)
	s := string(nb)

	updates := settings.Update{
		AuthOidcConfig: &s,
	}
	_, err = svc.UpdateSettings(ctx, updates)
	require.NoError(t, err)

	cfgVar, err := store.GetSetting(ctx, "authOidcConfig")
	require.NoError(t, err)
	require.NotNil(t, cfgVar)

	var merged settings.OidcConfig
	require.NoError(t, json.Unmarshal([]byte(cfgVar.Value), &merged))
	require.Equal(t, "new", merged.ClientID)
	require.Equal(t, "keep-this", merged.ClientSecret)
}

func TestSettingsService_LoadDatabaseSettings_ReloadsChanges(t *testing.T) {
	ctx, _, svc, cleanup := setupSettingsService(t)
	defer cleanup()

	// Initially empty DB -> defaults (not persisted yet)
	require.NoError(t, svc.EnsureDefaultSettings(ctx))

	// Update a value directly in DB
	require.NoError(t, svc.UpdateSetting(ctx, "projectsDirectory", "custom/projects"))

	// Force reload
	require.NoError(t, svc.LoadDatabaseSettings(ctx))

	cfg := svc.GetSettingsConfig()
	require.Equal(t, "custom/projects", cfg.ProjectsDirectory.Value)
}

func TestSettingsService_LoadDatabaseSettings_UIConfigurationDisabled_Env(t *testing.T) {
	// Set env + disable flag BEFORE service init
	t.Setenv("UI_CONFIGURATION_DISABLED", "true")
	t.Setenv("PROJECTS_DIRECTORY", "env/projects")
	t.Setenv("BASE_SERVER_URL", "https://env.example")

	c := config.Load()
	c.UIConfigurationDisabled = true

	ctx, _, svc, cleanup := setupSettingsService(t)
	defer cleanup()

	// Reload explicitly (NewSettingsService already did, but explicit for clarity)
	require.NoError(t, svc.LoadDatabaseSettings(ctx))

	cfg := svc.GetSettingsConfig()
	require.Equal(t, "env/projects", cfg.ProjectsDirectory.Value)
	require.Equal(t, "https://env.example", cfg.BaseServerURL.Value)
}

func TestSettingsService_UpdateSettings_RefreshesCache(t *testing.T) {
	ctx, _, svc, cleanup := setupSettingsService(t)
	defer cleanup()
	require.NoError(t, svc.EnsureDefaultSettings(ctx))

	newDir := "custom/projects2"
	req := settings.Update{
		ProjectsDirectory: &newDir,
	}

	_, err := svc.UpdateSettings(ctx, req)
	require.NoError(t, err)

	// ListSettings uses the cached snapshot; should reflect updated value
	list := svc.ListSettings(true)
	found := false
	for _, sv := range list {
		if sv.Key == "projectsDirectory" {
			found = true
			require.Equal(t, newDir, sv.Value)
		}
	}
	require.True(t, found, "projectsDirectory setting not found in cached list")
}

func TestSettingsService_LoadDatabaseSettings_InternalKeys_EnvMode(t *testing.T) {
	// Set env + disable flag
	t.Setenv("UI_CONFIGURATION_DISABLED", "true")

	ctx, store, svc, cleanup := setupSettingsService(t)
	defer cleanup()

	// Pre-populate an internal setting in the DB
	internalKey := "instanceId"
	internalVal := "test-instance-id"
	require.NoError(t, store.UpsertSetting(ctx, internalKey, internalVal))

	// Reload explicitly to trigger the env loading path
	require.NoError(t, svc.LoadDatabaseSettings(ctx))

	cfg := svc.GetSettingsConfig()
	// Should have loaded the internal setting from DB even in env mode
	require.Equal(t, internalVal, cfg.InstanceID.Value)
}

func TestSettingsService_MigrateOidcConfigToFields(t *testing.T) {
	ctx, store, svc, cleanup := setupSettingsService(t)
	defer cleanup()
	require.NoError(t, svc.EnsureDefaultSettings(ctx))

	// Seed legacy OIDC JSON config
	legacyConfig := settings.OidcConfig{
		ClientID:     "legacy-client-id",
		ClientSecret: "legacy-secret",
		IssuerURL:    "https://legacy-issuer.example",
		Scopes:       "openid email profile",
		AdminClaim:   "groups",
		AdminValue:   "admin",
	}
	b, err := json.Marshal(legacyConfig)
	require.NoError(t, err)
	require.NoError(t, svc.UpdateSetting(ctx, "authOidcConfig", string(b)))

	// Run migration
	err = svc.MigrateOidcConfigToFields(ctx)
	require.NoError(t, err)

	// Verify individual fields were populated
	clientId, err := store.GetSetting(ctx, "oidcClientId")
	require.NoError(t, err)
	require.NotNil(t, clientId)
	require.Equal(t, "legacy-client-id", clientId.Value)

	clientSecret, err := store.GetSetting(ctx, "oidcClientSecret")
	require.NoError(t, err)
	require.NotNil(t, clientSecret)
	require.Equal(t, "legacy-secret", clientSecret.Value)

	issuerURL, err := store.GetSetting(ctx, "oidcIssuerUrl")
	require.NoError(t, err)
	require.NotNil(t, issuerURL)
	require.Equal(t, "https://legacy-issuer.example", issuerURL.Value)

	scopes, err := store.GetSetting(ctx, "oidcScopes")
	require.NoError(t, err)
	require.NotNil(t, scopes)
	require.Equal(t, "openid email profile", scopes.Value)

	adminClaim, err := store.GetSetting(ctx, "oidcAdminClaim")
	require.NoError(t, err)
	require.NotNil(t, adminClaim)
	require.Equal(t, "groups", adminClaim.Value)

	adminValue, err := store.GetSetting(ctx, "oidcAdminValue")
	require.NoError(t, err)
	require.NotNil(t, adminValue)
	require.Equal(t, "admin", adminValue.Value)
}

func TestSettingsService_MigrateOidcConfigToFields_SkipsIfAlreadyMigrated(t *testing.T) {
	ctx, store, svc, cleanup := setupSettingsService(t)
	defer cleanup()
	require.NoError(t, svc.EnsureDefaultSettings(ctx))

	// Pre-populate individual field
	require.NoError(t, svc.UpdateSetting(ctx, "oidcClientId", "already-migrated"))

	// Seed legacy config too
	legacyConfig := settings.OidcConfig{
		ClientID:  "old-id",
		IssuerURL: "https://old-issuer.example",
	}
	b, err := json.Marshal(legacyConfig)
	require.NoError(t, err)
	require.NoError(t, svc.UpdateSetting(ctx, "authOidcConfig", string(b)))

	// Run migration - should skip since individual field is populated
	err = svc.MigrateOidcConfigToFields(ctx)
	require.NoError(t, err)

	// Verify field was NOT overwritten
	clientId, err := store.GetSetting(ctx, "oidcClientId")
	require.NoError(t, err)
	require.NotNil(t, clientId)
	require.Equal(t, "already-migrated", clientId.Value)
}

func TestSettingsService_MigrateOidcConfigToFields_RealWorldJSON(t *testing.T) {
	ctx, store, svc, cleanup := setupSettingsService(t)
	defer cleanup()
	require.NoError(t, svc.EnsureDefaultSettings(ctx))

	// Test with real-world JSON format (as stored in database)
	realWorldJSON := `{"clientId":"ab92b6cf-283d-4764-9308-92a9b9496bf1","clientSecret":"super-secret-value","issuerUrl":"https://id.ofkm.us","scopes":"openid email profile groups","adminClaim":"groups","adminValue":"_arcane_admins"}`
	require.NoError(t, svc.UpdateSetting(ctx, "authOidcConfig", realWorldJSON))

	// Run migration
	err := svc.MigrateOidcConfigToFields(ctx)
	require.NoError(t, err)

	// Verify all individual fields were populated correctly
	clientId, err := store.GetSetting(ctx, "oidcClientId")
	require.NoError(t, err)
	require.NotNil(t, clientId)
	require.Equal(t, "ab92b6cf-283d-4764-9308-92a9b9496bf1", clientId.Value)

	clientSecret, err := store.GetSetting(ctx, "oidcClientSecret")
	require.NoError(t, err)
	require.NotNil(t, clientSecret)
	require.Equal(t, "super-secret-value", clientSecret.Value)

	issuerURL, err := store.GetSetting(ctx, "oidcIssuerUrl")
	require.NoError(t, err)
	require.NotNil(t, issuerURL)
	require.Equal(t, "https://id.ofkm.us", issuerURL.Value)

	scopes, err := store.GetSetting(ctx, "oidcScopes")
	require.NoError(t, err)
	require.NotNil(t, scopes)
	require.Equal(t, "openid email profile groups", scopes.Value)

	adminClaim, err := store.GetSetting(ctx, "oidcAdminClaim")
	require.NoError(t, err)
	require.NotNil(t, adminClaim)
	require.Equal(t, "groups", adminClaim.Value)

	adminValue, err := store.GetSetting(ctx, "oidcAdminValue")
	require.NoError(t, err)
	require.NotNil(t, adminValue)
	require.Equal(t, "_arcane_admins", adminValue.Value)
}

func TestSettingsService_MigrateOidcConfigToFields_EmptyConfig(t *testing.T) {
	ctx, store, svc, cleanup := setupSettingsService(t)
	defer cleanup()
	require.NoError(t, svc.EnsureDefaultSettings(ctx))

	// Empty config should not cause errors
	require.NoError(t, svc.UpdateSetting(ctx, "authOidcConfig", "{}"))

	err := svc.MigrateOidcConfigToFields(ctx)
	require.NoError(t, err)

	// Verify fields remain empty
	clientId, err := store.GetSetting(ctx, "oidcClientId")
	require.NoError(t, err)
	require.NotNil(t, clientId)
	require.Empty(t, clientId.Value)
}

func TestSettingsService_MigrateOidcConfigToFields_InvalidJSON(t *testing.T) {
	ctx, store, svc, cleanup := setupSettingsService(t)
	defer cleanup()
	require.NoError(t, svc.EnsureDefaultSettings(ctx))

	// Invalid JSON should not cause errors (gracefully handled)
	require.NoError(t, svc.UpdateSetting(ctx, "authOidcConfig", "not valid json"))

	err := svc.MigrateOidcConfigToFields(ctx)
	require.NoError(t, err) // Should not return error, just skip

	// Verify fields remain empty
	clientId, err := store.GetSetting(ctx, "oidcClientId")
	require.NoError(t, err)
	require.NotNil(t, clientId)
	require.Empty(t, clientId.Value)
}

func TestSettingsService_MigrateOidcConfigToFields_DefaultScopes(t *testing.T) {
	ctx, store, svc, cleanup := setupSettingsService(t)
	defer cleanup()
	require.NoError(t, svc.EnsureDefaultSettings(ctx))

	// Config without scopes should get default scopes
	configWithoutScopes := `{"clientId":"test-client","issuerUrl":"https://test.example"}`
	require.NoError(t, svc.UpdateSetting(ctx, "authOidcConfig", configWithoutScopes))

	err := svc.MigrateOidcConfigToFields(ctx)
	require.NoError(t, err)

	scopes, err := store.GetSetting(ctx, "oidcScopes")
	require.NoError(t, err)
	require.NotNil(t, scopes)
	require.Equal(t, "openid email profile", scopes.Value)
}

func TestSettingsService_NormalizeProjectsDirectory_ConvertsRelativeToAbsolute(t *testing.T) {
	ctx, store, svc, cleanup := setupSettingsService(t)
	defer cleanup()

	// Seed with relative path
	require.NoError(t, svc.UpdateSetting(ctx, "projectsDirectory", "data/projects"))

	// Run normalization without env var set (empty string)
	err := svc.NormalizeProjectsDirectory(ctx, "")
	require.NoError(t, err)

	// Verify it was updated to absolute path
	setting, err := store.GetSetting(ctx, "projectsDirectory")
	require.NoError(t, err)
	require.NotNil(t, setting)

	// Should be converted to absolute path
	expectedPath, _ := filepath.Abs("data/projects")
	require.Equal(t, expectedPath, setting.Value)
	require.True(t, filepath.IsAbs(setting.Value), "path should be absolute")
}

func TestSettingsService_NormalizeProjectsDirectory_SkipsWhenEnvSet(t *testing.T) {
	ctx, store, svc, cleanup := setupSettingsService(t)
	defer cleanup()

	// Seed with relative path
	require.NoError(t, svc.UpdateSetting(ctx, "projectsDirectory", "data/projects"))

	// Run normalization WITH env var set
	err := svc.NormalizeProjectsDirectory(ctx, "/custom/env/path")
	require.NoError(t, err)

	// Verify it was NOT changed
	setting, err := store.GetSetting(ctx, "projectsDirectory")
	require.NoError(t, err)
	require.NotNil(t, setting)
	require.Equal(t, "data/projects", setting.Value, "should not change when env var is set")
}

func TestSettingsService_NormalizeProjectsDirectory_LeavesOtherPathsUnchanged(t *testing.T) {
	ctx, store, svc, cleanup := setupSettingsService(t)
	defer cleanup()

	customPath := "/custom/projects/path"
	require.NoError(t, svc.UpdateSetting(ctx, "projectsDirectory", customPath))

	// Run normalization
	err := svc.NormalizeProjectsDirectory(ctx, "")
	require.NoError(t, err)

	// Verify it was NOT changed
	setting, err := store.GetSetting(ctx, "projectsDirectory")
	require.NoError(t, err)
	require.NotNil(t, setting)
	require.Equal(t, customPath, setting.Value, "should not change custom paths")
}

func TestSettingsService_NormalizeProjectsDirectory_HandlesNotFound(t *testing.T) {
	ctx, _, svc, cleanup := setupSettingsService(t)
	defer cleanup()

	// Don't create the setting at all

	// Run normalization - should not error
	err := svc.NormalizeProjectsDirectory(ctx, "")
	require.NoError(t, err)
}

func TestSettingsService_NormalizeProjectsDirectory_UpdatesCacheAfterNormalization(t *testing.T) {
	ctx, _, svc, cleanup := setupSettingsService(t)
	defer cleanup()
	require.NoError(t, svc.EnsureDefaultSettings(ctx))

	// Set to relative path
	require.NoError(t, svc.UpdateSetting(ctx, "projectsDirectory", "data/projects"))
	require.NoError(t, svc.LoadDatabaseSettings(ctx))

	// Verify cache has relative path
	cfg1 := svc.GetSettingsConfig()
	require.Equal(t, "data/projects", cfg1.ProjectsDirectory.Value)

	// Run normalization
	err := svc.NormalizeProjectsDirectory(ctx, "")
	require.NoError(t, err)

	// Verify cache was updated to absolute path
	cfg2 := svc.GetSettingsConfig()
	expectedPath, _ := filepath.Abs("data/projects")
	require.Equal(t, expectedPath, cfg2.ProjectsDirectory.Value)
	require.True(t, filepath.IsAbs(cfg2.ProjectsDirectory.Value), "path should be absolute")
}
