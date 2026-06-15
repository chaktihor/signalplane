package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"github.com/chaktihor/signalplane/internal/store"
)

type Dispatcher struct {
	smtpAddr string
	from     string
	client   *http.Client
}

type Options struct {
	SMTPAddr string
	From     string
	Timeout  time.Duration
}

func NewDispatcher(options Options) *Dispatcher {
	timeout := options.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	return &Dispatcher{
		smtpAddr: options.SMTPAddr,
		from:     firstNonEmpty(options.From, "signalplane@localhost"),
		client:   &http.Client{Timeout: timeout},
	}
}

func (dispatcher *Dispatcher) NotifyAlert(alert store.Alert, channels []store.NotificationChannel) {
	for _, channel := range channels {
		ctx, cancel := context.WithTimeout(context.Background(), dispatcher.client.Timeout)
		_ = dispatcher.send(ctx, channel, alertPayload(alert, false))
		cancel()
	}
}

func (dispatcher *Dispatcher) TestNotification(ctx context.Context, channel store.NotificationChannel) error {
	return dispatcher.send(ctx, channel, map[string]any{
		"type":      "test",
		"title":     "SignalPlane test notification",
		"severity":  "info",
		"message":   "Notification channel is configured correctly.",
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
	})
}

func (dispatcher *Dispatcher) send(ctx context.Context, channel store.NotificationChannel, payload map[string]any) error {
	if !channel.Enabled {
		return nil
	}
	switch channel.Type {
	case "email":
		return dispatcher.sendEmail(channel, payload)
	case "webhook", "slack_webhook":
		return dispatcher.sendWebhook(ctx, channel, payload)
	default:
		return errors.New("unsupported notification channel type: " + channel.Type)
	}
}

func (dispatcher *Dispatcher) sendEmail(channel store.NotificationChannel, payload map[string]any) error {
	if dispatcher.smtpAddr == "" {
		return errors.New("SMTP address is not configured")
	}
	if channel.Target == "" {
		return errors.New("email channel target is required")
	}
	subject := fmt.Sprintf("[SignalPlane] %s", value(payload, "title"))
	body := fmt.Sprintf("%s\n\nSeverity: %s\nMessage: %s\nTimestamp: %s\n",
		value(payload, "title"), value(payload, "severity"), value(payload, "message"), value(payload, "timestamp"))
	message := []byte("From: " + dispatcher.from + "\r\n" +
		"To: " + channel.Target + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"Content-Type: text/plain; charset=utf-8\r\n\r\n" +
		body)
	return smtp.SendMail(dispatcher.smtpAddr, nil, dispatcher.from, []string{channel.Target}, message)
}

func (dispatcher *Dispatcher) sendWebhook(ctx context.Context, channel store.NotificationChannel, payload map[string]any) error {
	if channel.Target == "" {
		return errors.New("webhook target is required")
	}
	body := payload
	if channel.Type == "slack_webhook" {
		body = map[string]any{
			"text": fmt.Sprintf("*%s* [%s]\n%s", value(payload, "title"), value(payload, "severity"), value(payload, "message")),
		}
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, channel.Target, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range channel.Config {
		if strings.HasPrefix(strings.ToLower(key), "header.") {
			req.Header.Set(strings.TrimPrefix(key, "header."), value)
		}
	}
	resp, err := dispatcher.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned %s", resp.Status)
	}
	return nil
}

func alertPayload(alert store.Alert, test bool) map[string]any {
	kind := "alert"
	if test {
		kind = "test"
	}
	return map[string]any{
		"type":      kind,
		"id":        alert.ID,
		"title":     alert.Title,
		"severity":  alert.Severity,
		"status":    alert.Status,
		"source":    alert.Source,
		"entityId":  alert.EntityID,
		"message":   alert.Message,
		"timestamp": alert.Timestamp,
		"labels":    alert.Labels,
	}
}

func value(payload map[string]any, key string) string {
	if payload[key] == nil {
		return ""
	}
	return fmt.Sprint(payload[key])
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
