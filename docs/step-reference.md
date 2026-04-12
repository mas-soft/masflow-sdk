# Step Types Reference

**Complete reference for every workflow step type available in the Masflow engine.**

Each step type includes its YAML syntax, parameters, and a working example. For a tutorial on creating workflows, see the [Workflow Authoring Guide](workflow-guide.md).

---

## Table of Contents

**Activity Execution**
- [activity](#activity) — Execute registered activities

**Control Flow**
- [if](#if) — Conditional branching
- [switch](#switch) — Multi-way branching
- [try_catch](#try_catch) — Error handling

**Iteration**
- [loop](#loop) — While-loop with condition
- [foreach](#foreach) — Collection iteration

**Parallelism**
- [parallel](#parallel) — Concurrent branch execution

**Data and State**
- [context_mutation](#context_mutation) — Modify workflow variables
- [script](#script) — Inline JavaScript execution

**Waiting and Events**
- [wait](#wait) — Pause execution
- [webhook_wait](#webhook_wait) — Wait for external webhook

**Human Tasks**
- [human_task](#human_task) — Human-in-the-loop tasks with forms

**Workflow Composition**
- [child_workflow](#child_workflow) — Execute child workflows
- [subworkflow](#subworkflow) — Inline sub-workflow
- [terminate](#terminate) — Stop workflow execution

**AI / LLM**
- [ai_agent](#ai_agent) — Single-turn LLM call
- [ai_agent_loop](#ai_agent_loop) — Multi-turn agentic loop
- [ai_tool_call](#ai_tool_call) — Direct tool invocation

---

## Activity Execution

### activity

Execute a registered activity from any module. This is the most commonly used step type.

**Parameters:**

| Field              | Required | Description                                                                            |
| ------------------ | -------- | -------------------------------------------------------------------------------------- |
| `type`             | Yes      | Activity name (matches the name passed to `Register` in the SDK)                       |
| `module`           | Yes*     | Module name. Used to derive the task queue as `{module}-task-queue`. **Required for all third-party activities.** Only built-in platform activities may omit this field. |
| `task_queue`       | No       | Explicit task queue override (takes precedence over module-derived queue)               |
| `args`             | No       | Input arguments as a key-value map. Values support `${expression}` template syntax      |
| `ref`              | No       | Binding key to store the activity's output for use in later steps                      |
| `async`            | No       | Set to `true` for async activities (workflow pauses until signal)                      |
| `callback_signal`  | No       | Signal name the external system uses to resume the workflow (required when `async: true`)|
| `callback_timeout` | No       | Maximum time to wait for the callback signal (e.g., `"72h"`, `"30m"`)                 |
| `retry`            | No       | Retry configuration (see below)                                                        |

**Retry Configuration:**

```yaml
retry:
  max_attempts: 3
  initial_interval: "1s"
  max_interval: "30s"
  backoff_coefficient: 2.0
```

**Example — Sync Activity:**

```yaml
- name: send-email
  activity:
    type: sendEmail
    module: notifications
    args:
      to: "${customer_email}"
      subject: "Order ${order_id} confirmed"
      body: "Thank you for your purchase!"
    ref: emailResult
```

**Example — Async Activity:**

```yaml
- name: wait-for-approval
  activity:
    type: requestApproval
    module: approvals
    args:
      title: "Approve expense ${expense_id}"
      assignee: "manager@company.com"
    ref: approvalResult
    async: true
    callback_signal: "approval-decision"
    callback_timeout: "72h"
```

**Example — With Retry:**

```yaml
- name: call-external-api
  activity:
    type: callAPI
    module: integrations
    args:
      url: "https://api.example.com/data"
    ref: apiResponse
    retry:
      max_attempts: 5
      initial_interval: "2s"
      max_interval: "60s"
      backoff_coefficient: 2.0
```

---

## Control Flow

### if

Conditional branching with `then` and optional `else` blocks. The `condition` is a JavaScript-like expression evaluated against workflow bindings.

**Parameters:**

| Field       | Required | Description                                          |
| ----------- | -------- | ---------------------------------------------------- |
| `condition` | Yes      | Expression that evaluates to true/false              |
| `then`      | Yes      | Steps to execute when condition is true              |
| `else`      | No       | Steps to execute when condition is false             |

**Example:**

```yaml
- name: check-amount
  if:
    condition: "${order_total} > 500"
    then:
      - name: apply-discount
        activity:
          type: applyDiscount
          module: pricing
          args:
            order_id: "${order_id}"
            discount_pct: 10
          ref: discount

      - name: notify-vip
        activity:
          type: sendSlack
          module: notifications
          args:
            channel: "#vip-orders"
            message: "High-value order ${order_id}: $${order_total}"
    else:
      - name: standard-processing
        activity:
          type: processOrder
          module: orders
          args:
            order_id: "${order_id}"
```

---

### switch

Multi-way branching based on an expression value. Each case matches a specific value; `default` handles unmatched values.

**Parameters:**

| Field        | Required | Description                                              |
| ------------ | -------- | -------------------------------------------------------- |
| `expression` | Yes      | Expression whose value determines which case to execute  |
| `cases`      | Yes      | Map of value → step list                                 |
| `default`    | No       | Steps to execute if no case matches                      |

**Example:**

```yaml
- name: route-by-priority
  switch:
    expression: "${ticket.priority}"
    cases:
      urgent:
        steps:
          - name: page-on-call
            activity:
              type: sendPagerDuty
              module: alerts
              args:
                severity: critical
                title: "${ticket.title}"
      high:
        steps:
          - name: send-slack-alert
            activity:
              type: sendSlack
              module: notifications
              args:
                channel: "#incidents"
                message: "High priority: ${ticket.title}"
      low:
        steps:
          - name: queue-for-later
            activity:
              type: addToBacklog
              module: ticketing
              args:
                ticket_id: "${ticket.id}"
    default:
      - name: standard-handling
        activity:
          type: assignTicket
          module: ticketing
          args:
            ticket_id: "${ticket.id}"
            queue: "general"
```

---

### try_catch

Structured error handling with `try`, `catch`, and optional `finally` blocks.

**Parameters:**

| Field     | Required | Description                                                      |
| --------- | -------- | ---------------------------------------------------------------- |
| `try`     | Yes      | Steps to attempt                                                 |
| `catch`   | No       | Steps to execute if any step in `try` fails                      |
| `finally` | No       | Steps that always execute, regardless of success or failure      |

**Example:**

```yaml
- name: safe-payment-processing
  try_catch:
    try:
      - name: charge-card
        activity:
          type: chargeCard
          module: payments
          args:
            card_token: "${payment.token}"
            amount: "${order_total}"
          ref: charge

      - name: record-transaction
        activity:
          type: recordTransaction
          module: accounting
          args:
            charge_id: "${charge.transaction_id}"
            amount: "${order_total}"

    catch:
      - name: refund-if-charged
        activity:
          type: refundCharge
          module: payments
          args:
            charge_id: "${charge.transaction_id}"

      - name: notify-failure
        activity:
          type: sendEmail
          module: notifications
          args:
            to: "finance@company.com"
            subject: "Payment failed for order ${order_id}"
            body: "Automatic refund initiated."

    finally:
      - name: log-attempt
        activity:
          type: logEvent
          module: audit
          args:
            message: "Payment attempt for order ${order_id}"
            level: "info"
```

---

## Iteration

### loop

Repeat steps while a condition is true, up to an optional maximum number of iterations. Useful for polling or retry patterns.

**Parameters:**

| Field            | Required | Description                                              |
| ---------------- | -------- | -------------------------------------------------------- |
| `condition`      | Yes      | Expression that is evaluated before each iteration       |
| `max_iterations` | No       | Safety limit on number of iterations (prevents runaway)  |
| `steps`          | Yes      | Steps to execute each iteration                          |

**Built-in loop variables** available inside steps:

| Variable            | Description                    |
| ------------------- | ------------------------------ |
| `__loop_iteration`  | Current iteration number (1-based) |
| `__loop_index`      | Current iteration index (0-based)  |

**Example:**

```yaml
- name: poll-until-ready
  loop:
    condition: "${resource_status} != 'ready'"
    max_iterations: 20
    steps:
      - name: wait-between-polls
        wait:
          duration: "10s"

      - name: check-resource
        activity:
          type: getResourceStatus
          module: infrastructure
          args:
            resource_id: "${resource_id}"
          ref: statusCheck

      - name: update-status
        context_mutation:
          set_variable:
            variable_name: resource_status
            variable_value: "${statusCheck.status}"
```

---

### foreach

Iterate over a collection, executing steps for each item. Supports sequential (default), parallel, and batch execution modes.

**Parameters:**

| Field            | Required | Description                                                          |
| ---------------- | -------- | -------------------------------------------------------------------- |
| `collection`     | Yes      | Expression resolving to an array (e.g., `"${items}"`)                |
| `item_variable`  | No       | Variable name for the current item (default: `"item"`)               |
| `index_variable` | No       | Variable name for the current index (default: `"idx"`)               |
| `mode`           | No       | Execution mode: `"sequential"` (default), `"parallel"`, `"batch"`    |
| `batch_size`     | No       | Number of items per batch (only for `mode: batch`)                   |
| `steps`          | Yes      | Steps to execute for each item                                       |

**Example — Sequential:**

```yaml
- name: process-line-items
  foreach:
    collection: "${order.items}"
    item_variable: item
    index_variable: idx
    steps:
      - name: validate-item
        activity:
          type: validateItem
          module: inventory
          args:
            sku: "${item.sku}"
            quantity: "${item.quantity}"
          ref: itemValidation
```

**Example — Parallel:**

```yaml
- name: send-all-notifications
  foreach:
    collection: "${subscribers}"
    item_variable: subscriber
    mode: parallel
    steps:
      - name: notify-subscriber
        activity:
          type: sendEmail
          module: notifications
          args:
            to: "${subscriber.email}"
            subject: "New update available"
```

**Example — Batch:**

```yaml
- name: import-records
  foreach:
    collection: "${csv_rows}"
    item_variable: row
    mode: batch
    batch_size: 50
    steps:
      - name: insert-record
        activity:
          type: insertRecord
          module: database
          args:
            data: "${row}"
```

---

## Parallelism

### parallel

Execute multiple branches concurrently with configurable completion semantics.

**Parameters:**

| Field      | Required | Description                                                         |
| ---------- | -------- | ------------------------------------------------------------------- |
| `mode`     | No       | Completion mode: `"all"` (default), `"any"`, `"first"`, `"race"`   |
| `branches` | Yes      | List of named branches, each with its own steps                     |

**Modes:**

| Mode    | Behavior                                                                     |
| ------- | ---------------------------------------------------------------------------- |
| `all`   | Wait for every branch to complete. Fail if any branch fails. (Default)       |
| `any`   | Continue when any single branch succeeds. Other branches continue running.   |
| `first` | Use the result of the first branch to complete. Others continue running.     |
| `race`  | First branch to complete wins. All other branches are cancelled.             |

**Branch Parameters:**

| Field       | Required | Description                                            |
| ----------- | -------- | ------------------------------------------------------ |
| `name`      | Yes      | Branch identifier                                      |
| `condition` | No       | Expression; branch only executes if condition is true  |
| `steps`     | Yes      | Steps in this branch                                   |

**Example:**

```yaml
- name: enrich-data
  parallel:
    mode: all
    branches:
      - name: fetch-credit-score
        steps:
          - name: credit-check
            activity:
              type: getCreditScore
              module: underwriting
              args:
                ssn: "${applicant.ssn}"
              ref: creditScore

      - name: fetch-employment
        steps:
          - name: employment-verify
            activity:
              type: verifyEmployment
              module: background-check
              args:
                applicant_id: "${applicant.id}"
              ref: employment

      - name: fetch-address
        condition: "${applicant.needs_address_verify} == true"
        steps:
          - name: address-verify
            activity:
              type: verifyAddress
              module: background-check
              args:
                address: "${applicant.address}"
              ref: addressCheck
```

---

## Data and State

### context_mutation

Directly set or update workflow variables (bindings) without calling an activity.

**Parameters (set_variable):**

| Field            | Required | Description                              |
| ---------------- | -------- | ---------------------------------------- |
| `variable_name`  | Yes      | Name of the variable to set              |
| `variable_value` | Yes      | Value to assign (supports expressions)   |
| `ref`            | No       | Optional ref to store the result         |

**Parameters (update_context):**

Used to update multiple variables at once.

**Example — Set Variable:**

```yaml
- name: set-status
  context_mutation:
    set_variable:
      variable_name: processing_status
      variable_value: "in_progress"
```

**Example — Computed Value:**

```yaml
- name: calculate-total
  context_mutation:
    set_variable:
      variable_name: total_with_tax
      variable_value: "${order_subtotal * 1.08}"
```

**Example — Store from Previous Step:**

```yaml
- name: extract-id
  context_mutation:
    set_variable:
      variable_name: active_order_id
      variable_value: "${orderResult.id}"
```

---

### script

Execute inline JavaScript code within the workflow. The script has access to all workflow bindings and can return a value.

**Parameters:**

| Field    | Required | Description                                   |
| -------- | -------- | --------------------------------------------- |
| `code`   | Yes      | JavaScript code to execute                    |
| `ref`    | No       | Binding key to store the script's return value|

**Example:**

```yaml
- name: compute-discount
  script:
    code: |
      var total = bindings.order_total;
      var tier = bindings.customer_tier;
      if (tier === 'gold') return total * 0.15;
      if (tier === 'silver') return total * 0.10;
      return total * 0.05;
    ref: discount_amount
```

> **Note:** Scripts run inside a sandboxed JavaScript VM (goja). They are deterministic — side effects should be performed via activities, not scripts.

---

## Waiting and Events

### wait

Pause workflow execution for a specified duration, until a timestamp, or until a signal is received.

**Parameters:**

| Field      | Required | Description                                                        |
| ---------- | -------- | ------------------------------------------------------------------ |
| `duration` | Varies   | How long to wait (e.g., `"30s"`, `"5m"`, `"2h"`)                  |
| `until`    | Varies   | ISO 8601 timestamp to wait until                                   |
| `signal`   | Varies   | Signal name to wait for (workflow pauses until signal is received)  |

Exactly one of `duration`, `until`, or `signal` should be specified.

**Example — Duration:**

```yaml
- name: cool-down-period
  wait:
    duration: "5m"
```

**Example — Until Timestamp:**

```yaml
- name: wait-for-market-open
  wait:
    until: "${market_open_time}"
```

**Example — Wait for Signal:**

```yaml
- name: wait-for-confirmation
  wait:
    signal: "user-confirmed"
```

To send the signal from your Go code:

```go
wc.Signal(ctx, workflowID, "user-confirmed", map[string]any{"confirmed": true})
```

---

### webhook_wait

Pause the workflow until an external system sends an HTTP callback to a generated webhook URL. Useful for integrating with third-party services that support webhook notifications.

**Parameters:**

| Field     | Required | Description                                                          |
| --------- | -------- | -------------------------------------------------------------------- |
| `ref`     | No       | Binding key to store the webhook payload when received               |
| `timeout` | No       | Maximum time to wait for the webhook (e.g., `"24h"`)                 |

**Example:**

```yaml
- name: wait-for-payment-webhook
  webhook_wait:
    ref: paymentWebhook
    timeout: "1h"
```

The engine generates a unique callback URL and stores it in the bindings. Configure your external system to POST to this URL when the event occurs.

---

## Human Tasks

### human_task

Create a task for human review or action. The workflow pauses until the assignee completes the task. Supports forms with fields, tabs, and rich content.

**Parameters:**

| Field               | Required | Description                                                           |
| ------------------- | -------- | --------------------------------------------------------------------- |
| `title`             | Yes      | Task title (supports template expressions)                            |
| `description`       | No       | Task description                                                      |
| `assignee`          | No       | User or group to assign the task to                                   |
| `priority`          | No       | `"low"`, `"normal"`, `"high"`, `"urgent"`                             |
| `timeout`           | No       | Maximum time to wait for completion (e.g., `"48h"`)                   |
| `category`          | No       | Task category for filtering                                           |
| `tags`              | No       | Searchable tags                                                       |
| `documentation_url` | No       | Link to documentation or context for the assignee                     |
| `ref`               | No       | Binding key to store the task result (form submissions)               |
| `form`              | No       | Form definition with fields (see below)                               |

**Form Fields:**

| Property          | Required | Description                                                |
| ----------------- | -------- | ---------------------------------------------------------- |
| `name`            | Yes      | Field identifier (used in the result object)               |
| `type`            | Yes      | Field type (see types below)                               |
| `label`           | No       | Display label                                              |
| `required`        | No       | Whether the field must be filled                           |
| `default`         | No       | Default value                                              |
| `description`     | No       | Help text                                                  |
| `options`         | No       | List of options (for `select`, `radio`, `checkbox`)        |
| `group`           | No       | Tab/group name for organizing fields                       |
| `show_when`       | No       | Condition expression — field is visible only when true     |
| `required_when`   | No       | Dynamic required condition                                 |
| `disabled_when`   | No       | Dynamic disabled condition                                 |

**Field Types:** `text`, `textarea`, `select`, `radio`, `checkbox`, `number`, `date`, `time`, `email`, `url`, `tel`, `password`, `hidden`, `color`, `range`, `custom`

**Example — Simple Approval:**

```yaml
- name: manager-review
  human_task:
    title: "Review expense report: ${expense.title}"
    description: "Amount: $${expense.amount} — Submitted by ${expense.submitter}"
    assignee: "${expense.manager_email}"
    priority: high
    timeout: "48h"
    ref: reviewResult
    form:
      fields:
        - name: decision
          type: select
          label: Decision
          required: true
          options:
            - label: Approve
              value: approved
            - label: Reject
              value: rejected
            - label: Request Changes
              value: changes_requested
        - name: comments
          type: textarea
          label: Comments
          description: "Provide feedback for the submitter"
        - name: adjusted_amount
          type: number
          label: Adjusted Amount
          show_when: "${decision} == 'changes_requested'"
```

**Example — Multi-Tab Form:**

```yaml
- name: onboarding-form
  human_task:
    title: "Complete onboarding for ${employee.name}"
    assignee: "hr@company.com"
    priority: normal
    timeout: "5d"
    ref: onboarding
    form:
      fields:
        - name: start_date
          type: date
          label: Start Date
          required: true
          group: Employment Details

        - name: department
          type: select
          label: Department
          required: true
          group: Employment Details
          options:
            - label: Engineering
              value: engineering
            - label: Marketing
              value: marketing
            - label: Sales
              value: sales

        - name: laptop_model
          type: select
          label: Laptop Model
          group: Equipment
          options:
            - label: MacBook Pro 14"
              value: mbp14
            - label: MacBook Pro 16"
              value: mbp16
            - label: Dell XPS 15
              value: dell15

        - name: special_requests
          type: textarea
          label: Special Requests
          group: Equipment
```

Fields with the same `group` value are automatically grouped into tabs in the UI.

---

## Workflow Composition

### child_workflow

Execute another workflow as a child of the current workflow. The parent can wait for the child to complete or continue immediately.

**Parameters:**

| Field                  | Required | Description                                              |
| ---------------------- | -------- | -------------------------------------------------------- |
| `workflow_type`        | Yes      | Name of the child workflow to execute                    |
| `workflow_id`          | No       | Custom workflow ID for the child (auto-generated if omitted) |
| `args`                 | No       | Input arguments for the child workflow                   |
| `result_ref`           | No       | Binding key to store the child workflow's result         |
| `timeout`              | No       | Maximum duration for the child workflow                  |
| `wait_for_completion`  | No       | `true` (default) to wait; `false` to fire-and-forget    |

**Example:**

```yaml
- name: run-sub-processing
  child_workflow:
    workflow_type: data-enrichment
    workflow_id: "enrichment-${record_id}"
    args:
      record_id: "${record_id}"
      source: "main-pipeline"
    result_ref: enrichmentResult
    timeout: "10m"
    wait_for_completion: true
```

**Example — Fire and Forget:**

```yaml
- name: trigger-async-report
  child_workflow:
    workflow_type: generate-report
    args:
      report_type: "daily-summary"
    wait_for_completion: false
```

---

### subworkflow

Define and execute an inline sub-workflow within the current workflow. Useful for encapsulating reusable logic without defining a separate workflow.

**Parameters:**

| Field                 | Required | Description                                          |
| --------------------- | -------- | ---------------------------------------------------- |
| `workflow_id`         | No       | Optional ID for the sub-workflow                     |
| `input_variables`     | No       | Variables to pass into the sub-workflow               |
| `wait_for_completion` | No       | `true` (default) to wait; `false` to run in background |
| `workflow`            | Yes      | Inline workflow definition with `variables` and `steps`|

**Example:**

```yaml
- name: process-payment-sub
  subworkflow:
    input_variables:
      payment_amount: "${order_total}"
      payment_method: "${payment.method}"
    workflow:
      variables:
        payment_amount: 0
        payment_method: ""
      steps:
        - name: validate-payment
          activity:
            type: validatePayment
            module: payments
            args:
              amount: "${payment_amount}"
              method: "${payment_method}"
            ref: validation

        - name: process-charge
          activity:
            type: chargePayment
            module: payments
            args:
              amount: "${payment_amount}"
              validation_token: "${validation.token}"
            ref: charge
```

---

### terminate

Immediately stop workflow execution with an optional exit code and reason. Steps after `terminate` are not executed.

**Parameters:**

| Field       | Required | Description                                  |
| ----------- | -------- | -------------------------------------------- |
| `exit_code` | No       | Numeric exit code (0 = success, non-zero = error) |
| `reason`    | No       | Human-readable termination reason            |

**Example:**

```yaml
- name: check-eligibility
  if:
    condition: "${applicant.age} < 18"
    then:
      - name: reject-underage
        terminate:
          exit_code: 1
          reason: "Applicant must be 18 or older"

- name: continue-processing
  activity:
    type: processApplication
    module: applications
    args:
      applicant_id: "${applicant.id}"
```

---

## AI / LLM

### ai_agent

Execute a single-turn LLM call with an optional structured output schema and tool definitions. The LLM processes the prompt and returns a response in one turn.

**Parameters:**

| Field             | Required | Description                                                          |
| ----------------- | -------- | -------------------------------------------------------------------- |
| `provider`        | Yes      | LLM provider configuration (see below)                              |
| `system_prompt`   | No       | System instructions for the LLM                                     |
| `user_prompt`     | Yes      | User message to send to the LLM (supports template expressions)     |
| `response_schema` | No       | JSON Schema for structured output (LLM response is parsed as JSON)  |
| `history`         | No       | Conversation history (array of `{role, content}` messages)           |
| `history_ref`     | No       | Binding key containing conversation history array                    |
| `tools`           | No       | Tool definitions the LLM can call (see below)                       |
| `timeout`         | No       | Maximum time for the LLM call                                       |
| `ref`             | No       | Binding key to store the LLM response                                |

**Provider Configuration:**

| Field         | Required | Description                                                 |
| ------------- | -------- | ----------------------------------------------------------- |
| `provider`    | Yes      | Provider name: `"anthropic"`, `"openai"`, etc.              |
| `model`       | Yes      | Model identifier (e.g., `"claude-sonnet-4-20250514"`, `"gpt-4o"`) |
| `api_key_env` | Yes      | Environment variable containing the API key                 |
| `temperature` | No       | Sampling temperature (0.0 – 1.0)                            |
| `max_tokens`  | No       | Maximum tokens in the response                              |
| `top_p`       | No       | Nucleus sampling parameter                                  |
| `extra_params`| No       | Additional provider-specific parameters                     |

> **Note:** The API key environment variable must be set on the server where the workflow engine runs. Provider configuration is platform-side — your module does not need API keys unless it implements AI activities directly.

**Example — Structured Classification:**

```yaml
- name: classify-support-ticket
  ai_agent:
    provider:
      provider: anthropic
      model: claude-sonnet-4-20250514
      api_key_env: ANTHROPIC_API_KEY
    system_prompt: |
      You are a support ticket classifier.
      Classify the ticket into exactly one category.
      Respond with JSON matching the provided schema.
    user_prompt: "Classify this ticket: ${ticket.subject} — ${ticket.body}"
    response_schema:
      type: object
      properties:
        category:
          type: string
          enum: [billing, technical, account, general]
        confidence:
          type: number
          minimum: 0
          maximum: 1
        reasoning:
          type: string
    ref: classification
```

**Example — With Tools:**

```yaml
- name: smart-lookup
  ai_agent:
    provider:
      provider: openai
      model: gpt-4o
      api_key_env: OPENAI_API_KEY
    system_prompt: "You are a helpful assistant with access to a knowledge base."
    user_prompt: "Find the pricing for ${product_name}"
    tools:
      - name: searchKnowledgeBase
        description: Search the company knowledge base
        activity_type: searchKB
        task_queue: kb-task-queue
        parameters_schema:
          type: object
          properties:
            query:
              type: string
    ref: lookupResult
```

---

### ai_agent_loop

Execute a multi-turn agentic loop where the LLM can call tools iteratively until a task is complete or a stop condition is met.

**Parameters:**

| Field               | Required | Description                                                            |
| ------------------- | -------- | ---------------------------------------------------------------------- |
| `provider`          | Yes      | LLM provider configuration (same as `ai_agent`)                       |
| `system_prompt`     | No       | System instructions                                                    |
| `user_prompt`       | Yes      | Initial user message                                                   |
| `tools`             | Yes      | Tools the agent can invoke (activity-backed or inline steps)           |
| `max_iterations`    | No       | Maximum number of LLM turns (default depends on platform config)       |
| `max_tokens_budget` | No       | Total token budget across all turns                                    |
| `stop_condition`    | No       | Expression to evaluate after each turn (stops loop when true)          |
| `initial_history`   | No       | Pre-existing conversation history                                      |
| `history_ref`       | No       | Binding key containing conversation history                            |
| `timeout`           | No       | Overall timeout for the entire loop                                    |
| `on_error`          | No       | Error handling strategy: `"stop"` or `"continue"`                      |
| `ref`               | No       | Binding key to store the final result                                  |

**Built-in loop variables** (accessible in `stop_condition`):

| Variable             | Description                        |
| -------------------- | ---------------------------------- |
| `__ai_iteration`     | Current iteration number           |
| `__ai_total_tokens`  | Total tokens consumed so far       |

**Tool Definition:**

| Field               | Required | Description                                               |
| ------------------- | -------- | --------------------------------------------------------- |
| `name`              | Yes      | Tool name (shown to the LLM)                              |
| `description`       | Yes      | What the tool does (LLM uses this to decide when to call) |
| `parameters_schema` | No       | JSON Schema for the tool's input parameters               |
| `activity_type`     | Varies   | Activity name to execute (mutually exclusive with `steps`)|
| `task_queue`        | No       | Task queue for the activity                               |
| `steps`             | Varies   | Inline steps to execute (mutually exclusive with `activity_type`) |

**Example:**

```yaml
- name: research-and-summarize
  ai_agent_loop:
    provider:
      provider: anthropic
      model: claude-sonnet-4-20250514
      api_key_env: ANTHROPIC_API_KEY
    system_prompt: |
      You are a research assistant. Use the available tools to gather
      information, then compile your findings into a summary. Stop
      when you have enough information to write a comprehensive answer.
    user_prompt: "Research: ${research_question}"
    max_iterations: 10
    max_tokens_budget: 50000
    stop_condition: "${__ai_iteration} >= 8"
    tools:
      - name: searchWeb
        description: Search the web for current information
        activity_type: webSearch
        task_queue: search-task-queue
        parameters_schema:
          type: object
          properties:
            query:
              type: string
              description: Search query

      - name: readDocument
        description: Read and extract text from a document URL
        activity_type: fetchDocument
        task_queue: search-task-queue
        parameters_schema:
          type: object
          properties:
            url:
              type: string
              description: URL to fetch
    ref: research_output
```

---

### ai_tool_call

Execute a direct tool invocation from within an AI workflow — typically used when the LLM has decided which tool to call and with what parameters.

**Parameters:**

| Field              | Required | Description                                                  |
| ------------------ | -------- | ------------------------------------------------------------ |
| `tool_name`        | Yes      | Name of the tool to invoke                                   |
| `activity_type`    | No       | Activity to execute for this tool call                       |
| `task_queue`       | No       | Task queue for the activity                                  |
| `args`             | No       | Arguments to pass to the tool                                |
| `ref`              | No       | Binding key to store the tool result                         |

**Example:**

```yaml
- name: execute-search
  ai_tool_call:
    tool_name: searchWeb
    activity_type: webSearch
    task_queue: search-task-queue
    args:
      query: "${search_query}"
    ref: searchResult
```

---

## Quick Reference Table

| Step Type          | Key                | Purpose                                |
| ------------------ | ------------------- | -------------------------------------- |
| Activity           | `activity`          | Call a registered activity             |
| If                 | `if`                | Conditional branching                  |
| Switch             | `switch`            | Multi-way branching                    |
| Try/Catch          | `try_catch`         | Error handling                         |
| Loop               | `loop`              | While-loop with condition              |
| ForEach            | `foreach`           | Iterate over collections               |
| Parallel           | `parallel`          | Concurrent branches                    |
| Context Mutation   | `context_mutation`  | Set/update workflow variables          |
| Script             | `script`            | Inline JavaScript                      |
| Wait               | `wait`              | Pause (duration, timestamp, or signal) |
| Webhook Wait       | `webhook_wait`      | Wait for external webhook              |
| Human Task         | `human_task`        | Human-in-the-loop with forms           |
| Child Workflow     | `child_workflow`    | Execute child workflow                 |
| Subworkflow        | `subworkflow`       | Inline sub-workflow                    |
| Terminate          | `terminate`         | Stop workflow execution                |
| AI Agent           | `ai_agent`          | Single-turn LLM call                  |
| AI Agent Loop      | `ai_agent_loop`     | Multi-turn agentic loop               |
| AI Tool Call       | `ai_tool_call`      | Direct LLM tool invocation            |

---

## See Also

- **[Workflow Authoring Guide](workflow-guide.md)** — Tutorial for creating workflows step by step
- **[SDK README](../README.md)** — Building and registering activity modules
- **[Samples](../samples/)** — Working code examples
