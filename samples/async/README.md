# Async Sample

Demonstrates async (human-in-the-loop) activities that pause workflow execution until an external system signals completion.

## What it demonstrates

- `RegisterAsync[TReq, TRes]` for async-capable activities
- `AsyncCallbackInfo` with WorkflowID, RunID, CallbackSignal, CallbackTimeout
- Mixing async and sync activities in the same module
- Callback pattern: external system signals Temporal to resume the workflow
- Workflow YAML with `async: true`, `callback_signal`, and `callback_timeout`

## How async activities work

```
Workflow ──> requestApproval ──> creates ticket ──> returns immediately
                                     │
                               (workflow pauses)
                                     │
                               Approver acts ──> approval system signals Temporal
                                     │
                               (workflow resumes)
                                     │
Workflow ──> notifyComplete ──> sends notification
```

## Activities

| Name | Type | Description |
|------|------|-------------|
| `requestApproval` | async | Create approval ticket, wait for human decision |
| `startExternalJob` | async | Start external job, wait for completion signal |
| `notifyComplete` | sync | Send a completion notification |

## Run

```bash
go run . --temporal=localhost:7233
```

## Signaling back

When the external system is ready, it calls Temporal's SignalWorkflow:

```go
client.SignalWorkflow(ctx, workflowID, runID, "approval-decision",
    map[string]any{"approved": true, "comment": "Looks good"})
```

## Example Workflow

See `workflows/approval-flow.yaml` for a complete expense approval flow.
