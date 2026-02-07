package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"strings"
	"text/template"
	"time"

	"github.com/getarcaneapp/arcane/backend/internal/config"
	"github.com/getarcaneapp/arcane/backend/internal/database"
	"github.com/getarcaneapp/arcane/backend/internal/utils/crypto"
	"github.com/getarcaneapp/arcane/backend/internal/utils/notifications"
	"github.com/getarcaneapp/arcane/backend/resources"
	"github.com/getarcaneapp/arcane/types/base"
	"github.com/getarcaneapp/arcane/types/imageupdate"
	"github.com/getarcaneapp/arcane/types/notification"
)

const logoURLPath = "/api/app-images/logo-email"

type NotificationService struct {
	store          database.NotificationStore
	appriseStore   database.AppriseStore
	config         *config.Config
	appriseService *AppriseService
}

func NewNotificationService(store database.NotificationStore, appriseStore database.AppriseStore, cfg *config.Config) *NotificationService {
	return &NotificationService{
		store:          store,
		appriseStore:   appriseStore,
		config:         cfg,
		appriseService: NewAppriseService(appriseStore, cfg),
	}
}

func (s *NotificationService) GetAllSettings(ctx context.Context) ([]notification.NotificationSettings, error) {
	settings, err := s.store.ListNotificationSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification settings: %w", err)
	}
	return settings, nil
}

func (s *NotificationService) GetSettingsByProvider(ctx context.Context, provider notification.NotificationProvider) (*notification.NotificationSettings, error) {
	setting, err := s.store.GetNotificationSettingByProvider(ctx, provider)
	if err != nil {
		return nil, err
	}
	if setting == nil {
		return nil, errors.New("notification settings not found")
	}
	return setting, nil
}

func (s *NotificationService) CreateOrUpdateSettings(ctx context.Context, provider notification.NotificationProvider, enabled bool, config base.JSON) (*notification.NotificationSettings, error) {
	// Clear config if provider is disabled
	if !enabled {
		config = base.JSON{}
	}
	setting, err := s.store.UpsertNotificationSetting(ctx, provider, enabled, config)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert notification settings: %w", err)
	}
	return setting, nil
}

func (s *NotificationService) DeleteSettings(ctx context.Context, provider notification.NotificationProvider) error {
	if err := s.store.DeleteNotificationSetting(ctx, provider); err != nil {
		return fmt.Errorf("failed to delete notification settings: %w", err)
	}
	return nil
}

func (s *NotificationService) SendImageUpdateNotification(ctx context.Context, imageRef string, updateInfo *imageupdate.Response, eventType notification.NotificationEventType) error {
	// Send to Apprise if enabled (don't block on error)
	if appriseErr := s.appriseService.SendImageUpdateNotification(ctx, imageRef, updateInfo); appriseErr != nil {
		slog.WarnContext(ctx, "Failed to send Apprise notification", "error", appriseErr)
	}

	settings, err := s.GetAllSettings(ctx)
	if err != nil {
		return fmt.Errorf("failed to get notification settings: %w", err)
	}

	var errors []string
	for _, setting := range settings {
		if !setting.Enabled {
			continue
		}

		// Check if this event type is enabled for this provider
		if !s.isEventEnabled(setting.Config, eventType) {
			continue
		}

		var sendErr error
		switch setting.Provider {
		case notification.NotificationProviderDiscord:
			sendErr = s.sendDiscordNotification(ctx, imageRef, updateInfo, setting.Config)
		case notification.NotificationProviderEmail:
			sendErr = s.sendEmailNotification(ctx, imageRef, updateInfo, setting.Config)
		case notification.NotificationProviderTelegram:
			sendErr = s.sendTelegramNotification(ctx, imageRef, updateInfo, setting.Config)
		case notification.NotificationProviderSignal:
			sendErr = s.sendSignalNotification(ctx, imageRef, updateInfo, setting.Config)
		case notification.NotificationProviderSlack:
			sendErr = s.sendSlackNotification(ctx, imageRef, updateInfo, setting.Config)
		case notification.NotificationProviderNtfy:
			sendErr = s.sendNtfyNotification(ctx, imageRef, updateInfo, setting.Config)
		case notification.NotificationProviderPushover:
			sendErr = s.sendPushoverNotification(ctx, imageRef, updateInfo, setting.Config)
		case notification.NotificationProviderGotify:
			sendErr = s.sendGotifyNotification(ctx, imageRef, updateInfo, setting.Config)
		case notification.NotificationProviderGeneric:
			sendErr = s.sendGenericNotification(ctx, imageRef, updateInfo, setting.Config)
		default:
			slog.WarnContext(ctx, "Unknown notification provider", "provider", setting.Provider)
			continue
		}

		status := "success"
		var errMsg *string
		if sendErr != nil {
			status = "failed"
			msg := sendErr.Error()
			errMsg = &msg
			errors = append(errors, fmt.Sprintf("%s: %s", setting.Provider, msg))
		}

		s.logNotification(ctx, setting.Provider, imageRef, status, errMsg, base.JSON{
			"hasUpdate":     updateInfo.HasUpdate,
			"currentDigest": updateInfo.CurrentDigest,
			"latestDigest":  updateInfo.LatestDigest,
			"updateType":    updateInfo.UpdateType,
			"eventType":     string(eventType),
		})
	}

	if len(errors) > 0 {
		return fmt.Errorf("notification errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

// isEventEnabled checks if a specific event type is enabled in the config
func (s *NotificationService) isEventEnabled(config base.JSON, eventType notification.NotificationEventType) bool {
	configBytes, err := json.Marshal(config)
	if err != nil {
		return true // Default to enabled if we can't parse
	}

	var configMap map[string]interface{}
	if err := json.Unmarshal(configBytes, &configMap); err != nil {
		return true // Default to enabled if we can't parse
	}

	events, ok := configMap["events"].(map[string]interface{})
	if !ok {
		return true // If no events config, default to enabled
	}

	enabled, ok := events[string(eventType)].(bool)
	if !ok {
		return true // If event type not specified, default to enabled
	}

	return enabled
}

func (s *NotificationService) SendContainerUpdateNotification(ctx context.Context, containerName, imageRef, oldDigest, newDigest string) error {
	// Send to Apprise if enabled (don't block on error)
	if appriseErr := s.appriseService.SendContainerUpdateNotification(ctx, containerName, imageRef, oldDigest, newDigest); appriseErr != nil {
		slog.WarnContext(ctx, "Failed to send Apprise notification", "error", appriseErr)
	}

	settings, err := s.GetAllSettings(ctx)
	if err != nil {
		return fmt.Errorf("failed to get notification settings: %w", err)
	}

	var errors []string
	for _, setting := range settings {
		if !setting.Enabled {
			continue
		}

		// Check if container update event is enabled for this provider
		if !s.isEventEnabled(setting.Config, notification.NotificationEventContainerUpdate) {
			continue
		}

		var sendErr error
		switch setting.Provider {
		case notification.NotificationProviderDiscord:
			sendErr = s.sendDiscordContainerUpdateNotification(ctx, containerName, imageRef, oldDigest, newDigest, setting.Config)
		case notification.NotificationProviderEmail:
			sendErr = s.sendEmailContainerUpdateNotification(ctx, containerName, imageRef, oldDigest, newDigest, setting.Config)
		case notification.NotificationProviderTelegram:
			sendErr = s.sendTelegramContainerUpdateNotification(ctx, containerName, imageRef, oldDigest, newDigest, setting.Config)
		case notification.NotificationProviderSignal:
			sendErr = s.sendSignalContainerUpdateNotification(ctx, containerName, imageRef, oldDigest, newDigest, setting.Config)
		case notification.NotificationProviderSlack:
			sendErr = s.sendSlackContainerUpdateNotification(ctx, containerName, imageRef, oldDigest, newDigest, setting.Config)
		case notification.NotificationProviderNtfy:
			sendErr = s.sendNtfyContainerUpdateNotification(ctx, containerName, imageRef, oldDigest, newDigest, setting.Config)
		case notification.NotificationProviderPushover:
			sendErr = s.sendPushoverContainerUpdateNotification(ctx, containerName, imageRef, oldDigest, newDigest, setting.Config)
		case notification.NotificationProviderGotify:
			sendErr = s.sendGotifyContainerUpdateNotification(ctx, containerName, imageRef, oldDigest, newDigest, setting.Config)
		case notification.NotificationProviderGeneric:
			sendErr = s.sendGenericContainerUpdateNotification(ctx, containerName, imageRef, oldDigest, newDigest, setting.Config)
		default:
			slog.WarnContext(ctx, "Unknown notification provider", "provider", setting.Provider)
			continue
		}

		status := "success"
		var errMsg *string
		if sendErr != nil {
			status = "failed"
			msg := sendErr.Error()
			errMsg = &msg
			errors = append(errors, fmt.Sprintf("%s: %s", setting.Provider, msg))
		}

		s.logNotification(ctx, setting.Provider, imageRef, status, errMsg, base.JSON{
			"containerName": containerName,
			"oldDigest":     oldDigest,
			"newDigest":     newDigest,
			"eventType":     string(notification.NotificationEventContainerUpdate),
		})
	}

	if len(errors) > 0 {
		return fmt.Errorf("notification errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

func (s *NotificationService) sendDiscordNotification(ctx context.Context, imageRef string, updateInfo *imageupdate.Response, config base.JSON) error {
	var discordConfig notification.DiscordConfig
	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal Discord config: %w", err)
	}
	if err := json.Unmarshal(configBytes, &discordConfig); err != nil {
		return fmt.Errorf("failed to unmarshal Discord config: %w", err)
	}

	if discordConfig.WebhookID == "" || discordConfig.Token == "" {
		return fmt.Errorf("discord webhook ID or token not configured")
	}

	// Decrypt token if encrypted
	if discordConfig.Token != "" {
		if decrypted, err := crypto.Decrypt(discordConfig.Token); err == nil {
			discordConfig.Token = decrypted
		} else {
			slog.Warn("Failed to decrypt Discord token, using raw value (may be unencrypted legacy value)", "error", err)
		}
	}

	// Build message content - Discord embeds via Shoutrrr are sent as formatted markdown
	updateStatus := "No Update"
	if updateInfo.HasUpdate {
		updateStatus = "⚠️ Update Available"
	}

	message := fmt.Sprintf("**🔔 Container Image Update Notification**\n\n"+
		"**Image:** %s\n"+
		"**Status:** %s\n"+
		"**Update Type:** %s\n",
		imageRef, updateStatus, updateInfo.UpdateType)

	if updateInfo.CurrentDigest != "" {
		message += fmt.Sprintf("**Current Digest:** `%s`\n", updateInfo.CurrentDigest)
	}
	if updateInfo.LatestDigest != "" {
		message += fmt.Sprintf("**Latest Digest:** `%s`\n", updateInfo.LatestDigest)
	}

	if err := notifications.SendDiscord(ctx, discordConfig, message); err != nil {
		return fmt.Errorf("failed to send Discord notification: %w", err)
	}

	return nil
}

func (s *NotificationService) sendTelegramNotification(ctx context.Context, imageRef string, updateInfo *imageupdate.Response, config base.JSON) error {
	var telegramConfig notification.TelegramConfig
	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal Telegram config: %w", err)
	}
	if err := json.Unmarshal(configBytes, &telegramConfig); err != nil {
		return fmt.Errorf("failed to unmarshal Telegram config: %w", err)
	}

	if telegramConfig.BotToken == "" {
		return fmt.Errorf("telegram bot token not configured")
	}
	if len(telegramConfig.ChatIDs) == 0 {
		return fmt.Errorf("no telegram chat IDs configured")
	}

	// Decrypt bot token if encrypted
	if telegramConfig.BotToken != "" {
		if decrypted, err := crypto.Decrypt(telegramConfig.BotToken); err == nil {
			telegramConfig.BotToken = decrypted
		} else {
			slog.Warn("Failed to decrypt Telegram bot token, using raw value (may be unencrypted legacy value)", "error", err)
		}
	}

	// Build message content using HTML formatting
	// HTML is easier to escape than Markdown and better supported
	updateStatus := "No Update"
	if updateInfo.HasUpdate {
		updateStatus = "⚠️ Update Available"
	}

	// Use HTML formatting - it's more reliable than Markdown
	message := fmt.Sprintf("🔔 <b>Container Image Update Notification</b>\n\n"+
		"<b>Image:</b> %s\n"+
		"<b>Status:</b> %s\n"+
		"<b>Update Type:</b> %s\n",
		imageRef, updateStatus, updateInfo.UpdateType)

	if updateInfo.CurrentDigest != "" {
		message += fmt.Sprintf("<b>Current Digest:</b> <code>%s</code>\n", updateInfo.CurrentDigest)
	}
	if updateInfo.LatestDigest != "" {
		message += fmt.Sprintf("<b>Latest Digest:</b> <code>%s</code>\n", updateInfo.LatestDigest)
	}

	// Set parse mode to HTML if not already set
	if telegramConfig.ParseMode == "" {
		telegramConfig.ParseMode = "HTML"
	}

	if err := notifications.SendTelegram(ctx, telegramConfig, message); err != nil {
		return fmt.Errorf("failed to send Telegram notification: %w", err)
	}

	return nil
}

func (s *NotificationService) sendEmailNotification(ctx context.Context, imageRef string, updateInfo *imageupdate.Response, config base.JSON) error {
	var emailConfig notification.EmailConfig
	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal email config: %w", err)
	}
	if err := json.Unmarshal(configBytes, &emailConfig); err != nil {
		return fmt.Errorf("failed to unmarshal email config: %w", err)
	}

	if emailConfig.SMTPHost == "" || emailConfig.SMTPPort == 0 {
		return fmt.Errorf("SMTP host or port not configured")
	}
	if len(emailConfig.ToAddresses) == 0 {
		return fmt.Errorf("no recipient email addresses configured")
	}

	if _, err := mail.ParseAddress(emailConfig.FromAddress); err != nil {
		return fmt.Errorf("invalid from address: %w", err)
	}
	for _, addr := range emailConfig.ToAddresses {
		if _, err := mail.ParseAddress(addr); err != nil {
			return fmt.Errorf("invalid to address %s: %w", addr, err)
		}
	}

	if emailConfig.SMTPPassword != "" {
		if decrypted, err := crypto.Decrypt(emailConfig.SMTPPassword); err == nil {
			emailConfig.SMTPPassword = decrypted
		} else {
			slog.Warn("Failed to decrypt email SMTP password, using raw value (may be unencrypted legacy value)", "error", err)
		}
	}

	htmlBody, _, err := s.renderEmailTemplate(imageRef, updateInfo)
	if err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}

	subject := fmt.Sprintf("Container Update Available: %s", notifications.SanitizeForEmail(imageRef))
	if err := notifications.SendEmail(ctx, emailConfig, subject, htmlBody); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

func (s *NotificationService) renderEmailTemplate(imageRef string, updateInfo *imageupdate.Response) (string, string, error) {
	appURL := s.config.GetAppURL()
	logoURL := appURL + logoURLPath
	data := map[string]interface{}{
		"LogoURL":       logoURL,
		"AppURL":        appURL,
		"Environment":   "Local Docker",
		"ImageRef":      imageRef,
		"HasUpdate":     updateInfo.HasUpdate,
		"UpdateType":    updateInfo.UpdateType,
		"CurrentDigest": updateInfo.CurrentDigest,
		"LatestDigest":  updateInfo.LatestDigest,
		"CheckTime":     updateInfo.CheckTime.Format(time.RFC1123),
	}

	htmlContent, err := resources.FS.ReadFile("email-templates/image-update_html.tmpl")
	if err != nil {
		return "", "", fmt.Errorf("failed to read HTML template: %w", err)
	}

	htmlTmpl, err := template.New("html").Parse(string(htmlContent))
	if err != nil {
		return "", "", fmt.Errorf("failed to parse HTML template: %w", err)
	}

	var htmlBuf bytes.Buffer
	if err := htmlTmpl.ExecuteTemplate(&htmlBuf, "root", data); err != nil {
		return "", "", fmt.Errorf("failed to execute HTML template: %w", err)
	}

	textContent, err := resources.FS.ReadFile("email-templates/image-update_text.tmpl")
	if err != nil {
		return "", "", fmt.Errorf("failed to read text template: %w", err)
	}

	textTmpl, err := template.New("text").Parse(string(textContent))
	if err != nil {
		return "", "", fmt.Errorf("failed to parse text template: %w", err)
	}

	var textBuf bytes.Buffer
	if err := textTmpl.ExecuteTemplate(&textBuf, "root", data); err != nil {
		return "", "", fmt.Errorf("failed to execute text template: %w", err)
	}

	return htmlBuf.String(), textBuf.String(), nil
}

func (s *NotificationService) sendDiscordContainerUpdateNotification(ctx context.Context, containerName, imageRef, oldDigest, newDigest string, config base.JSON) error {
	var discordConfig notification.DiscordConfig
	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal Discord config: %w", err)
	}
	if err := json.Unmarshal(configBytes, &discordConfig); err != nil {
		return fmt.Errorf("failed to unmarshal Discord config: %w", err)
	}

	if discordConfig.WebhookID == "" || discordConfig.Token == "" {
		return fmt.Errorf("discord webhook ID or token not configured")
	}

	// Decrypt token if encrypted
	if discordConfig.Token != "" {
		if decrypted, err := crypto.Decrypt(discordConfig.Token); err == nil {
			discordConfig.Token = decrypted
		} else {
			slog.Warn("Failed to decrypt Discord token, using raw value (may be unencrypted legacy value)", "error", err)
		}
	}

	// Build message content
	message := fmt.Sprintf("**✅ Container Successfully Updated**\n\n"+
		"Your container has been updated with the latest image version.\n\n"+
		"**Container:** %s\n"+
		"**Image:** %s\n"+
		"**Status:** ✅ Updated Successfully\n",
		containerName, imageRef)

	if oldDigest != "" {
		message += fmt.Sprintf("**Previous Version:** `%s`\n", oldDigest)
	}
	if newDigest != "" {
		message += fmt.Sprintf("**Current Version:** `%s`\n", newDigest)
	}

	if err := notifications.SendDiscord(ctx, discordConfig, message); err != nil {
		return fmt.Errorf("failed to send Discord notification: %w", err)
	}

	return nil
}

func (s *NotificationService) sendTelegramContainerUpdateNotification(ctx context.Context, containerName, imageRef, oldDigest, newDigest string, config base.JSON) error {
	var telegramConfig notification.TelegramConfig
	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal Telegram config: %w", err)
	}
	if err := json.Unmarshal(configBytes, &telegramConfig); err != nil {
		return fmt.Errorf("failed to unmarshal Telegram config: %w", err)
	}

	if telegramConfig.BotToken == "" {
		return fmt.Errorf("telegram bot token not configured")
	}
	if len(telegramConfig.ChatIDs) == 0 {
		return fmt.Errorf("no telegram chat IDs configured")
	}

	// Decrypt bot token if encrypted
	if telegramConfig.BotToken != "" {
		if decrypted, err := crypto.Decrypt(telegramConfig.BotToken); err == nil {
			telegramConfig.BotToken = decrypted
		} else {
			slog.Warn("Failed to decrypt Telegram bot token, using raw value (may be unencrypted legacy value)", "error", err)
		}
	}

	// Build message content using HTML formatting
	message := fmt.Sprintf("✅ <b>Container Successfully Updated</b>\n\n"+
		"Your container has been updated with the latest image version.\n\n"+
		"<b>Container:</b> %s\n"+
		"<b>Image:</b> %s\n"+
		"<b>Status:</b> ✅ Updated Successfully\n",
		containerName, imageRef)

	if oldDigest != "" {
		message += fmt.Sprintf("<b>Previous Version:</b> <code>%s</code>\n", oldDigest)
	}
	if newDigest != "" {
		message += fmt.Sprintf("<b>Current Version:</b> <code>%s</code>\n", newDigest)
	}

	// Set parse mode to HTML if not already set
	if telegramConfig.ParseMode == "" {
		telegramConfig.ParseMode = "HTML"
	}

	if err := notifications.SendTelegram(ctx, telegramConfig, message); err != nil {
		return fmt.Errorf("failed to send Telegram notification: %w", err)
	}

	return nil
}

func (s *NotificationService) sendEmailContainerUpdateNotification(ctx context.Context, containerName, imageRef, oldDigest, newDigest string, config base.JSON) error {
	var emailConfig notification.EmailConfig
	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal email config: %w", err)
	}
	if err := json.Unmarshal(configBytes, &emailConfig); err != nil {
		return fmt.Errorf("failed to unmarshal email config: %w", err)
	}

	if emailConfig.SMTPHost == "" || emailConfig.SMTPPort == 0 {
		return fmt.Errorf("SMTP host or port not configured")
	}
	if len(emailConfig.ToAddresses) == 0 {
		return fmt.Errorf("no recipient email addresses configured")
	}

	if _, err := mail.ParseAddress(emailConfig.FromAddress); err != nil {
		return fmt.Errorf("invalid from address: %w", err)
	}
	for _, addr := range emailConfig.ToAddresses {
		if _, err := mail.ParseAddress(addr); err != nil {
			return fmt.Errorf("invalid to address %s: %w", addr, err)
		}
	}

	if emailConfig.SMTPPassword != "" {
		if decrypted, err := crypto.Decrypt(emailConfig.SMTPPassword); err == nil {
			emailConfig.SMTPPassword = decrypted
		} else {
			slog.Warn("Failed to decrypt email SMTP password, using raw value (may be unencrypted legacy value)", "error", err)
		}
	}

	htmlBody, _, err := s.renderContainerUpdateEmailTemplate(containerName, imageRef, oldDigest, newDigest)
	if err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}

	subject := fmt.Sprintf("Container Updated: %s", notifications.SanitizeForEmail(containerName))
	if err := notifications.SendEmail(ctx, emailConfig, subject, htmlBody); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

func (s *NotificationService) renderContainerUpdateEmailTemplate(containerName, imageRef, oldDigest, newDigest string) (string, string, error) {
	appURL := s.config.GetAppURL()
	logoURL := appURL + logoURLPath
	data := map[string]interface{}{
		"LogoURL":       logoURL,
		"AppURL":        appURL,
		"Environment":   "Local Docker",
		"ContainerName": containerName,
		"ImageRef":      imageRef,
		"OldDigest":     oldDigest,
		"NewDigest":     newDigest,
		"UpdateTime":    time.Now().Format(time.RFC1123),
	}

	htmlContent, err := resources.FS.ReadFile("email-templates/container-update_html.tmpl")
	if err != nil {
		return "", "", fmt.Errorf("failed to read HTML template: %w", err)
	}

	htmlTmpl, err := template.New("html").Parse(string(htmlContent))
	if err != nil {
		return "", "", fmt.Errorf("failed to parse HTML template: %w", err)
	}

	var htmlBuf bytes.Buffer
	if err := htmlTmpl.ExecuteTemplate(&htmlBuf, "root", data); err != nil {
		return "", "", fmt.Errorf("failed to execute HTML template: %w", err)
	}

	textContent, err := resources.FS.ReadFile("email-templates/container-update_text.tmpl")
	if err != nil {
		return "", "", fmt.Errorf("failed to read text template: %w", err)
	}

	textTmpl, err := template.New("text").Parse(string(textContent))
	if err != nil {
		return "", "", fmt.Errorf("failed to parse text template: %w", err)
	}

	var textBuf bytes.Buffer
	if err := textTmpl.ExecuteTemplate(&textBuf, "root", data); err != nil {
		return "", "", fmt.Errorf("failed to execute text template: %w", err)
	}

	return htmlBuf.String(), textBuf.String(), nil
}

func (s *NotificationService) TestNotification(ctx context.Context, provider notification.NotificationProvider, testType string) error {
	setting, err := s.GetSettingsByProvider(ctx, provider)
	if err != nil {
		return fmt.Errorf("please save your %s settings before testing", provider)
	}

	testUpdate := &imageupdate.Response{
		HasUpdate:      true,
		UpdateType:     "digest",
		CurrentDigest:  "sha256:abc123def456789012345678901234567890",
		LatestDigest:   "sha256:xyz789ghi012345678901234567890123456",
		CheckTime:      time.Now(),
		ResponseTimeMs: 100,
	}

	switch provider {
	case notification.NotificationProviderDiscord:
		return s.sendDiscordNotification(ctx, "test/image:latest", testUpdate, setting.Config)
	case notification.NotificationProviderEmail:
		if testType == "image-update" {
			return s.sendEmailNotification(ctx, "nginx:latest", testUpdate, setting.Config)
		}
		if testType == "batch-image-update" {
			// Create test batch updates with multiple images
			testUpdates := map[string]*imageupdate.Response{
				"nginx:latest": {
					HasUpdate:      true,
					UpdateType:     "digest",
					CurrentDigest:  "sha256:abc123def456789012345678901234567890",
					LatestDigest:   "sha256:xyz789ghi012345678901234567890123456",
					CheckTime:      time.Now(),
					ResponseTimeMs: 100,
				},
				"postgres:16-alpine": {
					HasUpdate:      true,
					UpdateType:     "digest",
					CurrentDigest:  "sha256:def456abc123789012345678901234567890",
					LatestDigest:   "sha256:ghi789xyz012345678901234567890123456",
					CheckTime:      time.Now(),
					ResponseTimeMs: 120,
				},
				"redis:7.2-alpine": {
					HasUpdate:      true,
					UpdateType:     "digest",
					CurrentDigest:  "sha256:123456789abc012345678901234567890def",
					LatestDigest:   "sha256:456789012def345678901234567890123abc",
					CheckTime:      time.Now(),
					ResponseTimeMs: 95,
				},
			}
			return s.sendBatchEmailNotification(ctx, testUpdates, setting.Config)
		}
		return s.sendTestEmail(ctx, setting.Config)
	case notification.NotificationProviderTelegram:
		return s.sendTelegramNotification(ctx, "nginx:latest", testUpdate, setting.Config)
	case notification.NotificationProviderSignal:
		return s.sendSignalNotification(ctx, "nginx:latest", testUpdate, setting.Config)
	case notification.NotificationProviderSlack:
		return s.sendSlackNotification(ctx, "nginx:latest", testUpdate, setting.Config)
	case notification.NotificationProviderNtfy:
		return s.sendNtfyNotification(ctx, "test/image:latest", testUpdate, setting.Config)
	case notification.NotificationProviderPushover:
		return s.sendPushoverNotification(ctx, "test/image:latest", testUpdate, setting.Config)
	case notification.NotificationProviderGotify:
		return s.sendGotifyNotification(ctx, "test/image:latest", testUpdate, setting.Config)
	case notification.NotificationProviderGeneric:
		return s.sendGenericNotification(ctx, "test/image:latest", testUpdate, setting.Config)
	default:
		return fmt.Errorf("unknown provider: %s", provider)
	}
}

func (s *NotificationService) sendTestEmail(ctx context.Context, config base.JSON) error {
	var emailConfig notification.EmailConfig
	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal email config: %w", err)
	}
	if err := json.Unmarshal(configBytes, &emailConfig); err != nil {
		return fmt.Errorf("failed to unmarshal email config: %w", err)
	}

	if emailConfig.SMTPHost == "" || emailConfig.SMTPPort == 0 {
		return fmt.Errorf("SMTP host or port not configured")
	}
	if len(emailConfig.ToAddresses) == 0 {
		return fmt.Errorf("no recipient email addresses configured")
	}

	if _, err := mail.ParseAddress(emailConfig.FromAddress); err != nil {
		return fmt.Errorf("invalid from address: %w", err)
	}
	for _, addr := range emailConfig.ToAddresses {
		if _, err := mail.ParseAddress(addr); err != nil {
			return fmt.Errorf("invalid to address %s: %w", addr, err)
		}
	}

	if emailConfig.SMTPPassword != "" {
		if decrypted, err := crypto.Decrypt(emailConfig.SMTPPassword); err == nil {
			emailConfig.SMTPPassword = decrypted
		}
	}

	htmlBody, _, err := s.renderTestEmailTemplate()
	if err != nil {
		return fmt.Errorf("failed to render test email template: %w", err)
	}

	subject := "Test Email from Arcane"
	if err := notifications.SendEmail(ctx, emailConfig, subject, htmlBody); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

func (s *NotificationService) renderTestEmailTemplate() (string, string, error) {
	appURL := s.config.GetAppURL()
	logoURL := appURL + logoURLPath
	data := map[string]interface{}{
		"LogoURL": logoURL,
		"AppURL":  appURL,
	}

	htmlContent, err := resources.FS.ReadFile("email-templates/test_html.tmpl")
	if err != nil {
		return "", "", fmt.Errorf("failed to read HTML template: %w", err)
	}

	htmlTmpl, err := template.New("html").Parse(string(htmlContent))
	if err != nil {
		return "", "", fmt.Errorf("failed to parse HTML template: %w", err)
	}

	var htmlBuf bytes.Buffer
	if err := htmlTmpl.ExecuteTemplate(&htmlBuf, "root", data); err != nil {
		return "", "", fmt.Errorf("failed to execute HTML template: %w", err)
	}

	textContent, err := resources.FS.ReadFile("email-templates/test_text.tmpl")
	if err != nil {
		return "", "", fmt.Errorf("failed to read text template: %w", err)
	}

	textTmpl, err := template.New("text").Parse(string(textContent))
	if err != nil {
		return "", "", fmt.Errorf("failed to parse text template: %w", err)
	}

	var textBuf bytes.Buffer
	if err := textTmpl.ExecuteTemplate(&textBuf, "root", data); err != nil {
		return "", "", fmt.Errorf("failed to execute text template: %w", err)
	}

	return htmlBuf.String(), textBuf.String(), nil
}

func (s *NotificationService) logNotification(ctx context.Context, provider notification.NotificationProvider, imageRef, status string, errMsg *string, metadata base.JSON) {
	log := &notification.NotificationLog{
		Provider: provider,
		ImageRef: imageRef,
		Status:   status,
		Error:    errMsg,
		Metadata: metadata,
		SentAt:   time.Now(),
	}

	if err := s.store.CreateNotificationLog(ctx, *log); err != nil {
		slog.WarnContext(ctx, "Failed to log notification", "provider", string(provider), "error", err.Error())
	}
}

func (s *NotificationService) SendBatchImageUpdateNotification(ctx context.Context, updates map[string]*imageupdate.Response) error {
	if len(updates) == 0 {
		return nil
	}

	updatesWithChanges := make(map[string]*imageupdate.Response)
	for imageRef, update := range updates {
		if update != nil && update.HasUpdate {
			updatesWithChanges[imageRef] = update
		}
	}

	if len(updatesWithChanges) == 0 {
		return nil
	}

	// Send to Apprise if enabled
	if appriseErr := s.appriseService.SendBatchImageUpdateNotification(ctx, updatesWithChanges); appriseErr != nil {
		slog.WarnContext(ctx, "Failed to send Apprise notification", "error", appriseErr)
	}

	settings, err := s.GetAllSettings(ctx)
	if err != nil {
		return fmt.Errorf("failed to get notification settings: %w", err)
	}

	var errors []string
	for _, setting := range settings {
		if !setting.Enabled {
			continue
		}

		if !s.isEventEnabled(setting.Config, notification.NotificationEventImageUpdate) {
			continue
		}

		var sendErr error
		switch setting.Provider {
		case notification.NotificationProviderDiscord:
			sendErr = s.sendBatchDiscordNotification(ctx, updatesWithChanges, setting.Config)
		case notification.NotificationProviderEmail:
			sendErr = s.sendBatchEmailNotification(ctx, updatesWithChanges, setting.Config)
		case notification.NotificationProviderTelegram:
			sendErr = s.sendBatchTelegramNotification(ctx, updatesWithChanges, setting.Config)
		case notification.NotificationProviderSignal:
			sendErr = s.sendBatchSignalNotification(ctx, updatesWithChanges, setting.Config)
		case notification.NotificationProviderSlack:
			sendErr = s.sendBatchSlackNotification(ctx, updatesWithChanges, setting.Config)
		case notification.NotificationProviderNtfy:
			sendErr = s.sendBatchNtfyNotification(ctx, updatesWithChanges, setting.Config)
		case notification.NotificationProviderPushover:
			sendErr = s.sendBatchPushoverNotification(ctx, updatesWithChanges, setting.Config)
		case notification.NotificationProviderGotify:
			sendErr = s.sendBatchGotifyNotification(ctx, updatesWithChanges, setting.Config)
		case notification.NotificationProviderGeneric:
			sendErr = s.sendBatchGenericNotification(ctx, updatesWithChanges, setting.Config)
		default:
			slog.WarnContext(ctx, "Unknown notification provider", "provider", setting.Provider)
			continue
		}

		status := "success"
		var errMsg *string
		if sendErr != nil {
			status = "failed"
			msg := sendErr.Error()
			errMsg = &msg
			errors = append(errors, fmt.Sprintf("%s: %s", setting.Provider, msg))
		}

		imageRefs := make([]string, 0, len(updatesWithChanges))
		for ref := range updatesWithChanges {
			imageRefs = append(imageRefs, ref)
		}

		s.logNotification(ctx, setting.Provider, strings.Join(imageRefs, ", "), status, errMsg, base.JSON{
			"updateCount": len(updatesWithChanges),
			"eventType":   string(notification.NotificationEventImageUpdate),
			"batch":       true,
		})
	}

	if len(errors) > 0 {
		return fmt.Errorf("notification errors: %s", strings.Join(errors, "; "))
	}

	return nil
}

func (s *NotificationService) sendBatchDiscordNotification(ctx context.Context, updates map[string]*imageupdate.Response, config base.JSON) error {
	var discordConfig notification.DiscordConfig
	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal discord config: %w", err)
	}
	if err := json.Unmarshal(configBytes, &discordConfig); err != nil {
		return fmt.Errorf("failed to unmarshal discord config: %w", err)
	}

	// Decrypt token if encrypted
	if decrypted, err := crypto.Decrypt(discordConfig.Token); err == nil {
		discordConfig.Token = decrypted
	}

	// Build batch message content
	title := "Container Image Updates Available"
	description := fmt.Sprintf("%d container image(s) have updates available.", len(updates))
	if len(updates) == 1 {
		description = "1 container image has an update available."
	}

	message := fmt.Sprintf("**%s**\n\n%s\n\n", title, description)

	for imageRef, update := range updates {
		message += fmt.Sprintf("**%s**\n"+
			"• **Type:** %s\n"+
			"• **Current:** `%s`\n"+
			"• **Latest:** `%s`\n\n",
			imageRef,
			update.UpdateType,
			update.CurrentDigest,
			update.LatestDigest,
		)
	}

	if err := notifications.SendDiscord(ctx, discordConfig, message); err != nil {
		return fmt.Errorf("failed to send batch Discord notification: %w", err)
	}

	return nil
}

func (s *NotificationService) sendBatchTelegramNotification(ctx context.Context, updates map[string]*imageupdate.Response, config base.JSON) error {
	var telegramConfig notification.TelegramConfig
	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal telegram config: %w", err)
	}
	if err := json.Unmarshal(configBytes, &telegramConfig); err != nil {
		return fmt.Errorf("failed to unmarshal telegram config: %w", err)
	}

	// Decrypt bot token if encrypted
	if decrypted, err := crypto.Decrypt(telegramConfig.BotToken); err == nil {
		telegramConfig.BotToken = decrypted
	}

	// Build batch message content
	title := "Container Image Updates Available"
	description := fmt.Sprintf("%d container image(s) have updates available.", len(updates))
	if len(updates) == 1 {
		description = "1 container image has an update available."
	}

	message := fmt.Sprintf("*%s*\n\n%s\n\n", title, description)

	for imageRef, update := range updates {
		message += fmt.Sprintf("*%s*\n"+
			"• *Type:* %s\n"+
			"• *Current:* `%s`\n"+
			"• *Latest:* `%s`\n\n",
			imageRef,
			update.UpdateType,
			update.CurrentDigest,
			update.LatestDigest,
		)
	}

	if err := notifications.SendTelegram(ctx, telegramConfig, message); err != nil {
		return fmt.Errorf("failed to send batch Telegram notification: %w", err)
	}

	return nil
}

func (s *NotificationService) sendBatchEmailNotification(ctx context.Context, updates map[string]*imageupdate.Response, config base.JSON) error {
	var emailConfig notification.EmailConfig
	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal email config: %w", err)
	}
	if err := json.Unmarshal(configBytes, &emailConfig); err != nil {
		return fmt.Errorf("failed to unmarshal email config: %w", err)
	}

	if emailConfig.SMTPHost == "" || emailConfig.SMTPPort == 0 {
		return fmt.Errorf("SMTP host or port not configured")
	}
	if len(emailConfig.ToAddresses) == 0 {
		return fmt.Errorf("no recipient email addresses configured")
	}

	if _, err := mail.ParseAddress(emailConfig.FromAddress); err != nil {
		return fmt.Errorf("invalid from address: %w", err)
	}
	for _, addr := range emailConfig.ToAddresses {
		if _, err := mail.ParseAddress(addr); err != nil {
			return fmt.Errorf("invalid to address %s: %w", addr, err)
		}
	}

	if emailConfig.SMTPPassword != "" {
		if decrypted, err := crypto.Decrypt(emailConfig.SMTPPassword); err == nil {
			emailConfig.SMTPPassword = decrypted
		} else {
			slog.Warn("Failed to decrypt email SMTP password, using raw value (may be unencrypted legacy value)", "error", err)
		}
	}

	htmlBody, _, err := s.renderBatchEmailTemplate(updates)
	if err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}

	updateCount := len(updates)
	subject := fmt.Sprintf("%d Image Update%s Available", updateCount, func() string {
		if updateCount > 1 {
			return "s"
		}
		return ""
	}())
	if err := notifications.SendEmail(ctx, emailConfig, subject, htmlBody); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

func (s *NotificationService) renderBatchEmailTemplate(updates map[string]*imageupdate.Response) (string, string, error) {
	// Build list of image names
	imageList := make([]string, 0, len(updates))
	for imageRef := range updates {
		imageList = append(imageList, imageRef)
	}

	appURL := s.config.GetAppURL()
	logoURL := appURL + logoURLPath
	data := map[string]interface{}{
		"LogoURL":     logoURL,
		"AppURL":      appURL,
		"UpdateCount": len(updates),
		"CheckTime":   time.Now().Format(time.RFC1123),
		"ImageList":   imageList,
	}

	htmlContent, err := resources.FS.ReadFile("email-templates/batch-image-updates_html.tmpl")
	if err != nil {
		return "", "", fmt.Errorf("failed to read HTML template: %w", err)
	}

	htmlTmpl, err := template.New("html").Parse(string(htmlContent))
	if err != nil {
		return "", "", fmt.Errorf("failed to parse HTML template: %w", err)
	}

	var htmlBuf bytes.Buffer
	if err := htmlTmpl.ExecuteTemplate(&htmlBuf, "root", data); err != nil {
		return "", "", fmt.Errorf("failed to execute HTML template: %w", err)
	}

	textContent, err := resources.FS.ReadFile("email-templates/batch-image-updates_text.tmpl")
	if err != nil {
		return "", "", fmt.Errorf("failed to read text template: %w", err)
	}

	textTmpl, err := template.New("text").Parse(string(textContent))
	if err != nil {
		return "", "", fmt.Errorf("failed to parse text template: %w", err)
	}

	var textBuf bytes.Buffer
	if err := textTmpl.ExecuteTemplate(&textBuf, "root", data); err != nil {
		return "", "", fmt.Errorf("failed to execute text template: %w", err)
	}

	return htmlBuf.String(), textBuf.String(), nil
}

func (s *NotificationService) sendSignalNotification(ctx context.Context, imageRef string, updateInfo *imageupdate.Response, config base.JSON) error {
	var signalConfig notification.SignalConfig
	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal Signal config: %w", err)
	}
	if err := json.Unmarshal(configBytes, &signalConfig); err != nil {
		return fmt.Errorf("failed to unmarshal Signal config: %w", err)
	}

	if signalConfig.Host == "" {
		return fmt.Errorf("signal host not configured")
	}
	if signalConfig.Port == 0 {
		return fmt.Errorf("signal port not configured")
	}
	if signalConfig.Source == "" {
		return fmt.Errorf("signal source phone number not configured")
	}
	if len(signalConfig.Recipients) == 0 {
		return fmt.Errorf("no signal recipients configured")
	}

	// Validate authentication
	hasBasicAuth := signalConfig.User != "" && signalConfig.Password != ""
	hasTokenAuth := signalConfig.Token != ""
	if !hasBasicAuth && !hasTokenAuth {
		return fmt.Errorf("signal requires either basic auth (user/password) or token authentication")
	}
	if hasBasicAuth && hasTokenAuth {
		return fmt.Errorf("signal cannot use both basic auth and token authentication simultaneously")
	}

	// Decrypt sensitive fields if encrypted
	if signalConfig.Password != "" {
		if decrypted, err := crypto.Decrypt(signalConfig.Password); err == nil {
			signalConfig.Password = decrypted
		} else {
			slog.Warn("Failed to decrypt Signal password, using raw value (may be unencrypted legacy value)", "error", err)
		}
	}
	if signalConfig.Token != "" {
		if decrypted, err := crypto.Decrypt(signalConfig.Token); err == nil {
			signalConfig.Token = decrypted
		} else {
			slog.Warn("Failed to decrypt Signal token, using raw value (may be unencrypted legacy value)", "error", err)
		}
	}

	// Build message content
	updateStatus := "No Update"
	if updateInfo.HasUpdate {
		updateStatus = "⚠️ Update Available"
	}

	message := fmt.Sprintf("🔔 Container Image Update Notification\n\n"+
		"Image: %s\n"+
		"Status: %s\n"+
		"Update Type: %s\n",
		imageRef, updateStatus, updateInfo.UpdateType)

	if updateInfo.CurrentDigest != "" {
		message += fmt.Sprintf("Current Digest: %s\n", updateInfo.CurrentDigest)
	}
	if updateInfo.LatestDigest != "" {
		message += fmt.Sprintf("Latest Digest: %s\n", updateInfo.LatestDigest)
	}

	if err := notifications.SendSignal(ctx, signalConfig, message); err != nil {
		return fmt.Errorf("failed to send Signal notification: %w", err)
	}

	return nil
}

func (s *NotificationService) sendSignalContainerUpdateNotification(ctx context.Context, containerName, imageRef, oldDigest, newDigest string, config base.JSON) error {
	var signalConfig notification.SignalConfig
	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal Signal config: %w", err)
	}
	if err := json.Unmarshal(configBytes, &signalConfig); err != nil {
		return fmt.Errorf("failed to unmarshal Signal config: %w", err)
	}

	if signalConfig.Host == "" {
		return fmt.Errorf("signal host not configured")
	}
	if signalConfig.Port == 0 {
		return fmt.Errorf("signal port not configured")
	}
	if signalConfig.Source == "" {
		return fmt.Errorf("signal source phone number not configured")
	}
	if len(signalConfig.Recipients) == 0 {
		return fmt.Errorf("no signal recipients configured")
	}

	// Validate authentication
	hasBasicAuth := signalConfig.User != "" && signalConfig.Password != ""
	hasTokenAuth := signalConfig.Token != ""
	if !hasBasicAuth && !hasTokenAuth {
		return fmt.Errorf("signal requires either basic auth (user/password) or token authentication")
	}
	if hasBasicAuth && hasTokenAuth {
		return fmt.Errorf("signal cannot use both basic auth and token authentication simultaneously")
	}

	// Decrypt sensitive fields if encrypted
	if signalConfig.Password != "" {
		if decrypted, err := crypto.Decrypt(signalConfig.Password); err == nil {
			signalConfig.Password = decrypted
		} else {
			slog.Warn("Failed to decrypt Signal password, using raw value (may be unencrypted legacy value)", "error", err)
		}
	}
	if signalConfig.Token != "" {
		if decrypted, err := crypto.Decrypt(signalConfig.Token); err == nil {
			signalConfig.Token = decrypted
		} else {
			slog.Warn("Failed to decrypt Signal token, using raw value (may be unencrypted legacy value)", "error", err)
		}
	}

	// Build message content
	message := fmt.Sprintf("✅ Container Successfully Updated\n\n"+
		"Your container has been updated with the latest image version.\n\n"+
		"Container: %s\n"+
		"Image: %s\n"+
		"Status: ✅ Updated Successfully\n",
		containerName, imageRef)

	if oldDigest != "" {
		message += fmt.Sprintf("Previous Version: %s\n", oldDigest)
	}
	if newDigest != "" {
		message += fmt.Sprintf("Current Version: %s\n", newDigest)
	}

	if err := notifications.SendSignal(ctx, signalConfig, message); err != nil {
		return fmt.Errorf("failed to send Signal notification: %w", err)
	}

	return nil
}

func (s *NotificationService) sendBatchSignalNotification(ctx context.Context, updates map[string]*imageupdate.Response, config base.JSON) error {
	var signalConfig notification.SignalConfig
	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal signal config: %w", err)
	}
	if err := json.Unmarshal(configBytes, &signalConfig); err != nil {
		return fmt.Errorf("failed to unmarshal signal config: %w", err)
	}

	// Validate authentication
	hasBasicAuth := signalConfig.User != "" && signalConfig.Password != ""
	hasTokenAuth := signalConfig.Token != ""
	if !hasBasicAuth && !hasTokenAuth {
		return fmt.Errorf("signal requires either basic auth (user/password) or token authentication")
	}
	if hasBasicAuth && hasTokenAuth {
		return fmt.Errorf("signal cannot use both basic auth and token authentication simultaneously")
	}

	// Decrypt sensitive fields if encrypted
	if signalConfig.Password != "" {
		if decrypted, err := crypto.Decrypt(signalConfig.Password); err == nil {
			signalConfig.Password = decrypted
		}
	}
	if signalConfig.Token != "" {
		if decrypted, err := crypto.Decrypt(signalConfig.Token); err == nil {
			signalConfig.Token = decrypted
		}
	}

	// Build batch message content
	title := "Container Image Updates Available"
	description := fmt.Sprintf("%d container image(s) have updates available.", len(updates))
	if len(updates) == 1 {
		description = "1 container image has an update available."
	}

	message := fmt.Sprintf("%s\n\n%s\n\n", title, description)

	for imageRef, update := range updates {
		message += fmt.Sprintf("%s\n"+
			"• Type: %s\n"+
			"• Current: %s\n"+
			"• Latest: %s\n\n",
			imageRef,
			update.UpdateType,
			update.CurrentDigest,
			update.LatestDigest,
		)
	}

	if err := notifications.SendSignal(ctx, signalConfig, message); err != nil {
		return fmt.Errorf("failed to send batch Signal notification: %w", err)
	}

	return nil
}

func (s *NotificationService) sendSlackNotification(ctx context.Context, imageRef string, updateInfo *imageupdate.Response, config base.JSON) error {
	var slackConfig notification.SlackConfig
	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack config: %w", err)
	}
	if err := json.Unmarshal(configBytes, &slackConfig); err != nil {
		return fmt.Errorf("failed to unmarshal Slack config: %w", err)
	}

	if slackConfig.Token == "" {
		return fmt.Errorf("slack token not configured")
	}

	// Decrypt token if encrypted
	if slackConfig.Token != "" {
		if decrypted, err := crypto.Decrypt(slackConfig.Token); err == nil {
			slackConfig.Token = decrypted
		} else {
			slog.Warn("Failed to decrypt Slack token, using raw value (may be unencrypted legacy value)", "error", err)
		}
	}

	// Build message content
	updateStatus := "No Update"
	if updateInfo.HasUpdate {
		updateStatus = "⚠️ Update Available"
	}

	message := fmt.Sprintf("🔔 *Container Image Update Notification*\n\n"+
		"*Image:* %s\n"+
		"*Status:* %s\n"+
		"*Update Type:* %s\n",
		imageRef, updateStatus, updateInfo.UpdateType)

	if updateInfo.CurrentDigest != "" {
		message += fmt.Sprintf("*Current Digest:* `%s`\n", updateInfo.CurrentDigest)
	}
	if updateInfo.LatestDigest != "" {
		message += fmt.Sprintf("*Latest Digest:* `%s`\n", updateInfo.LatestDigest)
	}

	if err := notifications.SendSlack(ctx, slackConfig, message); err != nil {
		return fmt.Errorf("failed to send Slack notification: %w", err)
	}

	return nil
}

func (s *NotificationService) sendSlackContainerUpdateNotification(ctx context.Context, containerName, imageRef, oldDigest, newDigest string, config base.JSON) error {
	var slackConfig notification.SlackConfig
	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack config: %w", err)
	}
	if err := json.Unmarshal(configBytes, &slackConfig); err != nil {
		return fmt.Errorf("failed to unmarshal Slack config: %w", err)
	}

	if slackConfig.Token == "" {
		return fmt.Errorf("slack token not configured")
	}

	// Decrypt token if encrypted
	if slackConfig.Token != "" {
		if decrypted, err := crypto.Decrypt(slackConfig.Token); err == nil {
			slackConfig.Token = decrypted
		} else {
			slog.Warn("Failed to decrypt Slack token, using raw value (may be unencrypted legacy value)", "error", err)
		}
	}

	// Build message content
	message := fmt.Sprintf("✅ *Container Successfully Updated*\n\n"+
		"Your container has been updated with the latest image version.\n\n"+
		"*Container:* %s\n"+
		"*Image:* %s\n"+
		"*Status:* ✅ Updated Successfully\n",
		containerName, imageRef)

	if oldDigest != "" {
		message += fmt.Sprintf("*Previous Version:* `%s`\n", oldDigest)
	}
	if newDigest != "" {
		message += fmt.Sprintf("*Current Version:* `%s`\n", newDigest)
	}

	if err := notifications.SendSlack(ctx, slackConfig, message); err != nil {
		return fmt.Errorf("failed to send Slack notification: %w", err)
	}

	return nil
}

func (s *NotificationService) sendBatchSlackNotification(ctx context.Context, updates map[string]*imageupdate.Response, config base.JSON) error {
	var slackConfig notification.SlackConfig
	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal slack config: %w", err)
	}
	if err := json.Unmarshal(configBytes, &slackConfig); err != nil {
		return fmt.Errorf("failed to unmarshal slack config: %w", err)
	}

	// Decrypt token if encrypted
	if decrypted, err := crypto.Decrypt(slackConfig.Token); err == nil {
		slackConfig.Token = decrypted
	}

	// Build batch message content
	title := "*Container Image Updates Available*"
	description := fmt.Sprintf("%d container image(s) have updates available.", len(updates))
	if len(updates) == 1 {
		description = "1 container image has an update available."
	}

	message := fmt.Sprintf("%s\n\n%s\n\n", title, description)

	for imageRef, update := range updates {
		message += fmt.Sprintf("*%s*\n"+
			"• *Type:* %s\n"+
			"• *Current:* `%s`\n"+
			"• *Latest:* `%s`\n\n",
			imageRef,
			update.UpdateType,
			update.CurrentDigest,
			update.LatestDigest,
		)
	}

	if err := notifications.SendSlack(ctx, slackConfig, message); err != nil {
		return fmt.Errorf("failed to send batch Slack notification: %w", err)
	}

	return nil
}

func (s *NotificationService) sendNtfyNotification(ctx context.Context, imageRef string, updateInfo *imageupdate.Response, config base.JSON) error {
	var ntfyConfig notification.NtfyConfig
	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal Ntfy config: %w", err)
	}
	if err := json.Unmarshal(configBytes, &ntfyConfig); err != nil {
		return fmt.Errorf("failed to unmarshal Ntfy config: %w", err)
	}

	if ntfyConfig.Topic == "" {
		return fmt.Errorf("ntfy topic is required")
	}

	// Decrypt password if encrypted
	if ntfyConfig.Password != "" {
		if decrypted, err := crypto.Decrypt(ntfyConfig.Password); err == nil {
			ntfyConfig.Password = decrypted
		} else {
			slog.Warn("Failed to decrypt Ntfy password, using raw value (may be unencrypted legacy value)", "error", err)
		}
	}

	// Build message content
	updateStatus := "No Update"
	if updateInfo.HasUpdate {
		updateStatus = "⚠️ Update Available"
	}

	message := fmt.Sprintf("🔔 Container Image Update Notification\n\n"+
		"Image: %s\n"+
		"Status: %s\n"+
		"Update Type: %s\n",
		imageRef, updateStatus, updateInfo.UpdateType)

	if updateInfo.CurrentDigest != "" {
		message += fmt.Sprintf("Current Digest: %s\n", updateInfo.CurrentDigest)
	}
	if updateInfo.LatestDigest != "" {
		message += fmt.Sprintf("Latest Digest: %s\n", updateInfo.LatestDigest)
	}

	if err := notifications.SendNtfy(ctx, ntfyConfig, message); err != nil {
		return fmt.Errorf("failed to send Ntfy notification: %w", err)
	}

	return nil
}

func (s *NotificationService) sendNtfyContainerUpdateNotification(ctx context.Context, containerName, imageRef, oldDigest, newDigest string, config base.JSON) error {
	var ntfyConfig notification.NtfyConfig
	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal Ntfy config: %w", err)
	}
	if err := json.Unmarshal(configBytes, &ntfyConfig); err != nil {
		return fmt.Errorf("failed to unmarshal Ntfy config: %w", err)
	}

	if ntfyConfig.Topic == "" {
		return fmt.Errorf("ntfy topic is required")
	}

	// Decrypt password if encrypted
	if ntfyConfig.Password != "" {
		if decrypted, err := crypto.Decrypt(ntfyConfig.Password); err == nil {
			ntfyConfig.Password = decrypted
		} else {
			slog.Warn("Failed to decrypt Ntfy password, using raw value (may be unencrypted legacy value)", "error", err)
		}
	}

	// Build message content
	message := fmt.Sprintf("✅ Container Successfully Updated\n\n"+
		"Your container has been updated with the latest image version.\n\n"+
		"Container: %s\n"+
		"Image: %s\n"+
		"Status: ✅ Updated Successfully\n",
		containerName, imageRef)

	if oldDigest != "" {
		message += fmt.Sprintf("Previous Version: %s\n", oldDigest)
	}
	if newDigest != "" {
		message += fmt.Sprintf("Current Version: %s\n", newDigest)
	}

	if err := notifications.SendNtfy(ctx, ntfyConfig, message); err != nil {
		return fmt.Errorf("failed to send Ntfy notification: %w", err)
	}

	return nil
}

func (s *NotificationService) sendBatchNtfyNotification(ctx context.Context, updates map[string]*imageupdate.Response, config base.JSON) error {
	var ntfyConfig notification.NtfyConfig
	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal ntfy config: %w", err)
	}
	if err := json.Unmarshal(configBytes, &ntfyConfig); err != nil {
		return fmt.Errorf("failed to unmarshal ntfy config: %w", err)
	}

	// Decrypt password if encrypted
	if ntfyConfig.Password != "" {
		if decrypted, err := crypto.Decrypt(ntfyConfig.Password); err == nil {
			ntfyConfig.Password = decrypted
		}
	}

	// Build batch message content
	title := "Container Image Updates Available"
	description := fmt.Sprintf("%d container image(s) have updates available.", len(updates))
	if len(updates) == 1 {
		description = "1 container image has an update available."
	}

	message := fmt.Sprintf("%s\n\n%s\n\n", title, description)

	for imageRef, update := range updates {
		message += fmt.Sprintf("%s\n"+
			"• Type: %s\n"+
			"• Current: %s\n"+
			"• Latest: %s\n\n",
			imageRef,
			update.UpdateType,
			update.CurrentDigest,
			update.LatestDigest,
		)
	}

	if err := notifications.SendNtfy(ctx, ntfyConfig, message); err != nil {
		return fmt.Errorf("failed to send batch Ntfy notification: %w", err)
	}

	return nil
}

func (s *NotificationService) sendPushoverNotification(ctx context.Context, imageRef string, updateInfo *imageupdate.Response, config base.JSON) error {
	var pushoverConfig notification.PushoverConfig
	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal Pushover config: %w", err)
	}
	if err := json.Unmarshal(configBytes, &pushoverConfig); err != nil {
		return fmt.Errorf("failed to unmarshal Pushover config: %w", err)
	}

	if pushoverConfig.Token == "" {
		return fmt.Errorf("pushover API token not configured")
	}
	if pushoverConfig.User == "" {
		return fmt.Errorf("pushover user key not configured")
	}

	if pushoverConfig.Token != "" {
		if decrypted, err := crypto.Decrypt(pushoverConfig.Token); err == nil {
			pushoverConfig.Token = decrypted
		} else {
			slog.Warn("Failed to decrypt Pushover token, using raw value (may be unencrypted legacy value)", "error", err)
		}
	}

	updateStatus := "No Update"
	if updateInfo.HasUpdate {
		updateStatus = "⚠️ Update Available"
	}

	message := fmt.Sprintf("🔔 Container Image Update Notification\n\n"+
		"Image: %s\n"+
		"Status: %s\n"+
		"Update Type: %s\n",
		imageRef, updateStatus, updateInfo.UpdateType)

	if updateInfo.CurrentDigest != "" {
		message += fmt.Sprintf("Current Digest: %s\n", updateInfo.CurrentDigest)
	}
	if updateInfo.LatestDigest != "" {
		message += fmt.Sprintf("Latest Digest: %s\n", updateInfo.LatestDigest)
	}

	if err := notifications.SendPushover(ctx, pushoverConfig, message); err != nil {
		return fmt.Errorf("failed to send Pushover notification: %w", err)
	}

	return nil
}

func (s *NotificationService) sendPushoverContainerUpdateNotification(ctx context.Context, containerName, imageRef, oldDigest, newDigest string, config base.JSON) error {
	var pushoverConfig notification.PushoverConfig
	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal Pushover config: %w", err)
	}
	if err := json.Unmarshal(configBytes, &pushoverConfig); err != nil {
		return fmt.Errorf("failed to unmarshal Pushover config: %w", err)
	}

	if pushoverConfig.Token == "" {
		return fmt.Errorf("pushover API token not configured")
	}
	if pushoverConfig.User == "" {
		return fmt.Errorf("pushover user key not configured")
	}

	if pushoverConfig.Token != "" {
		if decrypted, err := crypto.Decrypt(pushoverConfig.Token); err == nil {
			pushoverConfig.Token = decrypted
		} else {
			slog.Warn("Failed to decrypt Pushover token, using raw value (may be unencrypted legacy value)", "error", err)
		}
	}

	message := fmt.Sprintf("✅ Container Successfully Updated\n\n"+
		"Your container has been updated with the latest image version.\n\n"+
		"Container: %s\n"+
		"Image: %s\n"+
		"Status: ✅ Updated Successfully\n",
		containerName, imageRef)

	if oldDigest != "" {
		message += fmt.Sprintf("Previous Version: %s\n", oldDigest)
	}
	if newDigest != "" {
		message += fmt.Sprintf("Current Version: %s\n", newDigest)
	}

	if err := notifications.SendPushover(ctx, pushoverConfig, message); err != nil {
		return fmt.Errorf("failed to send Pushover notification: %w", err)
	}

	return nil
}

func (s *NotificationService) sendBatchPushoverNotification(ctx context.Context, updates map[string]*imageupdate.Response, config base.JSON) error {
	var pushoverConfig notification.PushoverConfig
	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal pushover config: %w", err)
	}
	if err := json.Unmarshal(configBytes, &pushoverConfig); err != nil {
		return fmt.Errorf("failed to unmarshal pushover config: %w", err)
	}

	if pushoverConfig.Token != "" {
		if decrypted, err := crypto.Decrypt(pushoverConfig.Token); err == nil {
			pushoverConfig.Token = decrypted
		}
	}

	// Build batch message content
	title := "Container Image Updates Available"
	description := fmt.Sprintf("%d container image(s) have updates available.", len(updates))
	if len(updates) == 1 {
		description = "1 container image has an update available."
	}

	message := fmt.Sprintf("%s\n\n%s\n\n", title, description)

	for imageRef, update := range updates {
		message += fmt.Sprintf("%s\n"+
			"• Type: %s\n"+
			"• Current: %s\n"+
			"• Latest: %s\n\n",
			imageRef,
			update.UpdateType,
			update.CurrentDigest,
			update.LatestDigest,
		)
	}

	if err := notifications.SendPushover(ctx, pushoverConfig, message); err != nil {
		return fmt.Errorf("failed to send batch Pushover notification: %w", err)
	}

	return nil
}

func (s *NotificationService) sendGenericNotification(ctx context.Context, imageRef string, updateInfo *imageupdate.Response, config base.JSON) error {
	var genericConfig notification.GenericConfig
	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal Generic config: %w", err)
	}
	if err := json.Unmarshal(configBytes, &genericConfig); err != nil {
		return fmt.Errorf("failed to unmarshal Generic config: %w", err)
	}

	if genericConfig.WebhookURL == "" {
		return fmt.Errorf("webhook URL not configured")
	}

	// Build message content
	updateStatus := "No Update"
	if updateInfo.HasUpdate {
		updateStatus = "Update Available"
	}

	message := fmt.Sprintf("Container Image Update Notification\n\n"+
		"Image: %s\n"+
		"Status: %s\n"+
		"Update Type: %s\n",
		imageRef, updateStatus, updateInfo.UpdateType)

	if updateInfo.CurrentDigest != "" {
		message += fmt.Sprintf("Current Digest: %s\n", updateInfo.CurrentDigest)
	}
	if updateInfo.LatestDigest != "" {
		message += fmt.Sprintf("Latest Digest: %s\n", updateInfo.LatestDigest)
	}

	// Use SendGenericWithTitle to include a title
	title := "Container Image Update"
	if err := notifications.SendGenericWithTitle(ctx, genericConfig, title, message); err != nil {
		return fmt.Errorf("failed to send Generic webhook notification: %w", err)
	}

	return nil
}

func (s *NotificationService) sendGenericContainerUpdateNotification(ctx context.Context, containerName, imageRef, oldDigest, newDigest string, config base.JSON) error {
	var genericConfig notification.GenericConfig
	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal Generic config: %w", err)
	}
	if err := json.Unmarshal(configBytes, &genericConfig); err != nil {
		return fmt.Errorf("failed to unmarshal Generic config: %w", err)
	}

	if genericConfig.WebhookURL == "" {
		return fmt.Errorf("webhook URL not configured")
	}

	// Build message content
	message := fmt.Sprintf("Container Successfully Updated\n\n"+
		"Your container has been updated with the latest image version.\n\n"+
		"Container: %s\n"+
		"Image: %s\n"+
		"Status: Updated Successfully\n",
		containerName, imageRef)

	if oldDigest != "" {
		message += fmt.Sprintf("Previous Version: %s\n", oldDigest)
	}
	if newDigest != "" {
		message += fmt.Sprintf("Current Version: %s\n", newDigest)
	}

	// Use SendGenericWithTitle to include a title
	title := "Container Updated"
	if err := notifications.SendGenericWithTitle(ctx, genericConfig, title, message); err != nil {
		return fmt.Errorf("failed to send Generic webhook notification: %w", err)
	}

	return nil
}

func (s *NotificationService) sendBatchGenericNotification(ctx context.Context, updates map[string]*imageupdate.Response, config base.JSON) error {
	var genericConfig notification.GenericConfig
	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal generic config: %w", err)
	}
	if err := json.Unmarshal(configBytes, &genericConfig); err != nil {
		return fmt.Errorf("failed to unmarshal generic config: %w", err)
	}

	if genericConfig.WebhookURL == "" {
		return fmt.Errorf("webhook URL not configured")
	}

	// Build batch message content
	title := "Container Image Updates Available"
	description := fmt.Sprintf("%d container image(s) have updates available.", len(updates))
	if len(updates) == 1 {
		description = "1 container image has an update available."
	}

	message := fmt.Sprintf("%s\n\n", description)

	for imageRef, update := range updates {
		message += fmt.Sprintf("%s\n"+
			"• Type: %s\n"+
			"• Current: %s\n"+
			"• Latest: %s\n\n",
			imageRef,
			update.UpdateType,
			update.CurrentDigest,
			update.LatestDigest,
		)
	}

	if err := notifications.SendGenericWithTitle(ctx, genericConfig, title, message); err != nil {
		return fmt.Errorf("failed to send batch Generic webhook notification: %w", err)
	}

	return nil
}

func (s *NotificationService) sendGotifyNotification(ctx context.Context, imageRef string, updateInfo *imageupdate.Response, config base.JSON) error {
	var gotifyConfig notification.GotifyConfig
	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal Gotify config: %w", err)
	}
	if err := json.Unmarshal(configBytes, &gotifyConfig); err != nil {
		return fmt.Errorf("failed to unmarshal Gotify config: %w", err)
	}

	if gotifyConfig.Token != "" {
		if decrypted, err := crypto.Decrypt(gotifyConfig.Token); err == nil {
			gotifyConfig.Token = decrypted
		} else {
			slog.Warn("Failed to decrypt Gotify token, using raw value (may be unencrypted legacy value)", "error", err)
		}
	}

	updateStatus := "No Update"
	if updateInfo.HasUpdate {
		updateStatus = "⚠️ Update Available"
	}

	message := fmt.Sprintf("🔔 Container Image Update Notification\n\n"+
		"Image: %s\n"+
		"Status: %s\n"+
		"Update Type: %s\n",
		imageRef, updateStatus, updateInfo.UpdateType)

	if updateInfo.CurrentDigest != "" {
		message += fmt.Sprintf("Current Digest: %s\n", updateInfo.CurrentDigest)
	}
	if updateInfo.LatestDigest != "" {
		message += fmt.Sprintf("Latest Digest: %s\n", updateInfo.LatestDigest)
	}

	if err := notifications.SendGotify(ctx, gotifyConfig, message); err != nil {
		return fmt.Errorf("failed to send Gotify notification: %w", err)
	}

	return nil
}

func (s *NotificationService) sendGotifyContainerUpdateNotification(ctx context.Context, containerName, imageRef, oldDigest, newDigest string, config base.JSON) error {
	var gotifyConfig notification.GotifyConfig
	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal Gotify config: %w", err)
	}
	if err := json.Unmarshal(configBytes, &gotifyConfig); err != nil {
		return fmt.Errorf("failed to unmarshal Gotify config: %w", err)
	}

	if gotifyConfig.Token != "" {
		if decrypted, err := crypto.Decrypt(gotifyConfig.Token); err == nil {
			gotifyConfig.Token = decrypted
		} else {
			slog.Warn("Failed to decrypt Gotify token, using raw value (may be unencrypted legacy value)", "error", err)
		}
	}

	message := fmt.Sprintf("✅ Container Successfully Updated\n\n"+
		"Your container has been updated with the latest image version.\n\n"+
		"Container: %s\n"+
		"Image: %s\n"+
		"Status: ✅ Updated Successfully\n",
		containerName, imageRef)

	if oldDigest != "" {
		message += fmt.Sprintf("Previous Version: %s\n", oldDigest)
	}
	if newDigest != "" {
		message += fmt.Sprintf("Current Version: %s\n", newDigest)
	}

	if err := notifications.SendGotify(ctx, gotifyConfig, message); err != nil {
		return fmt.Errorf("failed to send Gotify notification: %w", err)
	}

	return nil
}

func (s *NotificationService) sendBatchGotifyNotification(ctx context.Context, updates map[string]*imageupdate.Response, config base.JSON) error {
	var gotifyConfig notification.GotifyConfig
	configBytes, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal gotify config: %w", err)
	}
	if err := json.Unmarshal(configBytes, &gotifyConfig); err != nil {
		return fmt.Errorf("failed to unmarshal gotify config: %w", err)
	}

	if gotifyConfig.Token != "" {
		if decrypted, err := crypto.Decrypt(gotifyConfig.Token); err == nil {
			gotifyConfig.Token = decrypted
		}
	}

	// Build batch message content
	title := "Container Image Updates Available"
	description := fmt.Sprintf("%d container image(s) have updates available.", len(updates))
	if len(updates) == 1 {
		description = "1 container image has an update available."
	}

	message := fmt.Sprintf("%s\n\n%s\n\n", title, description)

	for imageRef, update := range updates {
		message += fmt.Sprintf("%s\n"+
			"• Type: %s\n"+
			"• Current: %s\n"+
			"• Latest: %s\n\n",
			imageRef,
			update.UpdateType,
			update.CurrentDigest,
			update.LatestDigest,
		)
	}

	if err := notifications.SendGotify(ctx, gotifyConfig, message); err != nil {
		return fmt.Errorf("failed to send batch Gotify notification: %w", err)
	}

	return nil
}

// MigrateDiscordWebhookUrlToFields migrates legacy Discord webhookUrl to separate webhookId and token fields.
// This should be called during bootstrap to ensure existing Discord configurations are preserved.
func (s *NotificationService) MigrateDiscordWebhookUrlToFields(ctx context.Context) error {
	setting, err := s.store.GetNotificationSettingByProvider(ctx, notification.NotificationProviderDiscord)
	if err != nil {
		return fmt.Errorf("failed to query Discord settings: %w", err)
	}
	if setting == nil {
		// No Discord config exists, nothing to migrate.
		return nil
	}

	var discordConfig notification.DiscordConfig
	configBytes, err := json.Marshal(setting.Config)
	if err != nil {
		return fmt.Errorf("failed to marshal Discord config: %w", err)
	}
	if err := json.Unmarshal(configBytes, &discordConfig); err != nil {
		slog.WarnContext(ctx, "Failed to parse Discord config for migration", "error", err)
		return nil
	}

	// Check if already migrated (has webhookId and token)
	if discordConfig.WebhookID != "" && discordConfig.Token != "" {
		slog.DebugContext(ctx, "Discord config already migrated, skipping")
		return nil
	}

	// Check for legacy webhookUrl field
	var legacyConfig struct {
		WebhookUrl string                                      `json:"webhookUrl"`
		Username   string                                      `json:"username,omitempty"`
		AvatarURL  string                                      `json:"avatarUrl,omitempty"`
		Events     map[notification.NotificationEventType]bool `json:"events,omitempty"`
	}
	if err := json.Unmarshal(configBytes, &legacyConfig); err != nil {
		slog.WarnContext(ctx, "Failed to parse legacy Discord config structure", "error", err)
		return nil
	}

	if legacyConfig.WebhookUrl == "" {
		slog.DebugContext(ctx, "No legacy webhookUrl to migrate")
		return nil
	}

	// Parse webhook URL: https://discord.com/api/webhooks/{id}/{token}
	parts := strings.Split(legacyConfig.WebhookUrl, "/webhooks/")
	if len(parts) != 2 {
		slog.WarnContext(ctx, "Invalid Discord webhook URL format, skipping migration", "url", legacyConfig.WebhookUrl)
		return nil
	}

	webhookParts := strings.Split(parts[1], "/")
	if len(webhookParts) != 2 {
		slog.WarnContext(ctx, "Invalid Discord webhook URL format, skipping migration", "url", legacyConfig.WebhookUrl)
		return nil
	}

	webhookID := webhookParts[0]
	token := webhookParts[1]

	slog.InfoContext(ctx, "Migrating legacy Discord webhookUrl to webhookId and token")

	// Encrypt token before storing
	encryptedToken, err := crypto.Encrypt(token)
	if err != nil {
		return fmt.Errorf("failed to encrypt Discord token: %w", err)
	}

	// Update with new structure
	newConfig := notification.DiscordConfig{
		WebhookID: webhookID,
		Token:     encryptedToken,
		Username:  legacyConfig.Username,
		AvatarURL: legacyConfig.AvatarURL,
		Events:    legacyConfig.Events,
	}

	var configMap base.JSON
	newConfigBytes, err := json.Marshal(newConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal new Discord config: %w", err)
	}
	if err = json.Unmarshal(newConfigBytes, &configMap); err != nil {
		return fmt.Errorf("failed to unmarshal new Discord config to JSON: %w", err)
	}

	_, err = s.store.UpsertNotificationSetting(ctx, notification.NotificationProviderDiscord, setting.Enabled, configMap)
	if err != nil {
		return fmt.Errorf("failed to save migrated Discord config: %w", err)
	}

	slog.InfoContext(ctx, "Successfully migrated Discord config")
	return nil
}
