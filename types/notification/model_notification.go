package notification

import (
	"time"

	"github.com/getarcaneapp/arcane/types/base"
)

type NotificationProvider = Provider

type NotificationEventType string

const (
	NotificationEventImageUpdate     NotificationEventType = "image_update"
	NotificationEventContainerUpdate NotificationEventType = "container_update"
)

type EmailTLSMode string

const (
	EmailTLSModeNone     EmailTLSMode = "none"
	EmailTLSModeStartTLS EmailTLSMode = "starttls"
	EmailTLSModeSSL      EmailTLSMode = "ssl"
)

func IsValidNotificationProvider(provider NotificationProvider) bool {
	switch provider {
	case NotificationProviderDiscord,
		NotificationProviderEmail,
		NotificationProviderTelegram,
		NotificationProviderSignal,
		NotificationProviderSlack,
		NotificationProviderNtfy,
		NotificationProviderPushover,
		NotificationProviderGotify,
		NotificationProviderGeneric:
		return true
	default:
		return false
	}
}

type NotificationSettings struct {
	ID        uint                 `json:"id"`
	Provider  NotificationProvider `json:"provider"`
	Enabled   bool                 `json:"enabled"`
	Config    base.JSON            `json:"config"`
	CreatedAt time.Time            `json:"createdAt"`
	UpdatedAt time.Time            `json:"updatedAt"`
}

type NotificationLog struct {
	ID        uint                 `json:"id"`
	Provider  NotificationProvider `json:"provider"`
	ImageRef  string               `json:"imageRef"`
	Status    string               `json:"status"`
	Error     *string              `json:"error,omitempty"`
	Metadata  base.JSON            `json:"metadata"`
	SentAt    time.Time            `json:"sentAt"`
	CreatedAt time.Time            `json:"createdAt"`
	UpdatedAt time.Time            `json:"updatedAt"`
}

type DiscordConfig struct {
	WebhookID string                         `json:"webhookId"`
	Token     string                         `json:"token"`
	Username  string                         `json:"username,omitempty"`
	AvatarURL string                         `json:"avatarUrl,omitempty"`
	Events    map[NotificationEventType]bool `json:"events,omitempty"`
}

type EmailConfig struct {
	SMTPHost     string                         `json:"smtpHost"`
	SMTPPort     int                            `json:"smtpPort"`
	SMTPUsername string                         `json:"smtpUsername"`
	SMTPPassword string                         `json:"smtpPassword"`
	FromAddress  string                         `json:"fromAddress"`
	ToAddresses  []string                       `json:"toAddresses"`
	TLSMode      EmailTLSMode                   `json:"tlsMode"`
	Events       map[NotificationEventType]bool `json:"events,omitempty"`
}

type TelegramConfig struct {
	BotToken     string                         `json:"botToken"`
	ChatIDs      []string                       `json:"chatIds"`
	Preview      bool                           `json:"preview"`
	Notification bool                           `json:"notification"`
	ParseMode    string                         `json:"parseMode,omitempty"`
	Title        string                         `json:"title,omitempty"`
	Events       map[NotificationEventType]bool `json:"events,omitempty"`
}

type SignalConfig struct {
	Host       string                         `json:"host"`
	Port       int                            `json:"port"`
	User       string                         `json:"user,omitempty"`
	Password   string                         `json:"password,omitempty"`
	Token      string                         `json:"token,omitempty"`
	Source     string                         `json:"source"`
	Recipients []string                       `json:"recipients"`
	DisableTLS bool                           `json:"disableTls"`
	Events     map[NotificationEventType]bool `json:"events,omitempty"`
}

type SlackConfig struct {
	Token    string                         `json:"token"`
	BotName  string                         `json:"botName,omitempty"`
	Icon     string                         `json:"icon,omitempty"`
	Color    string                         `json:"color,omitempty"`
	Title    string                         `json:"title,omitempty"`
	Channel  string                         `json:"channel,omitempty"`
	ThreadTS string                         `json:"threadTs,omitempty"`
	Events   map[NotificationEventType]bool `json:"events,omitempty"`
}

type NtfyConfig struct {
	Host                   string                         `json:"host"`
	Port                   int                            `json:"port"`
	Topic                  string                         `json:"topic"`
	Username               string                         `json:"username,omitempty"`
	Password               string                         `json:"password,omitempty"`
	Priority               string                         `json:"priority,omitempty"`
	Tags                   []string                       `json:"tags,omitempty"`
	Icon                   string                         `json:"icon,omitempty"`
	Cache                  bool                           `json:"cache"`
	Firebase               bool                           `json:"firebase"`
	DisableTLSVerification bool                           `json:"disableTlsVerification"`
	Events                 map[NotificationEventType]bool `json:"events,omitempty"`
}

type PushoverConfig struct {
	Token    string                         `json:"token"`
	User     string                         `json:"user"`
	Devices  []string                       `json:"devices,omitempty"`
	Priority int8                           `json:"priority"`
	Title    string                         `json:"title,omitempty"`
	Events   map[NotificationEventType]bool `json:"events,omitempty"`
}

type GotifyConfig struct {
	Host       string                         `json:"host"`
	Port       int                            `json:"port,omitempty"`
	Token      string                         `json:"token"`
	Path       string                         `json:"path,omitempty"`
	Priority   int                            `json:"priority,omitempty"`
	Title      string                         `json:"title,omitempty"`
	DisableTLS bool                           `json:"disableTls"`
	Events     map[NotificationEventType]bool `json:"events,omitempty"`
}

type GenericConfig struct {
	WebhookURL    string                         `json:"webhookUrl"`
	Method        string                         `json:"method,omitempty"`
	ContentType   string                         `json:"contentType,omitempty"`
	TitleKey      string                         `json:"titleKey,omitempty"`
	MessageKey    string                         `json:"messageKey,omitempty"`
	CustomHeaders map[string]string              `json:"customHeaders,omitempty"`
	DisableTLS    bool                           `json:"disableTls"`
	Events        map[NotificationEventType]bool `json:"events,omitempty"`
}

type AppriseSettings struct {
	ID                 uint      `json:"id"`
	APIURL             string    `json:"apiUrl"`
	Enabled            bool      `json:"enabled"`
	ImageUpdateTag     string    `json:"imageUpdateTag"`
	ContainerUpdateTag string    `json:"containerUpdateTag"`
	CreatedAt          time.Time `json:"createdAt"`
	UpdatedAt          time.Time `json:"updatedAt"`
}
