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
//	    masflowsdk.WithTemporalAddress("localhost:7233"),
//	    masflowsdk.WithPlatformURL("http://localhost:10000"),
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
//  1. Connects to Temporal (or uses a provided client)
//  2. Creates and starts a Temporal worker on the module's task queue
//  3. Registers the module with the Masflow platform (optional)
//  4. Handles graceful shutdown on SIGINT/SIGTERM
//
// For manual worker setup, use [RegisterAll] to register all module activities
// with a Temporal worker directly.
package masflowsdk
