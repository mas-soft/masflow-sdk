// Advanced example -- a full notifications module with multiple activities.
//
// Demonstrates multiple activity types (sync, void), rich metadata,
// input validation, structured error handling, and environment-based config.
// The platform provides Temporal connection details during registration.
//
//	go run . --platform=http://localhost:10000
package main

import (
	"context"
	"flag"
	"log"
	"log/slog"
	"os"

	sdk "github.com/mas-soft/masflow/sdk"
)

func main() {
	platformURL := flag.String("platform", envOr("MASFLOW_PLATFORM_URL", ""), "Masflow platform URL (required)")
	flag.Parse()

	if *platformURL == "" {
		log.Fatal("--platform (or MASFLOW_PLATFORM_URL) is required")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// ── Module ───────────────────────────────────────────────────────────

	mod := sdk.NewModule("notifications",
		sdk.WithModuleDescription("Email, SMS, Slack, and webhook notification activities"),
		sdk.WithModuleVersion("1.0.0"),
		sdk.WithModuleIcon("bell"),
		sdk.WithModuleTaskQueue("notifications-task-queue"),
		sdk.WithModuleAuthor("masflow-samples"),
		sdk.WithModuleCategory("notifications"),
		sdk.WithModuleTags("email", "sms", "slack", "webhook", "alerts"),
	)

	// ── Activities ───────────────────────────────────────────────────────

	sdk.Register(mod, "sendEmail", SendEmail,
		sdk.WithDescription("Send an email via SMTP or transactional email service"),
		sdk.WithIcon("mail"),
		sdk.WithCategory("email"),
		sdk.WithTags("email", "notification", "smtp"),
		sdk.WithDocumentationURL("https://docs.example.com/activities/send-email"),
	)

	sdk.Register(mod, "sendSMS", SendSMS,
		sdk.WithDescription("Send an SMS text message via Twilio or compatible provider"),
		sdk.WithIcon("smartphone"),
		sdk.WithCategory("sms"),
		sdk.WithTags("sms", "notification", "twilio"),
	)

	sdk.Register(mod, "sendSlack", SendSlack,
		sdk.WithDescription("Post a message to a Slack channel"),
		sdk.WithIcon("message-square"),
		sdk.WithCategory("chat"),
		sdk.WithTags("slack", "notification", "chat"),
	)

	sdk.Register(mod, "sendWebhook", SendWebhook,
		sdk.WithDescription("Send an HTTP webhook notification to an external endpoint"),
		sdk.WithIcon("webhook"),
		sdk.WithCategory("webhook"),
		sdk.WithTags("webhook", "http", "notification"),
	)

	sdk.RegisterVoid(mod, "logNotification", LogNotification,
		sdk.WithDescription("Write a notification event to the audit log"),
		sdk.WithIcon("file-text"),
		sdk.WithCategory("logging"),
		sdk.WithTags("log", "audit"),
	)

	// ── Runner ───────────────────────────────────────────────────────────

	runner, err := sdk.NewRunner(mod,
		sdk.WithPlatformURL(*platformURL),
		sdk.WithLogger(logger),
	)
	if err != nil {
		log.Fatalf("Failed to create runner: %v", err)
	}

	logger.Info("Starting notifications module",
		"platform", *platformURL,
		"task_queue", mod.TaskQueue,
		"activities", len(mod.Activities()),
	)

	if err := runner.Run(context.Background()); err != nil {
		log.Fatalf("Runner error: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
