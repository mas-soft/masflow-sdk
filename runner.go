package masflowsdk

import (
	"context"
	"fmt"
	"log/slog"
	"os/signal"
	"syscall"

	"github.com/mas-soft/masflow/sdk/platform"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// Runner manages the lifecycle of a masflow module:
// Temporal client connection, worker start, platform registration, and graceful shutdown.
type Runner struct {
	module         *Module
	config         *runnerConfig
	temporalClient client.Client
	worker         worker.Worker
	platformClient *platform.Client
	ownsClient     bool // true if we created the Temporal client (and must close it)
	registered     bool // true if we registered with the platform
	logger         *slog.Logger
}

// NewRunner creates a Runner for the given module.
func NewRunner(m *Module, opts ...RunnerOption) (*Runner, error) {
	if m == nil {
		return nil, fmt.Errorf("module is required")
	}
	if m.Name == "" {
		return nil, fmt.Errorf("module name is required")
	}
	if m.TaskQueue == "" {
		return nil, fmt.Errorf("module task queue is required (use WithModuleTaskQueue)")
	}

	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	return &Runner{
		module: m,
		config: cfg,
		logger: cfg.logger,
	}, nil
}

// Run starts the worker, registers with the platform (if configured),
// and blocks until ctx is cancelled or a termination signal (SIGINT/SIGTERM) is received.
func (r *Runner) Run(ctx context.Context) error {
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := r.Start(ctx); err != nil {
		return err
	}

	<-ctx.Done()
	r.logger.Info("Shutdown signal received, stopping...")

	stopCtx, stopCancel := context.WithTimeout(context.Background(), r.config.shutdownTimeout)
	defer stopCancel()

	return r.Stop(stopCtx)
}

// Start starts the Temporal worker and registers with the platform.
// Call Stop() to shut down. For a blocking version, use Run().
func (r *Runner) Start(ctx context.Context) error {
	// 1. Create or use provided Temporal client
	if r.config.temporalClient != nil {
		r.temporalClient = r.config.temporalClient
		r.ownsClient = false
	} else {
		tc, err := client.Dial(client.Options{
			HostPort:  r.config.temporalAddress,
			Namespace: r.config.temporalNamespace,
		})
		if err != nil {
			return fmt.Errorf("failed to connect to Temporal at %s: %w", r.config.temporalAddress, err)
		}
		r.temporalClient = tc
		r.ownsClient = true
		r.logger.Info("Connected to Temporal",
			"address", r.config.temporalAddress,
			"namespace", r.config.temporalNamespace)
	}

	// 2. Create and start the Temporal worker
	r.worker = worker.New(r.temporalClient, r.module.TaskQueue, r.config.workerOptions)
	RegisterAll(r.worker, r.module)

	if err := r.worker.Start(); err != nil {
		if r.ownsClient {
			r.temporalClient.Close()
		}
		return fmt.Errorf("failed to start Temporal worker: %w", err)
	}
	r.logger.Info("Temporal worker started",
		"module", r.module.Name,
		"task_queue", r.module.TaskQueue,
		"activities", len(r.module.activities))

	// 3. Register with platform (if configured)
	if r.config.platformURL != "" {
		r.platformClient = platform.NewClient(
			r.config.platformURL,
			platform.WithHTTPClient(r.config.httpClient),
			platform.WithConnectOptions(r.config.connectOptions...),
		)

		resp, err := r.platformClient.RegisterModule(ctx, r.module.toProto())
		if err != nil {
			r.logger.Warn("Failed to register with platform (module will still process activities)",
				"error", err,
				"platform_url", r.config.platformURL)
		} else {
			r.registered = true
			r.logger.Info("Registered with masflow platform",
				"module", resp.GetModuleName(),
				"activities", resp.GetRegisteredActivities(),
				"platform_url", r.config.platformURL)
		}
	}

	return nil
}

// Stop gracefully shuts down the worker and unregisters from the platform.
func (r *Runner) Stop(ctx context.Context) error {
	var errs []error

	// Unregister from platform
	if r.registered && r.platformClient != nil {
		if err := r.platformClient.UnregisterModule(ctx, r.module.Name); err != nil {
			r.logger.Warn("Failed to unregister from platform", "error", err)
			errs = append(errs, fmt.Errorf("unregister: %w", err))
		} else {
			r.logger.Info("Unregistered from masflow platform", "module", r.module.Name)
		}
		r.registered = false
	}

	// Stop worker
	if r.worker != nil {
		r.worker.Stop()
		r.logger.Info("Temporal worker stopped")
	}

	// Close Temporal client if we own it
	if r.ownsClient && r.temporalClient != nil {
		r.temporalClient.Close()
		r.logger.Info("Temporal client closed")
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}
