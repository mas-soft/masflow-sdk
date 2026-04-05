package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// ── Email ────────────────────────────────────────────────────────────────

type SendEmailInput struct {
	To      string            `json:"to"`
	Cc      []string          `json:"cc,omitempty"`
	Subject string            `json:"subject"`
	Body    string            `json:"body"`
	IsHTML  bool              `json:"is_html,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type SendEmailOutput struct {
	MessageID  string `json:"message_id"`
	Status     string `json:"status"`
	SentAt     string `json:"sent_at"`
	Recipients int    `json:"recipients"`
}

func SendEmail(_ context.Context, in SendEmailInput) (SendEmailOutput, error) {
	if in.To == "" {
		return SendEmailOutput{}, fmt.Errorf("recipient (to) is required")
	}
	if in.Subject == "" {
		return SendEmailOutput{}, fmt.Errorf("subject is required")
	}

	recipients := 1 + len(in.Cc)
	slog.Info("Sending email", "to", in.To, "subject", in.Subject, "recipients", recipients)

	return SendEmailOutput{
		MessageID:  fmt.Sprintf("msg-%d", time.Now().UnixNano()),
		Status:     "sent",
		SentAt:     time.Now().Format(time.RFC3339),
		Recipients: recipients,
	}, nil
}

// ── SMS ──────────────────────────────────────────────────────────────────

type SendSMSInput struct {
	PhoneNumber string `json:"phone_number"`
	Message     string `json:"message"`
	From        string `json:"from,omitempty"`
}

type SendSMSOutput struct {
	MessageSID string `json:"message_sid"`
	Status     string `json:"status"`
	Segments   int    `json:"segments"`
}

func SendSMS(_ context.Context, in SendSMSInput) (SendSMSOutput, error) {
	if in.PhoneNumber == "" {
		return SendSMSOutput{}, fmt.Errorf("phone_number is required")
	}
	if in.Message == "" {
		return SendSMSOutput{}, fmt.Errorf("message is required")
	}

	segments := (len(in.Message) / 160) + 1
	slog.Info("Sending SMS", "to", in.PhoneNumber, "segments", segments)

	return SendSMSOutput{
		MessageSID: fmt.Sprintf("SM%d", time.Now().UnixNano()),
		Status:     "queued",
		Segments:   segments,
	}, nil
}

// ── Slack ────────────────────────────────────────────────────────────────

type SendSlackInput struct {
	Channel  string `json:"channel"`
	Message  string `json:"message"`
	Username string `json:"username,omitempty"`
	IconURL  string `json:"icon_url,omitempty"`
}

type SendSlackOutput struct {
	Channel   string `json:"channel"`
	Timestamp string `json:"timestamp"`
	OK        bool   `json:"ok"`
}

func SendSlack(_ context.Context, in SendSlackInput) (SendSlackOutput, error) {
	if in.Channel == "" {
		return SendSlackOutput{}, fmt.Errorf("channel is required")
	}
	if in.Message == "" {
		return SendSlackOutput{}, fmt.Errorf("message is required")
	}
	if !strings.HasPrefix(in.Channel, "#") && !strings.HasPrefix(in.Channel, "@") {
		in.Channel = "#" + in.Channel
	}

	slog.Info("Posting to Slack", "channel", in.Channel)

	return SendSlackOutput{
		Channel:   in.Channel,
		Timestamp: fmt.Sprintf("%d.%06d", time.Now().Unix(), time.Now().Nanosecond()/1000),
		OK:        true,
	}, nil
}

// ── Webhook ──────────────────────────────────────────────────────────────

type SendWebhookInput struct {
	URL     string            `json:"url"`
	Method  string            `json:"method,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
}

type SendWebhookOutput struct {
	StatusCode int    `json:"status_code"`
	Duration   string `json:"duration"`
}

func SendWebhook(_ context.Context, in SendWebhookInput) (SendWebhookOutput, error) {
	if in.URL == "" {
		return SendWebhookOutput{}, fmt.Errorf("url is required")
	}
	if in.Method == "" {
		in.Method = "POST"
	}

	start := time.Now()
	slog.Info("Sending webhook", "url", in.URL, "method", in.Method)

	return SendWebhookOutput{
		StatusCode: 200,
		Duration:   time.Since(start).String(),
	}, nil
}

// ── Log (void) ───────────────────────────────────────────────────────────

type LogNotificationInput struct {
	Message  string            `json:"message"`
	Level    string            `json:"level"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

func LogNotification(_ context.Context, in LogNotificationInput) error {
	if in.Level == "" {
		in.Level = "info"
	}
	slog.Info("Notification log", "level", in.Level, "message", in.Message)
	return nil
}
