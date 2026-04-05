# Advanced Sample

A full-featured notifications module with five activities across multiple channels.

## What it demonstrates

- Multiple activity registrations (email, SMS, Slack, webhook, log)
- `Register[TReq, TRes]` for sync activities with typed input/output
- `RegisterVoid[TReq]` for fire-and-forget activities
- Rich metadata: icons, categories, tags, documentation URLs
- Input validation patterns
- Environment-based configuration with CLI flag overrides
- Example workflow YAML using all five activities

## Activities

| Name | Type | Description |
|------|------|-------------|
| `sendEmail` | sync | Send an email with To, Cc, Subject, Body |
| `sendSMS` | sync | Send an SMS via phone number |
| `sendSlack` | sync | Post a message to a Slack channel |
| `sendWebhook` | sync | Send an HTTP webhook |
| `logNotification` | void | Write a notification audit log entry |

## Run

```bash
go run . --temporal=localhost:7233 --platform=http://localhost:9999
```

## Test

```bash
go test -v ./...
```

## Example Workflow

See `workflows/order-notifications.yaml` for a multi-step workflow using all five activities.
