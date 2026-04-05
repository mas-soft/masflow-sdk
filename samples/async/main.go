// Async example -- demonstrates async activities with callback patterns.
//
// Async activities start long-running work (e.g., creating a Jira ticket,
// requesting human approval) and return immediately. The workflow pauses
// until an external system signals completion using the callback info.
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
	"time"

	sdk "github.com/mas-soft/masflow/sdk"
)

// ── Types ────────────────────────────────────────────────────────────────

// ApprovalRequest is the input for requesting a human approval.
type ApprovalRequest struct {
	RequestID   string `json:"request_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Approver    string `json:"approver"`
	Urgency     string `json:"urgency,omitempty"`
}

// ApprovalResult is the initial response (before approval completes).
type ApprovalResult struct {
	TicketID  string `json:"ticket_id"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// ExternalJobRequest is the input for starting an external job.
type ExternalJobRequest struct {
	JobType    string            `json:"job_type"`
	Parameters map[string]string `json:"parameters"`
	Priority   int               `json:"priority,omitempty"`
}

// ExternalJobResult is the initial acknowledgment.
type ExternalJobResult struct {
	JobID     string `json:"job_id"`
	Status    string `json:"status"`
	QueuedAt  string `json:"queued_at"`
}

// ── Handlers ─────────────────────────────────────────────────────────────

// RequestApproval creates an approval ticket and stores the callback info
// so the approval system can signal the workflow when a decision is made.
func RequestApproval(_ context.Context, in ApprovalRequest, async *sdk.AsyncCallbackInfo) (ApprovalResult, error) {
	if in.Title == "" {
		return ApprovalResult{}, fmt.Errorf("title is required")
	}
	if in.Approver == "" {
		return ApprovalResult{}, fmt.Errorf("approver is required")
	}

	ticketID := fmt.Sprintf("APR-%d", time.Now().UnixNano()%100000)

	slog.Info("Approval requested",
		"ticket_id", ticketID,
		"approver", in.Approver,
		"workflow_id", async.WorkflowID,
		"run_id", async.RunID,
		"callback_signal", async.CallbackSignal,
		"callback_timeout", async.CallbackTimeout,
	)

	return ApprovalResult{
		TicketID:  ticketID,
		Status:    "pending_approval",
		CreatedAt: time.Now().Format(time.RFC3339),
	}, nil
}

// StartExternalJob kicks off a long-running job in an external system
// and stores callback info for completion notification.
func StartExternalJob(_ context.Context, in ExternalJobRequest, async *sdk.AsyncCallbackInfo) (ExternalJobResult, error) {
	if in.JobType == "" {
		return ExternalJobResult{}, fmt.Errorf("job_type is required")
	}

	jobID := fmt.Sprintf("JOB-%d", time.Now().UnixNano()%100000)

	slog.Info("External job started",
		"job_id", jobID,
		"job_type", in.JobType,
		"callback_signal", async.CallbackSignal,
	)

	return ExternalJobResult{
		JobID:    jobID,
		Status:   "queued",
		QueuedAt: time.Now().Format(time.RFC3339),
	}, nil
}

// NotifyComplete is a simple sync activity that runs after the async step.
type NotifyInput struct {
	Message string `json:"message"`
	Channel string `json:"channel"`
}

type NotifyOutput struct {
	Delivered bool   `json:"delivered"`
	SentAt    string `json:"sent_at"`
}

func NotifyComplete(_ context.Context, in NotifyInput) (NotifyOutput, error) {
	slog.Info("Notification sent", "channel", in.Channel, "message", in.Message)
	return NotifyOutput{
		Delivered: true,
		SentAt:    time.Now().Format(time.RFC3339),
	}, nil
}

// ── Main ─────────────────────────────────────────────────────────────────

func main() {
	platformURL := flag.String("platform", envOr("MASFLOW_PLATFORM_URL", ""), "Masflow platform URL (required)")
	flag.Parse()

	if *platformURL == "" {
		log.Fatal("--platform (or MASFLOW_PLATFORM_URL) is required")
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	mod := sdk.NewModule("approvals",
		sdk.WithModuleDescription("Async approval and external job activities"),
		sdk.WithModuleVersion("1.0.0"),
		sdk.WithModuleIcon("clock"),
		sdk.WithModuleTaskQueue("approvals-task-queue"),
		sdk.WithModuleAuthor("masflow-samples"),
		sdk.WithModuleCategory("approvals"),
		sdk.WithModuleTags("async", "approval", "human-in-the-loop", "external-job"),
	)

	sdk.RegisterAsync(mod, "requestApproval", RequestApproval,
		sdk.WithDescription("Create an approval request and wait for human decision"),
		sdk.WithIcon("user-check"),
		sdk.WithCategory("approval"),
		sdk.WithTags("approval", "human", "async"),
	)

	sdk.RegisterAsync(mod, "startExternalJob", StartExternalJob,
		sdk.WithDescription("Start a long-running external job and wait for completion"),
		sdk.WithIcon("cpu"),
		sdk.WithCategory("jobs"),
		sdk.WithTags("external", "job", "async"),
	)

	sdk.Register(mod, "notifyComplete", NotifyComplete,
		sdk.WithDescription("Send a completion notification"),
		sdk.WithIcon("bell"),
		sdk.WithCategory("notifications"),
	)

	runner, err := sdk.NewRunner(mod,
		sdk.WithPlatformURL(*platformURL),
		sdk.WithLogger(logger),
	)
	if err != nil {
		log.Fatal(err)
	}

	logger.Info("Starting approvals module",
		"activities", len(mod.Activities()),
		"task_queue", mod.TaskQueue,
	)

	if err := runner.Run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
