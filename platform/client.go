package platform

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	pb "github.com/mas-soft/masflow-sdk/internal/pb/activity"
	pbconnect "github.com/mas-soft/masflow-sdk/internal/pb/activity/activityconnect"
)

// Client communicates with the masflow platform's ModuleRegistry service.
type Client struct {
	registry pbconnect.ModuleRegistryClient
	baseURL  string
}

// ClientOption configures a Client.
type ClientOption func(*clientConfig)

type clientConfig struct {
	httpClient     *http.Client
	connectOptions []connect.ClientOption
}

// WithHTTPClient sets the HTTP client used for Connect transport.
func WithHTTPClient(c *http.Client) ClientOption {
	return func(cfg *clientConfig) { cfg.httpClient = c }
}

// WithConnectOptions adds Connect client options.
func WithConnectOptions(opts ...connect.ClientOption) ClientOption {
	return func(cfg *clientConfig) { cfg.connectOptions = append(cfg.connectOptions, opts...) }
}

// NewClient creates a platform registration client.
func NewClient(baseURL string, opts ...ClientOption) *Client {
	cfg := &clientConfig{
		httpClient: http.DefaultClient,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	return &Client{
		registry: pbconnect.NewModuleRegistryClient(cfg.httpClient, baseURL, cfg.connectOptions...),
		baseURL:  baseURL,
	}
}

// RegisterModule registers a module with the masflow platform.
func (c *Client) RegisterModule(ctx context.Context, mod *pb.Module) (*pb.RegisterModuleResponse, error) {
	resp, err := c.registry.RegisterModule(ctx, connect.NewRequest(&pb.RegisterModuleRequest{
		Module: mod,
	}))
	if err != nil {
		return nil, err
	}
	return resp.Msg, nil
}

// UnregisterModule removes a module from the masflow platform.
func (c *Client) UnregisterModule(ctx context.Context, moduleName string) error {
	_, err := c.registry.UnregisterModule(ctx, connect.NewRequest(&pb.UnregisterModuleRequest{
		ModuleName: moduleName,
	}))
	return err
}

// ListModules returns all registered modules from the masflow platform.
func (c *Client) ListModules(ctx context.Context) ([]*pb.Module, error) {
	resp, err := c.registry.ListModules(ctx, connect.NewRequest(&pb.ListModulesRequest{}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetModules(), nil
}

// GetModule returns a specific module from the masflow platform.
func (c *Client) GetModule(ctx context.Context, moduleName string) (*pb.Module, error) {
	resp, err := c.registry.GetModule(ctx, connect.NewRequest(&pb.GetModuleRequest{
		ModuleName: moduleName,
	}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetModule(), nil
}
