// Advanced example -- a full notifications module with multiple activities.
//
// Demonstrates multiple activity types (sync, void), rich metadata,
// input validation, structured error handling, and workflow execution.
// The server provides Temporal connection details during registration.
//
//	# Worker only:
//	go run . --server=http://localhost:9999
//
//	# Worker + execute a workflow:
//	go run . --server=http://localhost:9999 --execute
//
//	# Execute workflow from YAML file:
//	go run . --server=http://localhost:9999 --execute --yaml workflows/order-notifications.yaml
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	sdk "github.com/mas-soft/masflow-sdk"
)

func main() {
	serverURL := flag.String("server", envOr("MASFLOW_SERVER_URL", ""), "Masflow server URL (required)")
	execute := flag.Bool("execute", true, "Execute a sample notification workflow after starting the worker")
	yamlFile := flag.String("yaml", "workflows/order-notifications.yaml", "Path to workflow YAML file (used with --execute)")

	flag.Parse()

	if *serverURL == "" {
		log.Fatal("--server (or MASFLOW_SERVER_URL) is required")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// ── Module ───────────────────────────────────────────────────────────

	mod := sdk.NewModule("notifications", "1.0.0",
		sdk.WithModuleDescription("Email, SMS, Slack, and webhook notification activities"),
		sdk.WithModuleIcon("bell"),
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
		sdk.WithServerURL(*serverURL),
		sdk.WithLogger(logger),
	)
	if err != nil {
		log.Fatalf("Failed to create runner: %v", err)
	}

	logger.Info("Starting notifications module",
		"server", *serverURL,
		"activities", len(mod.Activities()),
	)

	if *execute {
		if err := runner.Start(context.Background()); err != nil {
			log.Fatalf("Failed to start runner: %v", err)
		}

		time.Sleep(2 * time.Second)
		executeNotificationWorkflow(runner.Workflows(), logger, *yamlFile)

		logger.Info("Worker still running. Press Ctrl+C to stop.")
		select {}
	}

	if err := runner.Run(context.Background()); err != nil {
		log.Fatalf("Runner error: %v", err)
	}
}

// executeNotificationWorkflow executes a multi-channel notification workflow
// and monitors it to completion.
func executeNotificationWorkflow(wc *sdk.WorkflowClient, logger *slog.Logger, yamlFile string) {
	if wc == nil {
		logger.Error("WorkflowClient not available (WithServerURL not set)")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	data, err := os.ReadFile(yamlFile)
	if err != nil {
		logger.Error("Failed to read workflow file", "file", yamlFile, "error", err)
		return
	}
	yaml := string(data)
	logger.Info("Loaded workflow from file", "file", yamlFile)

	// Execute
	workflowID := fmt.Sprintf("notifications-%d", time.Now().Unix())
	logger.Info("Executing notification workflow", "workflow_id", workflowID)

	result, err := wc.ExecuteYAML(ctx, yaml, &sdk.ExecuteSourceOptions{
		WorkflowID: workflowID,
		Variables: map[string]any{
			"customer_email": "user@example.com",
			"customer_phone": "+1234567890",
			"order_id":       "ORD-42",
			"order_total":    99.99,
		},
	})
	if err != nil {
		logger.Error("Failed to execute workflow", "error", err)
		return
	}

	logger.Info("Workflow started",
		"workflow_id", result.WorkflowID,
		"status", result.Status,
	)

	// Poll for completion
	pollWorkflowStatus(ctx, wc, logger, result.WorkflowID)

	// Show monitoring data
	mon, err := wc.Monitor(ctx, result.WorkflowID, "")
	if err != nil {
		logger.Warn("Could not fetch monitor data", "error", err)
		return
	}

	logger.Info("Workflow monitor",
		"status", mon.Status,
		"progress", fmt.Sprintf("%d/%d steps (%.0f%%)",
			mon.Progress.CurrentStep, mon.Progress.TotalSteps, mon.Progress.Percentage),
		"completed", mon.StepCounts.Completed,
		"failed", mon.StepCounts.Failed,
	)
	for _, s := range mon.Steps {
		logger.Info("  step",
			"name", s.Name,
			"type", s.StepType,
			"status", s.Status,
			"duration_ms", s.DurationMs,
		)
	}
}

// pollWorkflowStatus polls a workflow until it completes, fails, or times out.
func pollWorkflowStatus(ctx context.Context, wc *sdk.WorkflowClient, logger *slog.Logger, workflowID string) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Warn("Timed out waiting for workflow", "workflow_id", workflowID)
			return
		case <-ticker.C:
			status, err := wc.GetStatus(ctx, workflowID)
			if err != nil {
				logger.Error("Failed to get status", "error", err)
				continue
			}

			logger.Info("Workflow status",
				"workflow_id", status.WorkflowID,
				"status", status.Status,
			)

			switch status.Status {
			case sdk.WorkflowStatusCompleted:
				logger.Info("Workflow completed successfully!")
				return
			case sdk.WorkflowStatusFailed, sdk.WorkflowStatusCancelled:
				logger.Error("Workflow finished with error", "status", status.Status)
				return
			}
		}
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
