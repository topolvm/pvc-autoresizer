package config

import (
	"github.com/topolvm/pvc-autoresizer/internal/notifications"
)

// Config represents the configuration for the PVC autoresizer
type Config struct {
	// RescanInterval is the interval between PVC scans
	RescanInterval string `json:"rescanInterval"`

	// Slack notification configuration
	Slack *SlackConfig `json:"slack,omitempty"`
}

// SlackConfig represents Slack notification settings
type SlackConfig struct {
	// WebhookURL is the Slack webhook URL for sending notifications
	WebhookURL string `json:"webhookUrl"`

	// Channel is the Slack channel to send notifications to
	Channel string `json:"channel"`

	// Username is the username that will appear as the sender
	Username string `json:"username"`

	// Enabled determines whether Slack notifications are enabled
	Enabled bool `json:"enabled"`
}

// ToNotifierConfig converts the SlackConfig to notifications.SlackConfig
func (c *SlackConfig) ToNotifierConfig() *notifications.SlackConfig {
	if c == nil {
		return nil
	}
	return &notifications.SlackConfig{
		WebhookURL: c.WebhookURL,
		Channel:    c.Channel,
		Username:   c.Username,
		Enabled:    c.Enabled,
	}
} 