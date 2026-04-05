# Masflow Go SDK

**Build and register third-party activity modules for the Masflow workflow engine.**

The Masflow SDK is a standalone Go module that lets you create custom activity modules and plug them into the Masflow platform without importing any server-internal packages. It provides:

- Type-safe activity registration with Go generics
- Automatic JSON Schema generation from Go structs
- Built-in Temporal worker lifecycle management
- Automatic platform registration via Connect/gRPC
- Graceful shutdown with signal handling
- Workflow execution, monitoring, and lifecycle management

---

## Table of Contents

1. [Installation](#installation)
2. [Quick Start](#quick-start)
3. [Concepts](#concepts)
4. [Defining a Module](#defining-a-module)
5. [Registering Activities](#registering-activities)
6. [Running the Module](#running-the-module)
7. [Advanced Usage](#advanced-usage)
8. [Architecture](#architecture)
9. [API Reference](#api-reference)
10. [Workflow Client](#workflow-client)
11. [Workflow YAML Integration](#workflow-yaml-integration)
12. [Configuration Reference](#configuration-reference)
13. [Troubleshooting](#troubleshooting)

---

## Installation

```bash
go get github.com/mas-soft/masflow/sdk@latest
```

**Requirements:**
- Go 1.25+
- A running Masflow platform instance (provides Temporal connection details during module registration)

---

## Quick Start

A complete module in under 40 lines:

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    sdk "github.com/mas-soft/masflow/sdk"
)

// 1. Define input/output types as plain Go structs with json tags
type GreetInput struct {
    Name string `json:"name"`
}

type GreetOutput struct {
    Message string `json:"message"`
    SentAt  string `json:"sent_at"`
}

// 2. Implement the activity handler
func Greet(_ context.Context, input GreetInput) (GreetOutput, error) {
    return GreetOutput{
        Message: fmt.Sprintf("Hello, %s!", input.Name),
        SentAt:  time.Now().Format(time.RFC3339),
    }, nil
}

func main() {
    // 3. Create a module and register activities
    mod := sdk.NewModule("greeter",
        sdk.WithModuleDescription("A friendly greeting module"),
        sdk.WithModuleVersion("1.0.0"),
        sdk.WithModuleTaskQueue("greeter-task-queue"),
    )

    sdk.Register(mod, "greet", Greet,
        sdk.WithDescription("Send a personalized greeting"),
        sdk.WithCategory("demo"),
    )

    // 4. Run - registers with platform, receives Temporal config, starts worker
    runner, err := sdk.NewRunner(mod,
        sdk.WithPlatformURL("http://localhost:9999"),
    )
    if err != nil {
        log.Fatal(err)
    }
    if err := runner.Run(context.Background()); err != nil {
        log.Fatal(err)
    }
}
```

That's it. The platform provides Temporal connection details during registration. The SDK handles worker lifecycle, JSON Schema generation, platform registration, and graceful shutdown.

---

## Concepts

### How Masflow Modules Work

```
                                   Masflow Platform
                                  +-----------------+
                                  |  Workflow Engine |
                                  |  (Temporal DSL)  |
                                  +--------+--------+
                                           |
                          dispatches activities via Temporal task queues
                                           |
              +----------------------------+----------------------------+
              |                            |                            |
    +---------v---------+       +----------v----------+      +----------v----------+
    | Built-in Module   |       | Your Module (SDK)   |      | Another Module      |
    | bpm-task-list     |       | my-custom-queue     |      | payments-queue      |
    | httpCall, dbQuery |       | sendEmail, sendSMS  |      | chargeCard, refund  |
    +-------------------+       +---------------------+      +---------------------+
```

1. **Module** -- A named group of related activities from one provider (e.g., "notifications", "payments")
2. **Activity** -- A single unit of work with typed input/output and a handler function
3. **Runner** -- Manages the full lifecycle: Temporal worker + platform registration
4. **Task Queue** -- Temporal routes activity execution to the correct worker via task queues

When a workflow step references your activity (e.g., `type: sendEmail`), Temporal dispatches it to the task queue where your module's worker is listening. Your handler executes and returns the result back to the workflow.

### Module Registration Flow

```
Your Module                    Masflow Platform              Temporal Server
    |                               |                            |
    |------ Register Module ------->|                            |
    |       (name, activities,      |                            |
    |        schemas, metadata)     |                            |
    |                               |                            |
    |<----- Registration OK --------|                            |
    |       (temporal_address,      |                            |
    |        temporal_namespace)    |                            |
    |                               |                            |
    |------ Connect Worker ---------------------------------->|
    |       (using platform-provided address & namespace)        |
    |                               |                            |
    |<----- Dispatch Activity --------------------------------|
    |       (when workflow runs)    |                            |
    |                               |                            |
    |------ Return Result ---------------------------------->|
    |                               |                            |
```

> **Note:** Third-party modules never configure Temporal address or namespace directly.
> The platform is the source of truth — it provides these values during registration.

---

## Defining a Module

A module is created with `NewModule` and configured via functional options:

```go
mod := sdk.NewModule("notifications",
    sdk.WithModuleDescription("Email, SMS, and Slack notification activities"),
    sdk.WithModuleVersion("1.0.0"),
    sdk.WithModuleIcon("bell"),
    sdk.WithModuleTaskQueue("notifications-task-queue"),
    sdk.WithModuleAuthor("acme-corp"),
    sdk.WithModuleCategory("notifications"),
    sdk.WithModuleTags("email", "sms", "slack", "alerts"),
)
```

### Module Options

| Option | Required | Description |
|--------|----------|-------------|
| `WithModuleDescription(string)` | No | Human-readable description shown in the Masflow UI |
| `WithModuleVersion(string)` | No | Semantic version for tracking deployments |
| `WithModuleIcon(string)` | No | Icon identifier (e.g., Lucide icon name) for UI display |
| `WithModuleTaskQueue(string)` | **Yes** | Temporal task queue name -- must be unique per module |
| `WithModuleAuthor(string)` | No | Author or team name |
| `WithModuleCategory(string)` | No | Top-level classification (e.g., "notifications", "payments") |
| `WithModuleTags(string...)` | No | Searchable tags for filtering and discovery |

> **Task Queue Naming:** Choose a descriptive, unique name like `{module}-task-queue`. All activities in the module share this queue by default. Individual activities can override it with `WithTaskQueue`.

---

## Registering Activities

### Input/Output Types

Define activity contracts as Go structs with `json` tags. The SDK auto-generates [JSON Schema](https://json-schema.org/) from these types for validation and UI form generation:

```go
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
```

**Best practices for types:**
- Use `json` tags on all fields -- these become the wire format
- Use `omitempty` for optional fields
- Use basic types: `string`, `int`, `float64`, `bool`, `[]T`, `map[string]T`
- Nested structs are supported and generate nested JSON Schema

### Sync Activities (`Register`)

The most common handler type -- takes input, returns output:

```go
func SendEmail(ctx context.Context, input SendEmailInput) (SendEmailOutput, error) {
    // Your implementation here
    recipients := 1 + len(input.Cc)
    return SendEmailOutput{
        MessageID:  generateID(),
        Status:     "sent",
        SentAt:     time.Now().Format(time.RFC3339),
        Recipients: recipients,
    }, nil
}

sdk.Register(mod, "sendEmail", SendEmail,
    sdk.WithDescription("Send an email notification via SMTP/SES"),
    sdk.WithIcon("mail"),
    sdk.WithCategory("email"),
    sdk.WithTags("email", "notification", "smtp"),
    sdk.WithDocumentationURL("https://docs.example.com/activities/send-email"),
)
```

### Void Activities (`RegisterVoid`)

For activities that perform side effects and don't return data:

```go
type LogEventInput struct {
    Message  string `json:"message"`
    Level    string `json:"level"`
    Metadata map[string]string `json:"metadata,omitempty"`
}

func LogEvent(ctx context.Context, input LogEventInput) error {
    // Write to logging system
    return nil
}

sdk.RegisterVoid(mod, "logEvent", LogEvent,
    sdk.WithDescription("Write an event to the audit log"),
    sdk.WithCategory("logging"),
)
```

### Async Activities (`RegisterAsync`)

For long-running operations where the activity starts work and a separate process signals completion:

```go
type CreateTicketInput struct {
    Title       string `json:"title"`
    Description string `json:"description"`
    Assignee    string `json:"assignee"`
}

type CreateTicketOutput struct {
    TicketID string `json:"ticket_id"`
    Status   string `json:"status"`
}

func CreateTicket(ctx context.Context, input CreateTicketInput, async *sdk.AsyncCallbackInfo) (CreateTicketOutput, error) {
    // 1. Create ticket in external system (e.g., Jira)
    ticketID := createJiraTicket(input)

    // 2. Store the callback info so the external system can signal back
    //    when the ticket is resolved
    storeCallback(ticketID, async.WorkflowID, async.RunID, async.CallbackSignal)

    // 3. Return immediately -- the workflow will wait for the signal
    return CreateTicketOutput{
        TicketID: ticketID,
        Status:   "waiting",
    }, nil
}

sdk.RegisterAsync(mod, "createTicket", CreateTicket,
    sdk.WithDescription("Create a Jira ticket and wait for resolution"),
    sdk.WithCategory("ticketing"),
)
```

The workflow pauses at this step until the external system sends a signal via Temporal's `SignalWorkflow` API using the callback info.

### Activity Options

| Option | Description |
|--------|-------------|
| `WithDescription(string)` | Human-readable description for UI and docs |
| `WithIcon(string)` | Icon identifier for visual representation |
| `WithCategory(string)` | Classification (e.g., "email", "database", "ai") |
| `WithTags(string...)` | Searchable tags for filtering |
| `WithTaskQueue(string)` | Override the module's default task queue |
| `WithDocumentationURL(string)` | Link to external activity documentation |

---

## Running the Module

### Using Runner (Recommended)

The `Runner` is the batteries-included entry point that handles everything:

```go
runner, err := sdk.NewRunner(mod,
    sdk.WithPlatformURL("http://localhost:9999"),
    sdk.WithLogger(slog.Default()),
)
if err != nil {
    log.Fatal(err)
}

// Blocks until SIGINT/SIGTERM -- handles graceful shutdown
if err := runner.Run(context.Background()); err != nil {
    log.Fatal(err)
}
```

`Runner.Run()` performs these steps:
1. Registers the module with the Masflow platform
2. Receives Temporal address and namespace from the platform
3. Connects to Temporal using the platform-provided config
4. Creates a Temporal worker on the module's task queue
5. Registers all activity handlers with the worker
6. Starts the worker
7. Blocks until context cancellation or SIGINT/SIGTERM
8. On shutdown: unregisters from platform, stops worker, closes client

### Non-Blocking Mode

For embedding in larger services:

```go
runner, _ := sdk.NewRunner(mod, opts...)

if err := runner.Start(ctx); err != nil {
    log.Fatal(err)
}

// ... do other work ...

// When ready to stop:
runner.Stop(context.Background())
```

### Runner Options

| Option | Default | Description |
|--------|---------|-------------|
| `WithPlatformURL(string)` | — | **Required.** Masflow platform URL. The platform provides Temporal connection details during registration |
| `WithWorkflowURL(string)` | — | Masflow platform URL for WorkflowClient. Enables `runner.Workflows()` |
| `WithLogger(*slog.Logger)` | `slog.Default()` | Structured logger |
| `WithShutdownTimeout(time.Duration)` | `30s` | Max time for graceful shutdown |
| `WithWorkerOptions(worker.Options)` | — | Pass-through Temporal worker options (concurrency, rate limits, etc.) |
| `WithHTTPClient(*http.Client)` | `http.DefaultClient` | HTTP client for platform communication |
| `WithConnectOptions(...connect.ClientOption)` | — | Connect client options for platform communication |

> **Note:** Temporal address and namespace are not configurable by third-party modules. The platform is the single source of truth for these values and returns them during module registration.

---

## Advanced Usage

### Multiple Modules in One Process

```go
emailMod := sdk.NewModule("email", sdk.WithModuleTaskQueue("email-queue"))
sdk.Register(emailMod, "sendEmail", SendEmail)

smsMod := sdk.NewModule("sms", sdk.WithModuleTaskQueue("sms-queue"))
sdk.Register(smsMod, "sendSMS", SendSMS)

// Run each module with its own Runner — each registers independently
g, ctx := errgroup.WithContext(context.Background())

g.Go(func() error {
    r, _ := sdk.NewRunner(emailMod, sdk.WithPlatformURL(platformURL))
    return r.Run(ctx)
})

g.Go(func() error {
    r, _ := sdk.NewRunner(smsMod, sdk.WithPlatformURL(platformURL))
    return r.Run(ctx)
})

g.Wait()
```

### Custom Worker Options

Control concurrency, rate limits, and task polling:

```go
runner, _ := sdk.NewRunner(mod,
    sdk.WithPlatformURL("http://masflow.internal:10000"),
    sdk.WithWorkerOptions(worker.Options{
        MaxConcurrentActivityExecutionSize:      20,
        MaxConcurrentActivityTaskPollers:        4,
        WorkerActivitiesPerSecond:               100,
    }),
)
```

### Inspecting Generated Schemas

Access the auto-generated JSON Schema for debugging or documentation:

```go
if def, ok := mod.GetActivity("sendEmail"); ok {
    fmt.Println("Input Schema:", string(def.InputSchemaJSON))
    fmt.Println("Output Schema:", string(def.OutputSchemaJSON))
    fmt.Println("Input Type URL:", def.InputType)
    fmt.Println("Output Type URL:", def.OutputType)
}
```

---

## Architecture

### Package Layout

```
sdk/
  module.go              Module type and options
  activity.go            Definition type and options
  handler.go             Handler type signatures
  register.go            Register, RegisterVoid, RegisterAsync
  schema.go              JSON Schema generation + type URL inference
  worker.go              RegisterAll (Temporal worker integration)
  runner.go              Runner lifecycle (start/stop/run)
  runner_options.go      RunnerOption functional options
  workflow_client.go     WorkflowClient (execute, status, monitor, lifecycle)
  workflow_types.go      Pure Go types for workflow operations
  platform/
    client.go            Connect/gRPC client for ModuleRegistry
  internal/
    pb/activity/         Generated protobuf + Connect code for module registry
    pb/workflow/         Generated protobuf + Connect code for workflow service
```

### Dependencies

The SDK depends only on:

| Dependency | Purpose |
|------------|---------|
| `go.temporal.io/sdk` | Temporal worker and activity registration |
| `connectrpc.com/connect` | Connect/gRPC client for platform registration |
| `google.golang.org/protobuf` | Protobuf serialization for platform protocol |
| `github.com/invopop/jsonschema` | JSON Schema generation from Go types |

It has **zero dependency** on the Masflow server codebase (`github.com/mas-soft/masflow`).

### Proto Contract

The SDK includes a vendored copy of the `activity.proto` and `workflow.proto` contracts from the server. These define the `ModuleRegistry` and `WorkflowService` gRPC services. The generated code is placed in `internal/pb/` so it is not exposed to SDK consumers.

To regenerate after a proto change:
```bash
cd sdk && make generate
```

---

## API Reference

### Module

```go
// Create a module
mod := sdk.NewModule(name string, opts ...ModuleOption) *Module

// Access activities
mod.Activities() map[string]*Definition
mod.GetActivity(name string) (*Definition, bool)
```

### Registration

```go
// Sync activity: input -> output
sdk.Register[TReq, TRes](mod, name, handler, opts...) error

// Void activity: input -> error only
sdk.RegisterVoid[TReq](mod, name, handler, opts...) error

// Async activity: input + callback info -> output
sdk.RegisterAsync[TReq, TRes](mod, name, handler, opts...) error
```

### Runner

```go
// Create runner (WithPlatformURL is required)
runner, err := sdk.NewRunner(mod, opts...) (*Runner, error)

// Blocking run (with signal handling)
runner.Run(ctx context.Context) error

// Non-blocking start/stop
runner.Start(ctx context.Context) error
runner.Stop(ctx context.Context) error

// Access integrated WorkflowClient (requires WithWorkflowURL)
runner.Workflows() *WorkflowClient
```

### Handler Signatures

```go
// Sync handler
type Handler[TReq, TRes any] func(ctx context.Context, req TReq) (TRes, error)

// Void handler
type VoidHandler[TReq any] func(ctx context.Context, req TReq) error

// Async handler
type AsyncHandler[TReq, TRes any] func(ctx context.Context, req TReq, async *AsyncCallbackInfo) (TRes, error)
```

---

## Workflow Client

The SDK includes a `WorkflowClient` for executing, monitoring, and managing workflows on the Masflow platform. This is useful for building CLI tools, dashboards, or triggering workflows programmatically from your Go services.

### Creating a WorkflowClient

**Standalone** (no Runner needed):

```go
wc := sdk.NewWorkflowClient("http://localhost:9999",
    sdk.WithWorkflowHTTPClient(customHTTPClient),       // optional
    sdk.WithWorkflowConnectOptions(connectOpts...),     // optional
)
```

**Via Runner** (shares HTTP/Connect config):

```go
runner, _ := sdk.NewRunner(mod,
    sdk.WithPlatformURL("http://localhost:9999"),
    sdk.WithWorkflowURL("http://localhost:9999"),
)
runner.Start(ctx)

wc := runner.Workflows() // non-nil when WithWorkflowURL is set
```

### Executing Workflows

```go
// From YAML source
result, err := wc.ExecuteYAML(ctx, yamlString, &sdk.ExecuteSourceOptions{
    WorkflowID: "my-workflow-123",
    Variables:  map[string]any{"env": "staging"},
    TaskQueue:  "custom-queue",
    WorkflowTimeout: 10 * time.Minute,
    Context: map[string]string{"tenant": "acme"},
})
fmt.Printf("Started: %s (status: %s)\n", result.WorkflowID, result.Status)

// From JSON source
result, err := wc.ExecuteJSON(ctx, jsonString, nil)

// From a saved declaration
declResult, err := wc.ExecuteDeclaration(ctx, "decl-uuid", &sdk.ExecuteDeclarationOptions{
    Variables: map[string]any{"amount": 99.99},
})
```

### Querying Status

```go
// Simple status + trace
status, err := wc.GetStatus(ctx, "workflow-id")
fmt.Printf("Status: %s, Trace entries: %d\n", status.Status, len(status.Trace))

// Detailed description (execution info, child workflows, etc.)
info, err := wc.Describe(ctx, "workflow-id", "run-id")
fmt.Printf("Type: %s, Duration: %s, Errors: %v\n",
    info.WorkflowType, info.ExecutionTime, info.HasErrors)
```

### Monitoring & Trace

```go
// Real-time step-level monitoring
monitor, err := wc.Monitor(ctx, "workflow-id", "run-id")
fmt.Printf("Progress: %d/%d (%.0f%%)\n",
    monitor.Progress.CurrentStep, monitor.Progress.TotalSteps, monitor.Progress.Percentage)
for _, step := range monitor.Steps {
    fmt.Printf("  [%s] %s - %s\n", step.StepType, step.Name, step.Status)
}

// BPM execution trace
trace, err := wc.Trace(ctx, "workflow-id", "run-id")
for _, entry := range trace {
    fmt.Printf("%s [%s] %s: %s\n", entry.Timestamp, entry.StepType, entry.Status, entry.Details)
}

// Custom queries
queryResult, err := wc.Query(ctx, "workflow-id", "currentState", nil)
fmt.Printf("Query result: %v\n", queryResult.Result)
```

### Listing & Searching

```go
// List with filters
list, err := wc.List(ctx, &sdk.ListWorkflowsOptions{
    Status:   sdk.WorkflowStatusRunning,
    PageSize: 20,
    OrderBy:  "start_time desc",
    Category: "notifications",
})
for _, w := range list.Workflows {
    fmt.Printf("%s [%s] %s\n", w.WorkflowID, w.Status, w.WorkflowType)
}

// Advanced search
results, err := wc.Search(ctx, &sdk.SearchWorkflowsOptions{
    Query:      "order processing",
    Tags:       []string{"critical"},
    Categories: []string{"orders", "payments"},
    SortBy:     "created_at",
    SortOrder:  "desc",
})
```

### Lifecycle Management

```go
// Signal a running workflow
wc.Signal(ctx, "workflow-id", "approval", map[string]any{"approved": true})

// Cancel (cooperative cancellation)
wc.Cancel(ctx, "workflow-id", "no longer needed")

// Terminate (forceful)
wc.Terminate(ctx, "workflow-id", "stuck workflow")

// Pause and resume
wc.Pause(ctx, "workflow-id", "run-id", "manual intervention")
wc.Resume(ctx, "workflow-id", "run-id", "issue resolved")
```

### Validate

```go
result, err := wc.Validate(ctx, yamlString)
if !result.Valid {
    for _, e := range result.Errors {
        fmt.Println("Error:", e)
    }
}
for _, w := range result.Warnings {
    fmt.Println("Warning:", w)
}
```

### WorkflowClient API Summary

| Method | Description |
|--------|-------------|
| `ExecuteYAML(ctx, yaml, opts)` | Execute workflow from YAML source |
| `ExecuteJSON(ctx, json, opts)` | Execute workflow from JSON source |
| `ExecuteDeclaration(ctx, id, opts)` | Execute a saved workflow declaration |
| `GetStatus(ctx, workflowID)` | Get workflow status and trace |
| `Describe(ctx, workflowID, runID)` | Get detailed workflow execution info |
| `List(ctx, opts)` | List workflows with filters and pagination |
| `Search(ctx, opts)` | Advanced search with facets |
| `Query(ctx, workflowID, queryType, args)` | Send a query to a running workflow |
| `Monitor(ctx, workflowID, runID)` | Get real-time step-level monitoring data |
| `Trace(ctx, workflowID, runID)` | Get BPM execution trace |
| `Cancel(ctx, workflowID, reason)` | Request cooperative cancellation |
| `Signal(ctx, workflowID, signalName, data)` | Send a signal to a workflow |
| `Terminate(ctx, workflowID, reason)` | Forcefully terminate a workflow |
| `Pause(ctx, workflowID, runID, reason)` | Pause a running workflow |
| `Resume(ctx, workflowID, runID, reason)` | Resume a paused workflow |
| `Validate(ctx, yaml)` | Validate workflow YAML without executing |

---

## Workflow YAML Integration

Once your module is registered, workflows can reference your activities by name:

```yaml
name: order-confirmation
description: Send confirmation after order placed
steps:
  - name: send-confirmation
    activity:
      type: sendEmail
      args:
        to: "${order.customer_email}"
        subject: "Order ${order.id} confirmed"
        body: "Thank you for your order!"
      ref: emailResult

  - name: log-sent
    activity:
      type: logEvent
      args:
        message: "Confirmation sent: ${emailResult.message_id}"
        level: info
```

The `type` field in the workflow YAML matches the activity `name` you passed to `Register`. The `args` map is deserialized into your input struct. The `ref` field stores the output for use in subsequent steps via `${refName.field}` expressions.

### Async Activities in YAML

```yaml
steps:
  - name: create-approval-ticket
    activity:
      type: createTicket
      args:
        title: "Approve order ${order.id}"
        assignee: "manager@example.com"
      ref: ticket
      async: true
      callback_signal: "ticket-resolved"
      callback_timeout: "72h"
```

---

## Configuration Reference

### Environment Variables

These are commonly set when deploying a module service:

| Variable | Description | Default |
|----------|-------------|---------|
| `MASFLOW_PLATFORM_URL` | Masflow platform Connect endpoint | — (required) |

> **Note:** Temporal address and namespace are not configured by modules. The platform provides these during registration.

Example `main.go` reading from environment:

```go
runner, _ := sdk.NewRunner(mod,
    sdk.WithPlatformURL(os.Getenv("MASFLOW_PLATFORM_URL")),
)
```

### Docker Deployment

```dockerfile
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /module ./cmd/module

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=builder /module /module
USER nobody:nobody
ENTRYPOINT ["/module"]
```

```yaml
# docker-compose.yml
services:
  my-module:
    build: .
    environment:
      MASFLOW_PLATFORM_URL: http://bpm-service:10000
    depends_on:
      - bpm-service
```

---

## Troubleshooting

### "platform URL is required"

`WithPlatformURL` is required when creating a Runner. The platform provides Temporal connection details during registration:
```go
sdk.NewRunner(mod, sdk.WithPlatformURL("http://localhost:9999"))
```

### "failed to register with masflow platform"

The platform is not reachable. This is fatal — the module cannot start without platform registration. Verify:
- Platform URL is correct
- Platform service is running
- Network/firewall rules allow the connection

### "failed to connect to Temporal"

The Temporal server address returned by the platform is not reachable. This is a platform-side configuration issue. Verify:
- Temporal is running
- The platform's Temporal configuration is correct
- Network between your module and Temporal allows the connection

### "module task queue is required"

You must set a task queue when creating the module:
```go
sdk.NewModule("my-module", sdk.WithModuleTaskQueue("my-task-queue"))
```

### "activity X is already registered"

You're registering the same activity name twice in one module. Activity names must be unique within a module.

### Activities not being picked up

Verify the task queue name in your workflow YAML matches your module's task queue. The workflow engine dispatches activities to the queue specified in the module registration.

---

## License

See the [Masflow License](../LICENSE) for details.
