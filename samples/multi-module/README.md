# Multi-Module Sample

Run multiple independent modules in a single Go process, each with its own Temporal task queue.

## What it demonstrates

- Creating and configuring multiple `Module` instances
- Running multiple `Runner` instances concurrently with `errgroup`
- Shared signal handling across runners
- Cross-module workflows (a single workflow can use activities from different modules)
- Per-module logging with `slog.With`

## Modules

### Email Module (`email-task-queue`)

| Activity | Description |
|----------|-------------|
| `sendEmail` | Send an email |
| `renderTemplate` | Render an email template |

### Analytics Module (`analytics-task-queue`)

| Activity | Description |
|----------|-------------|
| `trackEvent` | Track a user event |
| `aggregate` | Aggregate metrics |

## Run

```bash
go run . --temporal=localhost:7233
```

Both modules start on their respective task queues. A workflow can reference activities from either module -- Temporal routes each activity to the correct queue.

## Example Workflow

See `workflows/cross-module.yaml` for a user onboarding flow that uses activities from both modules.
