// Basic example -- the simplest possible Masflow module.
//
// This registers a single "greet" activity, runs the worker, and optionally
// executes a workflow to demonstrate the full round-trip.
//
//	# Worker only:
//	go run . --platform=http://localhost:9999
//
//	# Worker + execute a workflow:
//	go run . --platform=http://localhost:9999 --execute
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	sdk "github.com/mas-soft/masflow/sdk"
)

// ── Types ────────────────────────────────────────────────────────────────

// GreetInput is the activity input.
type GreetInput struct {
	Name string `json:"name"`
}

// GreetOutput is the activity output.
type GreetOutput struct {
	Message string `json:"message"`
	SentAt  string `json:"sent_at"`
}

// ── Handler ──────────────────────────────────────────────────────────────

// Greet returns a personalized greeting.
func Greet(_ context.Context, in GreetInput) (GreetOutput, error) {
	if in.Name == "" {
		return GreetOutput{}, fmt.Errorf("name is required")
	}
	return GreetOutput{
		Message: fmt.Sprintf("Hello, %s!", in.Name),
		SentAt:  time.Now().Format(time.RFC3339),
	}, nil
}

// ── Main ─────────────────────────────────────────────────────────────────

func main() {
	platformURL := flag.String("platform", envOr("MASFLOW_PLATFORM_URL", ""), "Masflow platform URL (required)")
	execute := flag.Bool("execute", false, "Execute a sample greeting workflow after starting the worker")
	name := flag.String("name", "World", "Name to greet (used with --execute)")
	flag.Parse()

	if *platformURL == "" {
		log.Fatal("--platform (or MASFLOW_PLATFORM_URL) is required")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// 1. Create module
	mod := sdk.NewModule("greeter",
		sdk.WithModuleDescription("A simple greeting module"),
		sdk.WithModuleVersion("1.0.0"),
		sdk.WithModuleTaskQueue("greeter-task-queue"),
		sdk.WithModuleAuthor("masflow-samples"),
		sdk.WithModuleCategory("demo"),
		sdk.WithModuleTags("greeting", "demo", "basic"),
	)

	// 2. Register activity
	sdk.Register(mod, "greet", Greet,
		sdk.WithDescription("Return a personalized greeting"),
		sdk.WithIcon("hand-wave"),
		sdk.WithCategory("demo"),
	)

	// 3. Run — platform provides Temporal connection details
	runner, err := sdk.NewRunner(mod,
		sdk.WithPlatformURL(*platformURL),
		sdk.WithWorkflowURL(*platformURL),
		sdk.WithLogger(logger),
	)
	if err != nil {
		log.Fatal(err)
	}

	// If --execute, start worker in background, run a workflow, then keep running
	if *execute {
		if err := runner.Start(context.Background()); err != nil {
			log.Fatalf("Failed to start runner: %v", err)
		}

		// Give the worker a moment to be ready
		time.Sleep(2 * time.Second)

		executeGreetingWorkflow(runner.Workflows(), logger, *name)

		// Block until signal
		logger.Info("Worker still running. Press Ctrl+C to stop.")
		select {}
	}

	// Normal mode: just run the worker and block
	if err := runner.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

// executeGreetingWorkflow demonstrates executing a workflow using the SDK's
// WorkflowClient, then polling for its status.
func executeGreetingWorkflow(wc *sdk.WorkflowClient, logger *slog.Logger, name string) {
	if wc == nil {
		logger.Error("WorkflowClient not available (WithWorkflowURL not set)")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// ── Execute workflow from inline YAML ────────────────────────────
	yaml := fmt.Sprintf(`name: greeting-workflow
description: Greet a user
variables:
  name: "%s"
steps:
  - name: say-hello
    activity:
      type: greet
      args:
        name: "${name}"
      ref: greeting
`, name)

	workflowID := fmt.Sprintf("greeting-%d", time.Now().Unix())
	logger.Info("Executing greeting workflow", "workflow_id", workflowID, "name", name)

	result, err := wc.ExecuteYAML(ctx, yaml, &sdk.ExecuteSourceOptions{
		WorkflowID: workflowID,
	})
	if err != nil {
		logger.Error("Failed to execute workflow", "error", err)
		return
	}

	logger.Info("Workflow started",
		"workflow_id", result.WorkflowID,
		"status", result.Status,
	)

	// ── Poll for completion ──────────────────────────────────────────
	pollWorkflowStatus(ctx, wc, logger, result.WorkflowID)
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
				if len(status.Trace) > 0 {
					for _, t := range status.Trace {
						logger.Info("  trace",
							"step", t.StepType,
							"status", t.Status,
							"details", t.Details,
						)
					}
				}
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
