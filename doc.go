// Package masflowsdk provides a standalone Go SDK for building third-party
// activity modules that register with the Masflow workflow engine platform.
//
// The SDK decouples module development from the Masflow server codebase.
// It provides type-safe activity registration with Go generics, automatic
// JSON Schema generation, Temporal worker lifecycle management, and
// platform registration via Connect/gRPC.
//
// # Quick Start
//
// Define a module, register activities, and run:
//
//	mod := masflowsdk.NewModule("my-module",
//	    masflowsdk.WithModuleTaskQueue("my-task-queue"),
//	    masflowsdk.WithModuleVersion("1.0.0"),
//	)
//
//	masflowsdk.Register(mod, "myActivity", MyHandler,
//	    masflowsdk.WithDescription("Does something useful"),
//	)
//
//	runner, _ := masflowsdk.NewRunner(mod,
//	    masflowsdk.WithPlatformURL("http://localhost:9999"),
//	)
//	runner.Run(context.Background())
//
// # Handler Types
//
// Three handler signatures are supported:
//
//   - [Handler] -- sync activities: func(ctx, TReq) (TRes, error)
//   - [VoidHandler] -- side-effect-only: func(ctx, TReq) error
//   - [AsyncHandler] -- long-running with callback: func(ctx, TReq, *AsyncCallbackInfo) (TRes, error)
//
// # Registration
//
// Activities are registered on a [Module] using the generic registration functions:
//
//   - [Register] -- for sync handlers
//   - [RegisterVoid] -- for void handlers
//   - [RegisterAsync] -- for async handlers
//
// Each registration auto-generates JSON Schema from the Go input/output types
// and infers type URLs for the activity contract.
//
// # Running
//
// The [Runner] manages the full module lifecycle:
//
//  1. Registers the module with the Masflow platform
//  2. Receives Temporal address and namespace from the platform
//  3. Connects to Temporal and starts a worker on the module's task queue
//  4. Handles graceful shutdown on SIGINT/SIGTERM
//
// Third-party modules do not configure Temporal address or namespace directly.
// The platform is the source of truth and provides these during registration.
package masflowsdk
