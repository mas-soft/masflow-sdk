package masflowsdk

import (
	"log/slog"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"go.temporal.io/sdk/worker"
)

// RunnerOption configures a Runner.
type RunnerOption func(*runnerConfig)

type runnerConfig struct {
	platformURL     string // required — the masflow platform URL
	workflowURL     string // empty = no WorkflowClient created
	logger          *slog.Logger
	shutdownTimeout time.Duration
	workerOptions   worker.Options
	httpClient      *http.Client
	connectOptions  []connect.ClientOption
	protocol        protocolMode
}

func defaultConfig() *runnerConfig {
	return &runnerConfig{
		shutdownTimeout: 30 * time.Second,
		logger:          slog.Default(),
		httpClient:      http.DefaultClient,
		protocol:        protocolAuto,
	}
}

// WithPlatformURL sets the masflow platform URL (required).
// The platform returns Temporal connection details during module registration.
func WithPlatformURL(url string) RunnerOption {
	return func(c *runnerConfig) { c.platformURL = url }
}

// WithLogger sets the logger for the runner.
func WithLogger(logger *slog.Logger) RunnerOption {
	return func(c *runnerConfig) { c.logger = logger }
}

// WithShutdownTimeout sets the graceful shutdown timeout (default: 30s).
func WithShutdownTimeout(d time.Duration) RunnerOption {
	return func(c *runnerConfig) { c.shutdownTimeout = d }
}

// WithWorkerOptions sets Temporal worker options.
func WithWorkerOptions(opts worker.Options) RunnerOption {
	return func(c *runnerConfig) { c.workerOptions = opts }
}

// WithHTTPClient sets the HTTP client used for platform and workflow communication.
func WithHTTPClient(hc *http.Client) RunnerOption {
	return func(c *runnerConfig) { c.httpClient = hc }
}

// WithConnectOptions adds Connect client options for platform communication.
func WithConnectOptions(opts ...connect.ClientOption) RunnerOption {
	return func(c *runnerConfig) { c.connectOptions = append(c.connectOptions, opts...) }
}

// WithWorkflowURL sets the masflow platform URL for the WorkflowClient.
// When set, Runner.Workflows() returns a WorkflowClient for executing
// and managing workflows. If not set, Runner.Workflows() returns nil.
func WithWorkflowURL(url string) RunnerOption {
	return func(c *runnerConfig) { c.workflowURL = url }
}

// WithConnect configures the runner to use Connect protocol over HTTP/1.1.
// By default, the runner chooses Connect for http:// URLs and gRPC for https:// URLs.
func WithConnect() RunnerOption {
	return func(c *runnerConfig) { c.protocol = protocolConnect }
}

// WithGRPC configures the runner to use gRPC transport.
// This is mainly useful for plaintext h2c endpoints where you want to force gRPC over HTTP/2.
func WithGRPC() RunnerOption {
	return func(c *runnerConfig) { c.protocol = protocolGRPC }
}
