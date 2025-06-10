package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
	"crypto/tls"
)

// SlackConfig holds the configuration for Slack notifications
type SlackConfig struct {
	// WebhookURL is the Slack webhook URL for sending notifications.
	// The webhook URL includes the default channel configuration.
	WebhookURL string

	// Channel is optional and overrides the default channel from webhook URL.
	// If empty, the channel configured in the webhook URL will be used.
	Channel string

	// Username is optional and overrides the default bot name in Slack.
	// If empty, the default bot name configured in Slack will be used.
	Username string

	// Enabled controls whether Slack notifications are active.
	Enabled bool

	// DisableStartupNotification controls whether to send the startup notification.
	// If true, startup notification will be disabled while keeping other notifications enabled.
	DisableStartupNotification bool
}

// SlackMessage represents a Slack message payload
type SlackMessage struct {
	Channel     string       `json:"channel,omitempty"`     // Optional: overrides the default channel
	Username    string       `json:"username,omitempty"`    // Optional: overrides the default bot name
	Text        string       `json:"text,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// Attachment represents a Slack message attachment
type Attachment struct {
	Color      string `json:"color"`
	Title      string `json:"title"`
	Text       string `json:"text"`
	Footer     string `json:"footer"`
	FooterIcon string `json:"footer_icon,omitempty"`
	Timestamp  int64  `json:"ts"`
}

// SlackNotifier handles sending notifications to Slack
type SlackNotifier struct {
	config SlackConfig
}

// NewSlackNotifier creates a new SlackNotifier instance
func NewSlackNotifier(config SlackConfig) *SlackNotifier {
	return &SlackNotifier{
		config: config,
	}
}

// createMessage creates a new Slack message with optional channel and username overrides
func (s *SlackNotifier) createMessage(title, text, color string) SlackMessage {
	message := SlackMessage{
		Attachments: []Attachment{
			{
				Color:      color,
				Title:      title,
				Text:       text,
				Footer:     "PVC Autoresizer | Kubernetes Storage Management",
				FooterIcon: "https://raw.githubusercontent.com/kubernetes/kubernetes/master/logo/logo.png",
				Timestamp:  time.Now().Unix(),
			},
		},
	}

	// Only set channel and username if they are explicitly configured
	if s.config.Channel != "" {
		message.Channel = s.config.Channel
	}
	if s.config.Username != "" {
		message.Username = s.config.Username
	}

	return message
}

// SendResizeNotification sends a notification about PVC resize events
func (s *SlackNotifier) SendResizeNotification(namespace, pvcName string, oldSize, newSize int64, success bool) error {
	if !s.config.Enabled {
		return nil
	}

	color := "#36a64f" // green for success
	status := "✅ Success"
	if !success {
		color = "#dc3545" // red for failure
		status = "⚠️ Limit Reached"
	}

	// Convert bytes to human readable format
	oldSizeStr := formatBytes(oldSize)
	newSizeStr := formatBytes(newSize)

	var title, details string
	if success {
		title = fmt.Sprintf("%s | PVC Resize Operation", status)
		growthPercent := float64(newSize-oldSize) / float64(oldSize) * 100
		details = fmt.Sprintf("*Resize Information* 📊\n"+
			"• Previous Size: `%s`\n"+
			"• New Size: `%s`\n"+
			"• Growth: `%.1f%%`",
			oldSizeStr,
			newSizeStr,
			growthPercent)
	} else {
		title = fmt.Sprintf("%s | PVC Storage Limit", status)
		details = fmt.Sprintf("*Storage Status* 📊\n"+
			"• Current Size: `%s`\n"+
			"• Storage Limit: `%s`\n"+
			"• Status: Cannot resize further - storage limit reached",
			oldSizeStr,
			newSizeStr)
	}

	message := s.createMessage(
		title,
		fmt.Sprintf("*PVC Details*\n"+
			"• Namespace: `%s`\n"+
			"• PVC Name: `%s`\n\n"+
			"%s\n\n"+
			"*Time*\n"+
			"• Detected: <!date^%d^{date_short} at {time}|%s>",
			namespace,
			pvcName,
			details,
			time.Now().Unix(),
			time.Now().Format(time.RFC1123)),
		color,
	)

	return s.sendMessage(message)
}

// SendLimitWarningNotification sends a notification when PVC size is approaching its limit
func (s *SlackNotifier) SendLimitWarningNotification(namespace, pvcName string, currentSize, limitSize int64, warningMessage string) error {
	if !s.config.Enabled {
		return nil
	}

	usagePercent := float64(currentSize) / float64(limitSize) * 100
	currentSizeStr := formatBytes(currentSize)
	limitSizeStr := formatBytes(limitSize)

	message := s.createMessage(
		"⚠️ PVC Storage Alert",
		fmt.Sprintf("*PVC Details*\n"+
			"• Namespace: `%s`\n"+
			"• PVC Name: `%s`\n\n"+
			"*Storage Status* 📊\n"+
			"• Current Usage: `%s` of `%s`\n"+
			"• Usage Percentage: `%.1f%%`\n"+
			"• Alert: %s\n\n"+
			"*Time*\n"+
			"• Detected: <!date^%d^{date_short} at {time}|%s>",
			namespace,
			pvcName,
			currentSizeStr,
			limitSizeStr,
			usagePercent,
			warningMessage,
			time.Now().Unix(),
			time.Now().Format(time.RFC1123)),
		"#ffc107", // warning yellow
	)

	return s.sendMessage(message)
}

// IsStartupNotificationDisabled returns true if startup notifications are disabled
func (s *SlackNotifier) IsStartupNotificationDisabled() bool {
	return s.config.DisableStartupNotification
}

// SendStartupNotification sends a notification when the application starts
func (s *SlackNotifier) SendStartupNotification(message string) error {
	if !s.config.Enabled || s.config.DisableStartupNotification {
		return nil
	}

	slackMessage := s.createMessage(
		"🚀 PVC Autoresizer Started",
		fmt.Sprintf("*System Status*\n"+
			"• Status: `Online & Monitoring`\n"+
			"• Time: <!date^%d^{date_short} at {time}|%s>\n\n"+
			"*Configuration Details* ⚙️\n%s",
			time.Now().Unix(),
			time.Now().Format(time.RFC1123),
			message),
		"#0066FF", // blue for startup/info
	)

	return s.sendMessage(slackMessage)
}

// formatBytes converts bytes to human readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// sendMessage handles the actual HTTP request to Slack
func (s *SlackNotifier) sendMessage(message SlackMessage) error {
	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal Slack message: %w", err)
	}

	// Create a custom HTTP client that skips TLS verification for Slack webhooks
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	resp, err := client.Post(s.config.WebhookURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to send Slack notification: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send Slack notification, got status code: %d", resp.StatusCode)
	}

	return nil
} 