// Basic example -- the simplest possible Masflow module.
//
// This registers a single "greet" activity and runs the worker.
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
	flag.Parse()

	if *platformURL == "" {
		log.Fatal("--platform (or MASFLOW_PLATFORM_URL) is required")
	}

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
		sdk.WithLogger(slog.New(slog.NewTextHandler(os.Stdout, nil))),
	)
	if err != nil {
		log.Fatal(err)
	}
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
