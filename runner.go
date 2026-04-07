package masflowsdk

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"

	"connectrpc.com/connect"
	"github.com/mas-soft/masflow/sdk/platform"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// Runner manages the lifecycle of a masflow module:
// platform registration, Temporal client connection, worker start, and graceful shutdown.
//
// The flow is: register with platform → receive Temporal config → connect to Temporal → start worker.
// Third-party modules do not configure Temporal address or namespace directly —
// those values are provided by the masflow platform during registration.
type Runner struct {
	module         *Module
	config         *runnerConfig
	temporalClient client.Client
	worker         worker.Worker
	platformClient *platform.Client
	workflowClient *WorkflowClient
	registered     bool // true if we registered with the platform
	logger         *slog.Logger
}

// NewRunner creates a Runner for the given module.
// WithPlatformURL is required — the platform provides Temporal connection details.
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

	if cfg.platformURL == "" {
		return nil, fmt.Errorf("platform URL is required (use WithPlatformURL)")
	}

	useGRPC := shouldUseGRPC(cfg.platformURL, cfg.protocol)

	// When gRPC mode is enabled, configure h2c transport and gRPC connect option.
	if useGRPC {
		cfg.connectOptions = append(cfg.connectOptions, connect.WithGRPC())
		if usesPlaintextHTTP(cfg.platformURL) && (cfg.httpClient == nil || cfg.httpClient == http.DefaultClient) {
			cfg.httpClient = newH2CClient()
		}
	}

	r := &Runner{
		module: m,
		config: cfg,
		logger: cfg.logger,
	}

	// Create WorkflowClient eagerly if URL is configured.
	if cfg.workflowURL != "" {
		wcOpts := []WorkflowClientOption{
			WithWorkflowHTTPClient(cfg.httpClient),
			WithWorkflowConnectOptions(cfg.connectOptions...),
		}
		if !shouldUseGRPC(cfg.workflowURL, cfg.protocol) {
			wcOpts = append(wcOpts, WithWorkflowConnect())
		} else {
			wcOpts = append(wcOpts, WithWorkflowGRPC())
		}
		r.workflowClient = NewWorkflowClient(cfg.workflowURL, wcOpts...)
	}

	return r, nil
}

// Workflows returns the WorkflowClient for executing and managing workflows.
// Returns nil if WithWorkflowURL was not configured.
func (r *Runner) Workflows() *WorkflowClient {
	return r.workflowClient
}

// Run starts the worker, registers with the platform,
// and blocks until ctx is cancelled or a termination signal (SIGINT/SIGTERM) is received.
func (r *Runner) Run(ctx context.Context, overwriteTemporalAddress *string) error {
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := r.Start(ctx, overwriteTemporalAddress); err != nil {
		return err
	}

	<-ctx.Done()
	r.logger.Info("Shutdown signal received, stopping...")

	stopCtx, stopCancel := context.WithTimeout(context.Background(), r.config.shutdownTimeout)
	defer stopCancel()

	return r.Stop(stopCtx)
}

// Start registers with the platform to obtain Temporal config,
// connects to Temporal, and starts the worker.
// Call Stop() to shut down. For a blocking version, use Run().
func (r *Runner) Start(ctx context.Context, overwriteTemporalAddress *string) error {
	if r.temporalClient != nil || r.worker != nil || r.registered {
		return fmt.Errorf("runner already started")
	}

	// 1. Register with platform — get Temporal connection details
	r.platformClient = platform.NewClient(
		r.config.platformURL,
		platform.WithHTTPClient(r.config.httpClient),
		platform.WithConnectOptions(r.config.connectOptions...),
	)

	resp, err := r.platformClient.RegisterModule(ctx, r.module.toProto())
	if err != nil {
		return fmt.Errorf("failed to register with masflow platform at %s: %w", r.config.platformURL, err)
	}
	r.registered = true

	temporalAddr := resp.GetTemporalAddress()
	temporalNS := resp.GetTemporalNamespace()

	if overwriteTemporalAddress != nil && *overwriteTemporalAddress != "" {
		temporalAddr = *overwriteTemporalAddress
	}

	r.logger.Info("Registered with masflow platform",
		"module", resp.GetModuleName(),
		"activities", resp.GetRegisteredActivities(),
		"platform_url", r.config.platformURL,
		"temporal_address", temporalAddr,
		"temporal_namespace", temporalNS,
	)

	// 2. Connect to Temporal using platform-provided config
	tc, err := client.Dial(client.Options{
		HostPort:  temporalAddr,
		Namespace: temporalNS,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to Temporal at %s (namespace %s): %w", temporalAddr, temporalNS, err)
	}
	r.temporalClient = tc
	r.logger.Info("Connected to Temporal",
		"address", temporalAddr,
		"namespace", temporalNS)

	// 3. Create and start the Temporal worker
	r.worker = worker.New(r.temporalClient, r.module.TaskQueue, r.config.workerOptions)
	if err := RegisterAll(r.worker, r.module); err != nil {
		r.temporalClient.Close()
		r.temporalClient = nil
		r.worker = nil
		return fmt.Errorf("register module activities: %w", err)
	}

	if err := r.worker.Start(); err != nil {
		r.temporalClient.Close()
		r.temporalClient = nil
		r.worker = nil
		return fmt.Errorf("start Temporal worker: %w", err)
	}
	r.logger.Info("Temporal worker started",
		"module", r.module.Name,
		"task_queue", r.module.TaskQueue,
		"activities", len(r.module.activities))

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
		r.platformClient = nil
	}

	// Stop worker
	if r.worker != nil {
		if err := stopWorker(r.worker); err != nil {
			r.logger.Warn("Failed to stop Temporal worker cleanly", "error", err)
			errs = append(errs, fmt.Errorf("stop worker: %w", err))
		} else {
			r.logger.Info("Temporal worker stopped")
		}
		r.worker = nil
	}

	// Close Temporal client
	if r.temporalClient != nil {
		r.temporalClient.Close()
		r.logger.Info("Temporal client closed")
		r.temporalClient = nil
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func stopWorker(w worker.Worker) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("panic while stopping worker: %v", recovered)
		}
	}()

	w.Stop()
	return nil
}
