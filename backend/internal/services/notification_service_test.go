package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/getarcaneapp/arcane/backend/internal/config"
	"github.com/getarcaneapp/arcane/backend/internal/database"
	"github.com/getarcaneapp/arcane/backend/internal/utils/crypto"
	"github.com/getarcaneapp/arcane/types/base"
	"github.com/getarcaneapp/arcane/types/notification"
)

func setupNotificationTestStore(t *testing.T) (context.Context, *database.SqlcStore, func()) {
	t.Helper()
	ctx := context.Background()
	db, err := database.Initialize(ctx, testNotificationSQLiteDSN(t))
	require.NoError(t, err)
	store, err := database.NewSqlcStore(db)
	require.NoError(t, err)

	// Initialize crypto for tests (requires 32+ byte key)
	testCfg := &config.Config{
		EncryptionKey: "test-encryption-key-for-testing-32bytes-min",
		Environment:   "test",
	}
	crypto.InitEncryption(testCfg)

	return ctx, store, func() { _ = db.Close() }
}

func testNotificationSQLiteDSN(t *testing.T) string {
	t.Helper()
	name := strings.ReplaceAll(t.Name(), "/", "_")
	return fmt.Sprintf("file:%s?mode=memory&cache=shared", name)
}

func setupNotificationService(t *testing.T) (context.Context, *database.SqlcStore, *NotificationService, func()) {
	t.Helper()
	ctx, store, cleanup := setupNotificationTestStore(t)
	cfg := &config.Config{
		EncryptionKey: "test-encryption-key-for-testing-32bytes-min",
		Environment:   "test",
	}
	svc := NewNotificationService(store, store, cfg)
	return ctx, store, svc, cleanup
}

func TestNotificationService_MigrateDiscordWebhookUrlToFields(t *testing.T) {
	ctx, store, svc, cleanup := setupNotificationService(t)
	defer cleanup()

	// Create legacy Discord config with webhookUrl
	legacyConfig := map[string]interface{}{
		"webhookUrl": "https://discord.com/api/webhooks/123456789/abcdef123456",
		"username":   "Arcane Bot",
		"avatarUrl":  "https://example.com/avatar.png",
		"events": map[string]bool{
			"image_update":     true,
			"container_update": false,
		},
	}

	configBytes, err := json.Marshal(legacyConfig)
	require.NoError(t, err)

	var configJSON base.JSON
	require.NoError(t, json.Unmarshal(configBytes, &configJSON))

	_, err = store.UpsertNotificationSetting(ctx, notification.NotificationProviderDiscord, true, configJSON)
	require.NoError(t, err)

	// Run migration
	err = svc.MigrateDiscordWebhookUrlToFields(ctx)
	require.NoError(t, err)

	// Verify migration results
	migratedSetting, err := store.GetNotificationSettingByProvider(ctx, notification.NotificationProviderDiscord)
	require.NoError(t, err)
	require.NotNil(t, migratedSetting)

	var discordConfig notification.DiscordConfig
	configBytes, err = json.Marshal(migratedSetting.Config)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(configBytes, &discordConfig))

	// Verify webhookId and token were extracted
	require.Equal(t, "123456789", discordConfig.WebhookID)
	require.NotEmpty(t, discordConfig.Token)

	// Verify token is encrypted and can be decrypted
	decryptedToken, err := crypto.Decrypt(discordConfig.Token)
	require.NoError(t, err)
	require.Equal(t, "abcdef123456", decryptedToken)

	// Verify other fields were preserved
	require.Equal(t, "Arcane Bot", discordConfig.Username)
	require.Equal(t, "https://example.com/avatar.png", discordConfig.AvatarURL)
	require.True(t, discordConfig.Events[notification.NotificationEventImageUpdate])
	require.False(t, discordConfig.Events[notification.NotificationEventContainerUpdate])
}

func TestNotificationService_MigrateDiscordWebhookUrlToFields_SkipsIfAlreadyMigrated(t *testing.T) {
	ctx, store, svc, cleanup := setupNotificationService(t)
	defer cleanup()

	// Create already-migrated config with webhookId and token
	encryptedToken, err := crypto.Encrypt("already-migrated-token")
	require.NoError(t, err)

	migratedConfig := notification.DiscordConfig{
		WebhookID: "999999999",
		Token:     encryptedToken,
		Username:  "Already Migrated",
	}

	configBytes, err := json.Marshal(migratedConfig)
	require.NoError(t, err)

	var configJSON base.JSON
	require.NoError(t, json.Unmarshal(configBytes, &configJSON))

	_, err = store.UpsertNotificationSetting(ctx, notification.NotificationProviderDiscord, true, configJSON)
	require.NoError(t, err)

	// Run migration - should skip
	err = svc.MigrateDiscordWebhookUrlToFields(ctx)
	require.NoError(t, err)

	// Verify config was NOT changed
	unchangedSetting, err := store.GetNotificationSettingByProvider(ctx, notification.NotificationProviderDiscord)
	require.NoError(t, err)
	require.NotNil(t, unchangedSetting)

	var discordConfig notification.DiscordConfig
	configBytes, err = json.Marshal(unchangedSetting.Config)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(configBytes, &discordConfig))

	require.Equal(t, "999999999", discordConfig.WebhookID)
	require.Equal(t, encryptedToken, discordConfig.Token)
	require.Equal(t, "Already Migrated", discordConfig.Username)
}

func TestNotificationService_MigrateDiscordWebhookUrlToFields_NoDiscordConfig(t *testing.T) {
	ctx, store, svc, cleanup := setupNotificationService(t)
	defer cleanup()

	// No Discord config exists - migration should not error
	err := svc.MigrateDiscordWebhookUrlToFields(ctx)
	require.NoError(t, err)

	// Verify no settings were created
	settings, err := store.ListNotificationSettings(ctx)
	require.NoError(t, err)
	require.Len(t, settings, 0)
}

func TestNotificationService_MigrateDiscordWebhookUrlToFields_InvalidWebhookUrl(t *testing.T) {
	ctx, store, svc, cleanup := setupNotificationService(t)
	defer cleanup()

	testCases := []struct {
		name       string
		webhookUrl string
	}{
		{
			name:       "missing webhooks path",
			webhookUrl: "https://discord.com/api/other/123456789/abcdef",
		},
		{
			name:       "incomplete webhook path",
			webhookUrl: "https://discord.com/api/webhooks/123456789",
		},
		{
			name:       "empty webhook url",
			webhookUrl: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clean up before each sub-test
			require.NoError(t, store.DeleteNotificationSetting(ctx, notification.NotificationProviderDiscord))

			legacyConfig := map[string]interface{}{
				"webhookUrl": tc.webhookUrl,
			}

			configBytes, err := json.Marshal(legacyConfig)
			require.NoError(t, err)

			var configJSON base.JSON
			require.NoError(t, json.Unmarshal(configBytes, &configJSON))

			_, err = store.UpsertNotificationSetting(ctx, notification.NotificationProviderDiscord, true, configJSON)
			require.NoError(t, err)

			// Migration should not error but should skip invalid URLs
			err = svc.MigrateDiscordWebhookUrlToFields(ctx)
			require.NoError(t, err)

			// Verify config was not changed
			unchangedSetting, err := store.GetNotificationSettingByProvider(ctx, notification.NotificationProviderDiscord)
			require.NoError(t, err)
			require.NotNil(t, unchangedSetting)

			var resultConfig map[string]interface{}
			configBytes, err = json.Marshal(unchangedSetting.Config)
			require.NoError(t, err)
			require.NoError(t, json.Unmarshal(configBytes, &resultConfig))

			// Should still have webhookUrl (not migrated)
			if tc.webhookUrl != "" {
				require.Equal(t, tc.webhookUrl, resultConfig["webhookUrl"])
			}
		})
	}
}

func TestNotificationService_MigrateDiscordWebhookUrlToFields_EmptyConfig(t *testing.T) {
	ctx, store, svc, cleanup := setupNotificationService(t)
	defer cleanup()

	// Create Discord setting with empty config
	_, err := store.UpsertNotificationSetting(ctx, notification.NotificationProviderDiscord, false, base.JSON{})
	require.NoError(t, err)

	// Migration should not error
	err = svc.MigrateDiscordWebhookUrlToFields(ctx)
	require.NoError(t, err)

	// Verify config remains empty
	unchangedSetting, err := store.GetNotificationSettingByProvider(ctx, notification.NotificationProviderDiscord)
	require.NoError(t, err)
	require.NotNil(t, unchangedSetting)
	require.Empty(t, unchangedSetting.Config)
}

func TestNotificationService_MigrateDiscordWebhookUrlToFields_PreservesAllFields(t *testing.T) {
	ctx, store, svc, cleanup := setupNotificationService(t)
	defer cleanup()

	// Create legacy config with all optional fields
	legacyConfig := map[string]interface{}{
		"webhookUrl": "https://discord.com/api/webhooks/111222333/token444555",
		"username":   "Custom Bot Name",
		"avatarUrl":  "https://cdn.example.com/bot-avatar.jpg",
		"events": map[string]bool{
			"image_update":     false,
			"container_update": true,
		},
	}

	configBytes, err := json.Marshal(legacyConfig)
	require.NoError(t, err)

	var configJSON base.JSON
	require.NoError(t, json.Unmarshal(configBytes, &configJSON))

	_, err = store.UpsertNotificationSetting(ctx, notification.NotificationProviderDiscord, true, configJSON)
	require.NoError(t, err)

	// Run migration
	err = svc.MigrateDiscordWebhookUrlToFields(ctx)
	require.NoError(t, err)

	// Verify all fields were preserved
	migratedSetting, err := store.GetNotificationSettingByProvider(ctx, notification.NotificationProviderDiscord)
	require.NoError(t, err)
	require.NotNil(t, migratedSetting)

	var discordConfig notification.DiscordConfig
	configBytes, err = json.Marshal(migratedSetting.Config)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(configBytes, &discordConfig))

	require.Equal(t, "111222333", discordConfig.WebhookID)
	require.NotEmpty(t, discordConfig.Token)

	decryptedToken, err := crypto.Decrypt(discordConfig.Token)
	require.NoError(t, err)
	require.Equal(t, "token444555", decryptedToken)

	require.Equal(t, "Custom Bot Name", discordConfig.Username)
	require.Equal(t, "https://cdn.example.com/bot-avatar.jpg", discordConfig.AvatarURL)
	require.False(t, discordConfig.Events[notification.NotificationEventImageUpdate])
	require.True(t, discordConfig.Events[notification.NotificationEventContainerUpdate])
}
