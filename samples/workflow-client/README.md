# Workflow Client Sample

A CLI tool that demonstrates the SDK's `WorkflowClient` for executing, monitoring, and managing workflows programmatically -- no Temporal worker needed.

## What it demonstrates

- Standalone `WorkflowClient` usage (independent of Runner)
- Executing workflows from YAML files
- Querying workflow status and detailed descriptions
- Real-time step-level monitoring
- Listing and searching workflows
- Lifecycle operations: cancel, pause, resume, signal
- Workflow validation

## Commands

```bash
# Execute a workflow
go run . --url http://localhost:10000 execute --yaml example.yaml

# Check status
go run . --url http://localhost:10000 status <workflow-id>

# Detailed info
go run . --url http://localhost:10000 describe <workflow-id>

# Monitor steps in real-time
go run . --url http://localhost:10000 monitor <workflow-id>

# List recent workflows
go run . --url http://localhost:10000 list

# Search
go run . --url http://localhost:10000 search "notifications"

# Lifecycle
go run . --url http://localhost:10000 cancel <workflow-id> "no longer needed"
go run . --url http://localhost:10000 pause <workflow-id>
go run . --url http://localhost:10000 resume <workflow-id>
go run . --url http://localhost:10000 signal <workflow-id> approval-granted

# Validate without executing
go run . --url http://localhost:10000 validate example.yaml
```
