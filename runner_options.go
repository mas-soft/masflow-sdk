package masflowsdk

import (
	"log/slog"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

// RunnerOption configures a Runner.
type RunnerOption func(*runnerConfig)

type runnerConfig struct {
	temporalAddress   string
	temporalNamespace string
	temporalClient    client.Client // nil = create one from address/namespace
	platformURL       string        // empty = skip platform registration
	logger            *slog.Logger
	shutdownTimeout   time.Duration
	workerOptions     worker.Options
	httpClient        *http.Client
	connectOptions    []connect.ClientOption
}

func defaultConfig() *runnerConfig {
	return &runnerConfig{
		temporalAddress:   "localhost:7233",
		temporalNamespace: "default",
		shutdownTimeout:   30 * time.Second,
		logger:            slog.Default(),
		httpClient:        http.DefaultClient,
	}
}

// WithTemporalAddress sets the Temporal server address (default: "localhost:7233").
func WithTemporalAddress(addr string) RunnerOption {
	return func(c *runnerConfig) { c.temporalAddress = addr }
}

// WithTemporalNamespace sets the Temporal namespace (default: "default").
func WithTemporalNamespace(ns string) RunnerOption {
	return func(c *runnerConfig) { c.temporalNamespace = ns }
}

// WithTemporalClient provides a pre-configured Temporal client.
// When set, WithTemporalAddress and WithTemporalNamespace are ignored.
func WithTemporalClient(tc client.Client) RunnerOption {
	return func(c *runnerConfig) { c.temporalClient = tc }
}

// WithPlatformURL sets the masflow platform URL for module registration.
// If not set, the module will not be registered with the platform
// (useful for development or when running alongside the server in-process).
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

// WithHTTPClient sets the HTTP client used for platform registration.
func WithHTTPClient(hc *http.Client) RunnerOption {
	return func(c *runnerConfig) { c.httpClient = hc }
}

// WithConnectOptions adds Connect client options for platform registration.
func WithConnectOptions(opts ...connect.ClientOption) RunnerOption {
	return func(c *runnerConfig) { c.connectOptions = append(c.connectOptions, opts...) }
}
