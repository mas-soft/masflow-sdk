// Multi-module example -- run multiple independent modules in one process.
//
// This demonstrates running two separate modules (email + analytics), each
// with its own task queue and runner, in a single Go process using errgroup.
// The platform provides Temporal connection details during registration.
//
//	go run . --platform=http://localhost:10000
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	sdk "github.com/mas-soft/masflow/sdk"
	"golang.org/x/sync/errgroup"
)

// ── Email Module Activities ──────────────────────────────────────────────

type EmailInput struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

type EmailOutput struct {
	MessageID string `json:"message_id"`
	SentAt    string `json:"sent_at"`
}

func SendEmail(_ context.Context, in EmailInput) (EmailOutput, error) {
	slog.Info("[email] Sending", "to", in.To, "subject", in.Subject)
	return EmailOutput{
		MessageID: fmt.Sprintf("eml-%d", time.Now().UnixNano()),
		SentAt:    time.Now().Format(time.RFC3339),
	}, nil
}

type TemplateInput struct {
	TemplateName string            `json:"template_name"`
	Variables    map[string]string `json:"variables"`
}

type TemplateOutput struct {
	RenderedHTML string `json:"rendered_html"`
}

func RenderTemplate(_ context.Context, in TemplateInput) (TemplateOutput, error) {
	slog.Info("[email] Rendering template", "template", in.TemplateName)
	html := fmt.Sprintf("<h1>%s</h1><p>Rendered with %d variables</p>",
		in.TemplateName, len(in.Variables))
	return TemplateOutput{RenderedHTML: html}, nil
}

// ── Analytics Module Activities ──────────────────────────────────────────

type TrackEventInput struct {
	EventName  string            `json:"event_name"`
	UserID     string            `json:"user_id"`
	Properties map[string]string `json:"properties,omitempty"`
}

type TrackEventOutput struct {
	EventID   string `json:"event_id"`
	Timestamp string `json:"timestamp"`
}

func TrackEvent(_ context.Context, in TrackEventInput) (TrackEventOutput, error) {
	slog.Info("[analytics] Tracking event", "event", in.EventName, "user", in.UserID)
	return TrackEventOutput{
		EventID:   fmt.Sprintf("evt-%d", time.Now().UnixNano()),
		Timestamp: time.Now().Format(time.RFC3339),
	}, nil
}

type AggregateInput struct {
	Metric    string `json:"metric"`
	Dimension string `json:"dimension"`
	Period    string `json:"period"`
}

type AggregateOutput struct {
	Value float64 `json:"value"`
	Count int     `json:"count"`
}

func Aggregate(_ context.Context, in AggregateInput) (AggregateOutput, error) {
	slog.Info("[analytics] Aggregating", "metric", in.Metric, "dimension", in.Dimension)
	return AggregateOutput{Value: 42.5, Count: 100}, nil
}

// ── Main ─────────────────────────────────────────────────────────────────

func main() {
	platformURL := flag.String("platform", envOr("MASFLOW_PLATFORM_URL", ""), "Masflow platform URL (required)")
	flag.Parse()

	if *platformURL == "" {
		log.Fatal("--platform (or MASFLOW_PLATFORM_URL) is required")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// ── Email Module ─────────────────────────────────────────────────

	emailMod := sdk.NewModule("email",
		sdk.WithModuleDescription("Email sending and template rendering"),
		sdk.WithModuleVersion("1.0.0"),
		sdk.WithModuleIcon("mail"),
		sdk.WithModuleTaskQueue("email-task-queue"),
		sdk.WithModuleCategory("email"),
	)

	sdk.Register(emailMod, "sendEmail", SendEmail,
		sdk.WithDescription("Send an email"),
		sdk.WithCategory("email"),
	)

	sdk.Register(emailMod, "renderTemplate", RenderTemplate,
		sdk.WithDescription("Render an email template"),
		sdk.WithCategory("email"),
	)

	// ── Analytics Module ─────────────────────────────────────────────

	analyticsMod := sdk.NewModule("analytics",
		sdk.WithModuleDescription("Event tracking and metric aggregation"),
		sdk.WithModuleVersion("1.0.0"),
		sdk.WithModuleIcon("bar-chart"),
		sdk.WithModuleTaskQueue("analytics-task-queue"),
		sdk.WithModuleCategory("analytics"),
	)

	sdk.Register(analyticsMod, "trackEvent", TrackEvent,
		sdk.WithDescription("Track a user event"),
		sdk.WithCategory("tracking"),
	)

	sdk.Register(analyticsMod, "aggregate", Aggregate,
		sdk.WithDescription("Aggregate metrics"),
		sdk.WithCategory("metrics"),
	)

	// ── Run both modules concurrently ────────────────────────────────

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	g, gctx := errgroup.WithContext(ctx)

	for _, mod := range []*sdk.Module{emailMod, analyticsMod} {
		mod := mod
		g.Go(func() error {
			runner, err := sdk.NewRunner(mod,
				sdk.WithPlatformURL(*platformURL),
				sdk.WithLogger(logger.With("module", mod.Name)),
			)
			if err != nil {
				return fmt.Errorf("create runner for %s: %w", mod.Name, err)
			}

			logger.Info("Starting module",
				"module", mod.Name,
				"task_queue", mod.TaskQueue,
				"activities", len(mod.Activities()),
			)

			return runner.Run(gctx)
		})
	}

	if err := g.Wait(); err != nil {
		log.Fatalf("Runner error: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
