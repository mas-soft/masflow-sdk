# Basic Sample

The simplest possible Masflow module -- one activity, one handler, one runner.

## What it demonstrates

- Creating a `Module` with metadata
- Registering a typed sync activity with `Register[TReq, TRes]`
- Auto-generated JSON Schema from Go structs
- Running the worker with `Runner`

## Run

```bash
go run . --temporal=localhost:7233
```

With platform registration:

```bash
go run . --temporal=localhost:7233 --platform=http://localhost:10000
```

## Workflow YAML

```yaml
name: greeting-demo
steps:
  - name: say-hello
    activity:
      type: greet
      args:
        name: "World"
      ref: result
```
