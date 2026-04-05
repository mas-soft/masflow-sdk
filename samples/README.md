# Masflow SDK Samples

Standalone examples demonstrating the Masflow Go SDK. Each sample is an independent Go module you can build and run directly.

## Samples

| Sample | Difficulty | Description |
|--------|-----------|-------------|
| [basic](basic/) | Beginner | Minimal module with one activity -- the "hello world" of Masflow |
| [advanced](advanced/) | Intermediate | Full notifications module with 5 activities (email, SMS, Slack, webhook, log) |
| [async](async/) | Intermediate | Async activities with human-in-the-loop callback patterns |
| [workflow-client](workflow-client/) | Intermediate | CLI tool for executing, monitoring, and managing workflows |
| [multi-module](multi-module/) | Advanced | Multiple modules in one process with concurrent runners |

## Prerequisites

- Go 1.25+
- A running [Temporal](https://temporal.io) server (for activity worker samples)
- A running Masflow platform instance (optional, for platform registration and workflow execution)

## Quick Start

```bash
# Run the basic sample
cd basic
go run . --temporal=localhost:7233

# Run the advanced notifications module
cd advanced
go run . --temporal=localhost:7233 --platform=http://localhost:10000

# Execute a workflow via the CLI client
cd workflow-client
go run . --url http://localhost:10000 execute --yaml example.yaml

# Run async approval activities
cd async
go run . --temporal=localhost:7233

# Run two modules in one process
cd multi-module
go run . --temporal=localhost:7233
```

## SDK Features by Sample

| Feature | basic | advanced | async | workflow-client | multi-module |
|---------|:-----:|:--------:|:-----:|:---------------:|:------------:|
| `Register[TReq, TRes]` | x | x | x | | x |
| `RegisterVoid[TReq]` | | x | | | |
| `RegisterAsync[TReq, TRes]` | | | x | | |
| Module metadata | x | x | x | | x |
| Activity options | x | x | x | | x |
| `Runner` | x | x | x | | x |
| `WorkflowClient` | | | | x | |
| Multiple modules | | | | | x |
| Input validation | | x | x | | |
| Workflow YAML examples | x | x | x | x | x |
