# Workflow Authoring Guide

**A step-by-step guide for third-party developers to create, execute, and manage workflows on the Masflow platform using the Go SDK.**

This guide walks you through the entire process of authoring workflows — from writing your first YAML definition to executing and monitoring it programmatically. For a reference of every available step type, see the [Step Types Reference](step-reference.md).

---

## Table of Contents

1. [Overview](#overview)
2. [Workflow YAML Structure](#workflow-yaml-structure)
3. [Template Expressions](#template-expressions)
4. [Data Flow Between Steps](#data-flow-between-steps)
5. [Creating Your First Workflow](#creating-your-first-workflow)
6. [Executing Workflows](#executing-workflows)
7. [Monitoring and Debugging](#monitoring-and-debugging)
8. [Common Patterns](#common-patterns)
   - [Sequential Processing](#sequential-processing)
   - [Conditional Branching](#conditional-branching)
   - [Parallel Execution](#parallel-execution)
   - [Iterating Over Collections](#iterating-over-collections)
   - [Async and Human-in-the-Loop](#async-and-human-in-the-loop)
   - [Error Handling](#error-handling)
   - [AI-Powered Workflows](#ai-powered-workflows)
9. [Best Practices](#best-practices)

---

## Overview

A Masflow workflow is a YAML document that defines a sequence of **steps**. Each step performs one operation — calling an activity, branching on a condition, looping over a collection, waiting for a signal, etc. The Masflow engine executes these steps in order, passing data between them via **workflow bindings** (variables).

As a third-party developer, your role is twofold:

1. **Build activity modules** using the SDK (covered in the main [README](../README.md))
2. **Author workflows** in YAML that orchestrate those activities

Workflows can use any combination of your custom activities and built-in step types (conditionals, loops, parallel execution, human tasks, AI agents, etc.).

---

## Workflow YAML Structure

Every workflow follows this top-level structure:

```yaml
name: my-workflow
description: A short description of what this workflow does
category: orders
tags:
  - processing
  - notifications

workflow:
  variables:
    customer_name: ""
    order_total: 0
    send_notification: true

  steps:
    - name: step-one
      activity:
        type: myActivity
        module: my-module
        args:
          input_field: "${customer_name}"
        ref: stepOneResult

    - name: step-two
      activity:
        type: anotherActivity
        module: my-module
        args:
          data: "${stepOneResult.output_field}"
        ref: stepTwoResult
```

### Top-Level Fields

| Field         | Required | Description                                                     |
| ------------- | -------- | --------------------------------------------------------------- |
| `name`        | Yes      | Unique workflow name identifier                                 |
| `description` | No       | Human-readable description                                      |
| `category`    | No       | Classification for filtering in the UI                          |
| `tags`        | No       | Searchable tags (array of strings)                              |
| `workflow`    | Yes      | The workflow definition containing `variables` and `steps`      |

### Workflow Definition Fields

| Field       | Required | Description                                                                      |
| ----------- | -------- | -------------------------------------------------------------------------------- |
| `variables` | No       | Initial values for workflow bindings. These are the "inputs" to your workflow.    |
| `steps`     | Yes      | Ordered list of step definitions to execute.                                     |

### Step Structure

Each step in the `steps` array has a `name` and exactly one step type key:

```yaml
- name: descriptive-step-name    # Required, unique within workflow
  description: Optional text     # Optional
  activity:                      # One of: activity, if, switch, parallel, loop, foreach,
    ...                          #         wait, human_task, context_mutation, child_workflow,
                                 #         subworkflow, script, terminate, ai_agent, ai_agent_loop
```

The step type key determines what operation is performed. See the [Step Types Reference](step-reference.md) for the full list.

---

## Template Expressions

Masflow uses `${expression}` syntax to dynamically inject values into step arguments, conditions, and other fields. Expressions reference workflow bindings — the shared data store that accumulates results as the workflow runs.

### Variable References

```yaml
# Reference a workflow variable
args:
  name: "${customer_name}"

# Reference a nested field
args:
  email: "${order.customer.email}"

# Reference a step output (via ref)
args:
  previous_result: "${stepOneResult.message_id}"
```

### Where Expressions Are Evaluated

Expressions can appear in:

- **Activity args** — `args: { to: "${email}" }`
- **Conditions** — `condition: "${order_total} > 100"`
- **Human task fields** — `title: "Review order ${order_id}"`
- **Wait durations** — `duration: "${delay_seconds}s"`
- **Loop/foreach** — `collection: "${items}"`
- **Context mutations** — `variable_value: "${stepResult.count + 1}"`
- **AI prompts** — `user_prompt: "Summarize: ${document_text}"`

### Expression Syntax

Conditions and computed values support JavaScript-like expressions:

```yaml
# Comparison
condition: "${order_total} > 100"

# Logical operators
condition: "${is_premium} == true && ${order_total} > 50"

# String matching
condition: "${status} == 'approved'"

# Nested access
condition: "${result.items.length} > 0"
```

---

## Data Flow Between Steps

Data flows through a workflow via **bindings** — a shared key-value context that every step can read from and write to.

### How It Works

1. **Variables** declared in `workflow.variables` are the initial bindings
2. **`ref`** on an activity step stores its output into bindings under that key
3. Subsequent steps access previous results via `${refName.field}` expressions

> **Important:** All third-party activity steps must include the `module` field. Only built-in platform activities may omit it. The engine uses the module name to route the activity to the correct task queue (`{module}-task-queue`).

```yaml
workflow:
  variables:
    customer_email: "alice@example.com"   # Initial binding

  steps:
    # Step 1: Output stored as "orderData"
    - name: fetch-order
      activity:
        type: getOrder
        module: orders
        args:
          email: "${customer_email}"      # Reads initial variable
        ref: orderData                    # Stores output in bindings

    # Step 2: Reads from Step 1's output
    - name: send-receipt
      activity:
        type: sendEmail
        module: notifications
        args:
          to: "${customer_email}"
          subject: "Order ${orderData.order_id} confirmed"  # Reads Step 1 output
          body: "Total: $${orderData.total}"
        ref: emailResult

    # Step 3: Reads from both Step 1 and Step 2
    - name: log-confirmation
      activity:
        type: logEvent
        module: notifications
        args:
          message: "Sent ${emailResult.message_id} for order ${orderData.order_id}"
          level: "info"
```

### Binding Lifecycle

```
Start:     { customer_email: "alice@example.com" }
                           │
After Step 1:  + { orderData: { order_id: "ORD-123", total: 59.99 } }
                           │
After Step 2:  + { emailResult: { message_id: "msg-456", status: "sent" } }
                           │
After Step 3:  (no ref, binding unchanged)
```

### Context Mutations

You can also set or update variables directly using the `context_mutation` step type:

```yaml
- name: update-status
  context_mutation:
    set_variable:
      variable_name: order_status
      variable_value: "processing"
```

---

## Creating Your First Workflow

Let's build a workflow from scratch. We'll assume you have a `greeter` module with a `greet` activity (from the [basic sample](../samples/basic/)).

### Step 1: Write the YAML

Create a file `greeting-workflow.yaml`:

```yaml
name: greeting-workflow
description: Send a personalized greeting

workflow:
  variables:
    name: ""

  steps:
    - name: send-greeting
      activity:
        type: greet
        module: greeter
        args:
          name: "${name}"
        ref: greetResult
```

This workflow:
1. Accepts a `name` variable as input
2. Calls the `greet` activity from the `greeter` module
3. Stores the result as `greetResult`

### Step 2: Start Your Module

Make sure your module is running and registered with the platform:

```go
// main.go
mod := sdk.NewModule("greeter",
    sdk.WithModuleTaskQueue("greeter-task-queue"),
)
sdk.Register(mod, "greet", Greet)

runner, _ := sdk.NewRunner(mod,
    sdk.WithPlatformURL("http://localhost:9999"),
    sdk.WithWorkflowURL("http://localhost:9999"),
)
runner.Run(context.Background())
```

### Step 3: Execute the Workflow

Use the `WorkflowClient` to execute your workflow:

```go
wc := runner.Workflows()

result, err := wc.ExecuteYAML(ctx, greetingYAML, &sdk.ExecuteSourceOptions{
    WorkflowID: "greeting-001",
    Variables:  map[string]any{"name": "Alice"},
})
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Workflow %s started (status: %s)\n", result.WorkflowID, result.Status)
```

### Step 4: Check the Result

```go
status, err := wc.GetStatus(ctx, result.WorkflowID)
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Status: %s\n", status.Status)
for _, t := range status.Trace {
    fmt.Printf("  [%s] %s → %s\n", t.StepType, t.Name, t.Status)
}
```

Output:

```
Workflow greeting-001 started (status: RUNNING)
Status: COMPLETED
  [activity] send-greeting → completed
```

---

## Executing Workflows

### From YAML

The most common method — provide the workflow definition inline:

```go
wc := sdk.NewWorkflowClient("http://localhost:9999")

result, err := wc.ExecuteYAML(ctx, yamlString, &sdk.ExecuteSourceOptions{
    WorkflowID:      "unique-workflow-id",
    Variables:        map[string]any{"key": "value"},
    TaskQueue:        "custom-queue",          // optional override
    WorkflowTimeout:  10 * time.Minute,        // optional timeout
    Context:          map[string]string{        // optional metadata
        "tenant": "acme",
    },
})
```

### From JSON

Same as YAML but from a JSON document:

```go
result, err := wc.ExecuteJSON(ctx, jsonString, &sdk.ExecuteSourceOptions{
    WorkflowID: "json-workflow-001",
    Variables:  map[string]any{"env": "production"},
})
```

### From a Saved Declaration

Execute a workflow that was previously stored on the platform:

```go
result, err := wc.ExecuteDeclaration(ctx, "declaration-uuid", &sdk.ExecuteDeclarationOptions{
    Variables: map[string]any{"amount": 99.99},
})
fmt.Printf("Execution: %s (Declaration: %s)\n", result.ExecutionID, result.DeclarationID)
```

### Validation Without Execution

Check syntax and structure before running:

```go
validation, err := wc.Validate(ctx, yamlString)
if !validation.Valid {
    for _, e := range validation.Errors {
        fmt.Println("Error:", e)
    }
    return
}
fmt.Println("Workflow is valid")
```

---

## Monitoring and Debugging

### GetStatus — Quick Status Check

Returns the workflow status and execution trace:

```go
status, err := wc.GetStatus(ctx, "workflow-id")
fmt.Printf("Status: %s\n", status.Status)

for _, entry := range status.Trace {
    fmt.Printf("  %s [%s] %s: %s\n",
        entry.Timestamp, entry.StepType, entry.Status, entry.Details)
}
```

### Monitor — Real-Time Step Progress

Get detailed per-step progress, completion percentage, and timing:

```go
monitor, err := wc.Monitor(ctx, "workflow-id", "run-id")

fmt.Printf("Step %d of %d (%.0f%%)\n",
    monitor.Progress.CurrentStep,
    monitor.Progress.TotalSteps,
    monitor.Progress.Percentage)

for _, step := range monitor.Steps {
    fmt.Printf("  [%s] %-20s %s (%dms)\n",
        step.StepType, step.Name, step.Status, step.DurationMs)
}
```

### Describe — Full Execution Details

Get comprehensive metadata including timing, parent/child relationships, and errors:

```go
info, err := wc.Describe(ctx, "workflow-id", "run-id")

fmt.Printf("Type: %s\n", info.WorkflowType)
fmt.Printf("Started: %s\n", info.StartTime)
fmt.Printf("Duration: %s\n", info.ExecutionTime)
fmt.Printf("Attempts: %d\n", info.AttemptCount)
fmt.Printf("Has Errors: %v\n", info.HasErrors)
```

### Trace — BPM-Level Execution Log

Step-by-step execution log with timing and details:

```go
trace, err := wc.Trace(ctx, "workflow-id", "run-id")
for _, entry := range trace {
    fmt.Printf("%s [%s] %s: %s\n",
        entry.Timestamp, entry.StepType, entry.Status, entry.Details)
}
```

### Polling for Completion

A common pattern: execute and poll until done:

```go
result, _ := wc.ExecuteYAML(ctx, yaml, opts)

for {
    status, _ := wc.GetStatus(ctx, result.WorkflowID)
    switch status.Status {
    case sdk.WorkflowStatusCompleted:
        fmt.Println("Done!")
        return
    case sdk.WorkflowStatusFailed:
        fmt.Println("Failed:", status.Error)
        return
    case sdk.WorkflowStatusCancelled:
        fmt.Println("Cancelled")
        return
    }
    time.Sleep(2 * time.Second)
}
```

---

## Common Patterns

### Sequential Processing

The simplest pattern — steps execute one after another. Each step can use outputs from previous steps.

```yaml
name: order-processing
description: Process an order end to end

workflow:
  variables:
    order_id: ""

  steps:
    - name: validate-order
      activity:
        type: validateOrder
        module: orders
        args:
          order_id: "${order_id}"
        ref: validation

    - name: charge-payment
      activity:
        type: chargeCard
        module: payments
        args:
          order_id: "${order_id}"
          amount: "${validation.total}"
        ref: payment

    - name: send-confirmation
      activity:
        type: sendEmail
        module: notifications
        args:
          to: "${validation.customer_email}"
          subject: "Order ${order_id} confirmed"
          body: "Payment of $${payment.amount_charged} received."
        ref: email
```

### Conditional Branching

Use `if` steps to branch based on runtime conditions:

```yaml
name: tiered-notification
description: Send different notifications based on order value

workflow:
  variables:
    order_id: ""
    order_total: 0
    customer_email: ""

  steps:
    - name: check-order-value
      if:
        condition: "${order_total} > 1000"
        then:
          - name: send-vip-email
            activity:
              type: sendEmail
              module: notifications
              args:
                to: "${customer_email}"
                subject: "VIP Order ${order_id} — Thank You!"
                body: "As a valued customer, enjoy free expedited shipping."

          - name: notify-sales-team
            activity:
              type: sendSlack
              module: notifications
              args:
                channel: "#high-value-orders"
                message: "VIP order ${order_id}: $${order_total}"
        else:
          - name: send-standard-email
            activity:
              type: sendEmail
              module: notifications
              args:
                to: "${customer_email}"
                subject: "Order ${order_id} — Confirmed"
                body: "Your order has been placed successfully."
```

For multi-way branching, use `switch`:

```yaml
    - name: route-by-region
      switch:
        expression: "${customer_region}"
        cases:
          us:
            steps:
              - name: us-processing
                activity:
                  type: processUSOrder
                  module: orders
                  args:
                    order_id: "${order_id}"
          eu:
            steps:
              - name: eu-processing
                activity:
                  type: processEUOrder
                  module: orders
                  args:
                    order_id: "${order_id}"
        default:
          - name: default-processing
            activity:
              type: processOrder
              module: orders
              args:
                order_id: "${order_id}"
```

### Parallel Execution

Run multiple branches concurrently:

```yaml
name: multi-channel-notification
description: Send notifications on all channels at once

workflow:
  variables:
    message: ""
    recipient_email: ""
    phone_number: ""

  steps:
    - name: notify-all-channels
      parallel:
        mode: all    # Wait for all branches to complete
        branches:
          - name: email-branch
            steps:
              - name: send-email
                activity:
                  type: sendEmail
                  module: notifications
                  args:
                    to: "${recipient_email}"
                    subject: "Alert"
                    body: "${message}"

          - name: sms-branch
            steps:
              - name: send-sms
                activity:
                  type: sendSMS
                  module: notifications
                  args:
                    phone: "${phone_number}"
                    message: "${message}"

          - name: slack-branch
            steps:
              - name: send-slack
                activity:
                  type: sendSlack
                  module: notifications
                  args:
                    channel: "#alerts"
                    message: "${message}"
```

Parallel modes:
- `all` — Wait for every branch to complete (default)
- `any` — Continue as soon as any one branch completes
- `first` — Use the result of the first branch to complete
- `race` — All branches run; first completion wins, others are cancelled

### Iterating Over Collections

Use `foreach` to process each item in a list:

```yaml
name: batch-email-send
description: Send personalized emails to a list of recipients

workflow:
  variables:
    recipients: []
    campaign_name: ""

  steps:
    - name: send-to-each
      foreach:
        collection: "${recipients}"
        item_variable: recipient
        index_variable: idx
        steps:
          - name: send-campaign-email
            activity:
              type: sendEmail
              module: notifications
              args:
                to: "${recipient.email}"
                subject: "${campaign_name}"
                body: "Hello ${recipient.name}, check out our latest updates!"
```

Use `loop` for count-based or condition-based iteration:

```yaml
    - name: retry-check
      loop:
        condition: "${check_result.ready} != true"
        max_iterations: 10
        steps:
          - name: wait-between-checks
            wait:
              duration: "5s"

          - name: check-status
            activity:
              type: checkReadiness
              module: provisioning
              args:
                resource_id: "${resource_id}"
              ref: check_result
```

### Async and Human-in-the-Loop

Async activities pause the workflow until an external signal arrives — ideal for approvals, manual reviews, and external system integration.

#### Writing an Async Activity

```go
func RequestApproval(ctx context.Context, in ApprovalRequest, async *sdk.AsyncCallbackInfo) (ApprovalResult, error) {
    // Create a ticket in your external system
    ticketID := createTicket(in.Title, in.Description, in.Assignee)

    // Store the callback info so the external system can signal completion
    storeCallback(ticketID, CallbackInfo{
        WorkflowID:     async.WorkflowID,
        RunID:          async.RunID,
        CallbackSignal: async.CallbackSignal,
    })

    // Return immediately — the workflow pauses here
    return ApprovalResult{TicketID: ticketID, Status: "pending"}, nil
}

sdk.RegisterAsync(mod, "requestApproval", RequestApproval)
```

#### Workflow YAML with Async Steps

```yaml
name: expense-approval
description: Submit expense for manager approval

workflow:
  variables:
    expense_id: ""
    amount: 0
    submitter: ""

  steps:
    - name: request-manager-approval
      activity:
        type: requestApproval
        module: approvals
        args:
          title: "Expense ${expense_id}: $${amount}"
          description: "Submitted by ${submitter}"
          assignee: "manager@company.com"
        ref: approval
        async: true
        callback_signal: "approval-decision"
        callback_timeout: "72h"

    - name: check-decision
      if:
        condition: "${approval.decision} == 'approved'"
        then:
          - name: process-reimbursement
            activity:
              type: processReimbursement
              module: finance
              args:
                expense_id: "${expense_id}"
                amount: "${amount}"
        else:
          - name: notify-rejection
            activity:
              type: sendEmail
              module: notifications
              args:
                to: "${submitter}"
                subject: "Expense ${expense_id} rejected"
                body: "Reason: ${approval.reason}"
```

#### Signaling Completion

From another Go service or CLI tool:

```go
wc := sdk.NewWorkflowClient("http://localhost:9999")

// The external system sends the approval decision
err := wc.Signal(ctx, "expense-approval-001", "approval-decision", map[string]any{
    "decision": "approved",
    "reason":   "Within budget",
})
```

### Error Handling

Use `try_catch` for structured error handling:

```yaml
name: resilient-processing
description: Process with error recovery

workflow:
  variables:
    order_id: ""

  steps:
    - name: safe-processing
      try_catch:
        try:
          - name: process-payment
            activity:
              type: chargeCard
              module: payments
              args:
                order_id: "${order_id}"
              ref: payment

          - name: ship-order
            activity:
              type: createShipment
              module: shipping
              args:
                order_id: "${order_id}"
              ref: shipment

        catch:
          - name: handle-failure
            activity:
              type: sendEmail
              module: notifications
              args:
                to: "ops-team@company.com"
                subject: "Order ${order_id} processing failed"
                body: "Error encountered during processing. Manual review required."

          - name: mark-failed
            context_mutation:
              set_variable:
                variable_name: order_status
                variable_value: "failed"

        finally:
          - name: audit-log
            activity:
              type: logEvent
              module: notifications
              args:
                message: "Order ${order_id} processing attempt complete"
                level: "info"
```

### AI-Powered Workflows

Integrate LLM-powered steps for classification, summarization, and agentic workflows.

#### Single-Turn AI Call

```yaml
    - name: classify-ticket
      ai_agent:
        provider:
          provider: anthropic
          model: claude-sonnet-4-20250514
          api_key_env: ANTHROPIC_API_KEY
        system_prompt: |
          You are a support ticket classifier. Classify tickets into:
          billing, technical, account, general.
          Respond with JSON only.
        user_prompt: "Classify this ticket: ${ticket.description}"
        response_schema:
          type: object
          properties:
            category:
              type: string
              enum: [billing, technical, account, general]
            confidence:
              type: number
        ref: classification
```

#### Multi-Turn AI Agent Loop

```yaml
    - name: research-agent
      ai_agent_loop:
        provider:
          provider: openai
          model: gpt-4o
          api_key_env: OPENAI_API_KEY
        system_prompt: |
          You are a research assistant. Use the provided tools to
          gather information and compile a summary.
        user_prompt: "Research ${topic} and provide a comprehensive summary."
        max_iterations: 5
        tools:
          - name: webSearch
            description: Search the web for information
            activity_type: webSearch
            task_queue: search-task-queue
            parameters_schema:
              type: object
              properties:
                query:
                  type: string
        ref: research_result
```

---

## Best Practices

### Naming Conventions

- **Workflow names**: Use lowercase kebab-case (`order-processing`, `user-onboarding`)
- **Step names**: Descriptive kebab-case (`validate-order`, `send-confirmation`)
- **Ref names**: camelCase matching the output concept (`orderData`, `emailResult`)
- **Variables**: snake_case (`customer_email`, `order_total`)

### Workflow Design

- **Keep steps focused** — Each step should do one thing. Prefer more small steps over fewer large ones.
- **Use meaningful refs** — Name refs after what the data represents, not the step that produced it.
- **Declare all variables** — List all expected inputs in `workflow.variables` with sensible defaults.
- **Validate early** — Use `wc.Validate()` to check workflow syntax before execution.

### Error Handling

- **Wrap critical sections** in `try_catch` blocks to gracefully handle failures.
- **Use `finally`** for cleanup or audit logging that must run regardless of success or failure.
- **Configure retries** at the activity level for transient failures (network timeouts, rate limits).

### Performance

- **Use `parallel`** for independent operations (sending notifications on multiple channels, independent API calls).
- **Use `foreach` batch mode** for large collections to avoid overwhelming downstream services.
- **Set appropriate timeouts** on workflows (`WorkflowTimeout`) and async steps (`callback_timeout`).

### Module Organization

- **Group related activities** in one module: a "notifications" module with email, SMS, Slack, and webhook activities.
- **Use separate task queues** for different modules to isolate workloads and scale independently.
- **Version your modules** with `WithModuleVersion` for deployment tracking.

---

## Next Steps

- **[Step Types Reference](step-reference.md)** — Detailed YAML syntax and parameters for every step type
- **[SDK README](../README.md)** — Building and registering activity modules
- **[Samples](../samples/)** — Working code examples:
  - [basic/](../samples/basic/) — Minimal module with one activity
  - [advanced/](../samples/advanced/) — Multi-activity module with void handlers
  - [async/](../samples/async/) — Async activities with callback patterns
  - [workflow-client/](../samples/workflow-client/) — CLI tool for workflow management
  - [multi-module/](../samples/multi-module/) — Multiple modules in one process
