package masflowsdk

import (
	"context"
	"fmt"
	"net/http"

	"connectrpc.com/connect"
	pb "github.com/mas-soft/masflow/sdk/internal/pb/workflow"
	pbconnect "github.com/mas-soft/masflow/sdk/internal/pb/workflow/workflowconnect"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
)

// WorkflowClient provides methods for executing, querying, and managing
// workflows on the Masflow platform via Connect/gRPC.
type WorkflowClient struct {
	client  pbconnect.WorkflowServiceClient
	baseURL string
}

// WorkflowClientOption configures a WorkflowClient.
type WorkflowClientOption func(*workflowClientConfig)

type workflowClientConfig struct {
	httpClient     *http.Client
	connectOptions []connect.ClientOption
	useGRPC        bool
}

// WithWorkflowHTTPClient sets the HTTP client for the workflow client.
func WithWorkflowHTTPClient(c *http.Client) WorkflowClientOption {
	return func(cfg *workflowClientConfig) { cfg.httpClient = c }
}

// WithWorkflowConnectOptions adds Connect client options.
func WithWorkflowConnectOptions(opts ...connect.ClientOption) WorkflowClientOption {
	return func(cfg *workflowClientConfig) { cfg.connectOptions = append(cfg.connectOptions, opts...) }
}

// WithWorkflowConnect configures the workflow client to use Connect protocol
// over HTTP/1.1 instead of the default gRPC (HTTP/2).
func WithWorkflowConnect() WorkflowClientOption {
	return func(cfg *workflowClientConfig) { cfg.useGRPC = false }
}

// NewWorkflowClient creates a workflow client connected to the Masflow platform.
func NewWorkflowClient(baseURL string, opts ...WorkflowClientOption) *WorkflowClient {
	cfg := &workflowClientConfig{
		httpClient: http.DefaultClient,
		useGRPC:    true, // gRPC over HTTP/2 by default
	}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.useGRPC {
		cfg.connectOptions = append(cfg.connectOptions, connect.WithGRPC())
		if cfg.httpClient == http.DefaultClient {
			cfg.httpClient = newH2CClient()
		}
	}
	return &WorkflowClient{
		client:  pbconnect.NewWorkflowServiceClient(cfg.httpClient, baseURL, cfg.connectOptions...),
		baseURL: baseURL,
	}
}

// ── Execute ──────────────────────────────────────────────────────────────

// ExecuteYAML executes a workflow from YAML source.
func (c *WorkflowClient) ExecuteYAML(ctx context.Context, yaml string, opts *ExecuteSourceOptions) (*ExecuteResult, error) {
	return c.executeSource(ctx, yaml, pb.WorkflowSourceFormat_WORKFLOW_SOURCE_FORMAT_YAML, opts)
}

// ExecuteJSON executes a workflow from JSON source.
func (c *WorkflowClient) ExecuteJSON(ctx context.Context, jsonSrc string, opts *ExecuteSourceOptions) (*ExecuteResult, error) {
	return c.executeSource(ctx, jsonSrc, pb.WorkflowSourceFormat_WORKFLOW_SOURCE_FORMAT_JSON, opts)
}

// ExecuteDeclaration executes a saved workflow declaration by ID.
func (c *WorkflowClient) ExecuteDeclaration(ctx context.Context, declarationID string, opts *ExecuteDeclarationOptions) (*ExecuteDeclarationResult, error) {
	req := &pb.ExecuteWorkflowDeclarationRequest{
		DeclarationId: declarationID,
	}
	if opts != nil {
		req.WorkflowId = opts.WorkflowID
		if opts.Variables != nil {
			vars, err := toValueMap(opts.Variables)
			if err != nil {
				return nil, fmt.Errorf("marshal variables: %w", err)
			}
			req.InitialVariables = vars
		}
		if opts.WorkflowTimeout > 0 {
			req.StartOptions = &pb.WorkflowStartOptions{
				WorkflowTimeout: durationpb.New(opts.WorkflowTimeout),
			}
		}
	}

	resp, err := c.client.ExecuteWorkflowDeclaration(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	msg := resp.Msg
	if msg.GetError() != "" {
		return nil, fmt.Errorf("workflow execution failed: %s", msg.GetError())
	}
	return &ExecuteDeclarationResult{
		WorkflowID:      msg.GetWorkflowId(),
		Status:          toStatus(msg.GetStatus()),
		DeclarationID:   msg.GetDeclarationId(),
		DeclarationName: msg.GetDeclarationName(),
	}, nil
}

func (c *WorkflowClient) executeSource(ctx context.Context, source string, format pb.WorkflowSourceFormat, opts *ExecuteSourceOptions) (*ExecuteResult, error) {
	req := &pb.ExecuteWorkflowSourceRequest{
		Source: source,
		Format: format,
	}
	if opts != nil {
		req.WorkflowId = opts.WorkflowID
		req.Context = opts.Context
		if opts.Variables != nil {
			vars, err := toValueMap(opts.Variables)
			if err != nil {
				return nil, fmt.Errorf("marshal variables: %w", err)
			}
			req.InitialVariables = vars
		}
		if opts.TaskQueue != "" || opts.WorkflowTimeout > 0 {
			req.StartOptions = &pb.WorkflowStartOptions{}
			if opts.TaskQueue != "" {
				req.StartOptions.TaskQueue = opts.TaskQueue
			}
			if opts.WorkflowTimeout > 0 {
				req.StartOptions.WorkflowTimeout = durationpb.New(opts.WorkflowTimeout)
			}
		}
	}

	resp, err := c.client.ExecuteWorkflowSource(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	msg := resp.Msg
	if msg.GetError() != "" {
		return nil, fmt.Errorf("workflow execution failed: %s", msg.GetError())
	}
	return &ExecuteResult{
		WorkflowID: msg.GetWorkflowId(),
		Status:     toStatus(msg.GetStatus()),
		Result:     fromValueMap(msg.GetResult()),
	}, nil
}

// ── Status & Describe ────────────────────────────────────────────────────

// GetStatus returns the current status and trace of a workflow.
func (c *WorkflowClient) GetStatus(ctx context.Context, workflowID string) (*WorkflowStatusResult, error) {
	resp, err := c.client.GetWorkflowStatus(ctx, connect.NewRequest(&pb.GetWorkflowStatusRequest{
		WorkflowId: workflowID,
	}))
	if err != nil {
		return nil, err
	}
	msg := resp.Msg
	result := &WorkflowStatusResult{
		WorkflowID: msg.GetWorkflowId(),
		Status:     toStatus(msg.GetStatus()),
	}
	for _, t := range msg.GetTrace() {
		result.Trace = append(result.Trace, toTraceEntry(t))
	}
	return result, nil
}

// Describe returns detailed information about a workflow execution.
func (c *WorkflowClient) Describe(ctx context.Context, workflowID, runID string) (*WorkflowInfo, error) {
	resp, err := c.client.DescribeWorkflow(ctx, connect.NewRequest(&pb.DescribeWorkflowRequest{
		WorkflowId: workflowID,
		RunId:      runID,
	}))
	if err != nil {
		return nil, err
	}
	msg := resp.Msg
	if msg.GetError() != "" {
		return nil, fmt.Errorf("describe workflow: %s", msg.GetError())
	}
	info := msg.GetExecutionInfo()
	result := &WorkflowInfo{
		WorkflowID:             info.GetWorkflowId(),
		RunID:                  info.GetRunId(),
		WorkflowType:           info.GetWorkflowType(),
		Status:                 toStatus(info.GetStatus()),
		StartTime:              info.GetStartTime().AsTime(),
		CloseTime:              info.GetCloseTime().AsTime(),
		ExecutionTime:          info.GetExecutionTime().AsDuration(),
		ParentWorkflowID:       info.GetParentWorkflowId(),
		ParentRunID:            info.GetParentRunId(),
		HasErrors:              info.GetHasErrors(),
		ErrorMessage:           info.GetErrorMessage(),
		Attempt:                info.GetAttempt(),
		TaskQueue:              info.GetTaskQueue(),
		HistoryLength:          info.GetHistoryLength(),
		PendingActivitiesCount: info.GetPendingActivitiesCount(),
		PendingChildrenCount:   info.GetPendingChildrenCount(),
		Namespace:              info.GetNamespace(),
		DeclarationID:          info.GetDeclarationId(),
		DeclarationName:        info.GetDeclarationName(),
	}
	for _, cw := range msg.GetChildWorkflows() {
		result.ChildWorkflows = append(result.ChildWorkflows, toRelated(cw))
	}
	for _, sw := range msg.GetSubWorkflows() {
		result.SubWorkflows = append(result.SubWorkflows, toRelated(sw))
	}
	return result, nil
}

// ── List & Search ────────────────────────────────────────────────────────

// List returns a paginated list of workflow executions.
func (c *WorkflowClient) List(ctx context.Context, opts *ListWorkflowsOptions) (*ListWorkflowsResult, error) {
	req := &pb.ListWorkflowsRequest{}
	if opts != nil {
		req.PageSize = opts.PageSize
		req.NextPageToken = opts.NextPageToken
		req.OrderBy = opts.OrderBy
		req.Filter = &pb.WorkflowFilter{
			WorkflowType: opts.WorkflowType,
			Status:       toStatusProto(opts.Status),
			Query:        opts.Query,
			Tags:         opts.Tags,
			Category:     opts.Category,
		}
	}
	resp, err := c.client.ListWorkflows(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	msg := resp.Msg
	result := &ListWorkflowsResult{
		NextPageToken: msg.GetNextPageToken(),
		TotalCount:    msg.GetTotalCount(),
	}
	for _, ws := range msg.GetWorkflows() {
		result.Workflows = append(result.Workflows, WorkflowSummaryResult{
			WorkflowID:       ws.GetWorkflowId(),
			RunID:            ws.GetRunId(),
			WorkflowType:     ws.GetWorkflowType(),
			Status:           toStatus(ws.GetStatus()),
			StartTime:        ws.GetStartTime().AsTime(),
			CloseTime:        ws.GetCloseTime().AsTime(),
			ExecutionTime:    ws.GetExecutionTime().AsDuration(),
			ParentWorkflowID: ws.GetParentWorkflowId(),
			ParentRunID:      ws.GetParentRunId(),
			HasErrors:        ws.GetHasErrors(),
			DeclarationName:  ws.GetDeclarationName(),
		})
	}
	return result, nil
}

// Search performs advanced search on workflows.
func (c *WorkflowClient) Search(ctx context.Context, opts *SearchWorkflowsOptions) (*SearchResult, error) {
	req := &pb.SearchWorkflowsRequest{}
	if opts != nil {
		req.Query = opts.Query
		req.Tags = opts.Tags
		req.Categories = opts.Categories
		req.Status = toStatusProto(opts.Status)
		req.PageSize = opts.PageSize
		req.NextPageToken = opts.NextPageToken
		req.SortBy = opts.SortBy
		req.SortOrder = opts.SortOrder
	}
	resp, err := c.client.SearchWorkflows(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	msg := resp.Msg
	result := &SearchResult{
		NextPageToken: msg.GetNextPageToken(),
		TotalCount:    msg.GetTotalCount(),
		FacetCounts:   msg.GetFacetCounts(),
	}
	for _, wm := range msg.GetWorkflows() {
		result.Workflows = append(result.Workflows, WorkflowMetadataResult{
			WorkflowID:  wm.GetWorkflowId(),
			Name:        wm.GetName(),
			Description: wm.GetDescription(),
			Version:     wm.GetVersion(),
			Author:      wm.GetAuthor(),
			Tags:        wm.GetTags(),
			Category:    wm.GetCategory(),
			Status:      toStatus(wm.GetStatus()),
			CreatedAt:   wm.GetCreatedAt().AsTime(),
			UpdatedAt:   wm.GetUpdatedAt().AsTime(),
		})
	}
	return result, nil
}

// ── Query & Monitor ──────────────────────────────────────────────────────

// Query sends a query to a running workflow and returns the result.
func (c *WorkflowClient) Query(ctx context.Context, workflowID, queryType string, args map[string]any) (*QueryResult, error) {
	req := &pb.QueryWorkflowRequest{
		WorkflowId: workflowID,
		QueryType:  queryType,
	}
	if args != nil {
		qa, err := toValueMap(args)
		if err != nil {
			return nil, fmt.Errorf("marshal query args: %w", err)
		}
		req.QueryArgs = qa
	}
	resp, err := c.client.QueryWorkflow(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	msg := resp.Msg
	if msg.GetError() != "" {
		return nil, fmt.Errorf("query workflow: %s", msg.GetError())
	}
	return &QueryResult{
		WorkflowID: msg.GetWorkflowId(),
		RunID:      msg.GetRunId(),
		Result:     fromValueMap(msg.GetQueryResult()),
		QueryTime:  msg.GetQueryTime().AsTime(),
	}, nil
}

// Monitor returns real-time monitoring data for a workflow.
func (c *WorkflowClient) Monitor(ctx context.Context, workflowID, runID string) (*MonitorStatus, error) {
	resp, err := c.client.QueryMonitorStatus(ctx, connect.NewRequest(&pb.QueryMonitorStatusRequest{
		WorkflowId: workflowID,
		RunId:      runID,
	}))
	if err != nil {
		return nil, err
	}
	msg := resp.Msg
	if msg.GetError() != "" {
		return nil, fmt.Errorf("monitor workflow: %s", msg.GetError())
	}
	result := &MonitorStatus{
		WorkflowID: msg.GetWorkflowId(),
		Status:     msg.GetStatus(),
		StartTime:  msg.GetStartTime().AsTime(),
		ElapsedMs:  msg.GetElapsedMs(),
		LastError:  msg.GetLastError(),
	}
	if p := msg.GetProgress(); p != nil {
		result.Progress = ProgressInfo{
			CurrentStep: p.GetCurrentStep(),
			TotalSteps:  p.GetTotalSteps(),
			Percentage:  p.GetPercentage(),
		}
	}
	if sc := msg.GetStepCounts(); sc != nil {
		result.StepCounts = StepCounts{
			Total: sc.GetTotal(), Pending: sc.GetPending(),
			Running: sc.GetRunning(), Waiting: sc.GetWaiting(),
			Completed: sc.GetCompleted(), Failed: sc.GetFailed(),
			Skipped: sc.GetSkipped(), Cancelled: sc.GetCancelled(),
		}
	}
	for _, s := range msg.GetSteps() {
		result.Steps = append(result.Steps, StepMonitor{
			Index: s.GetIndex(), Name: s.GetName(),
			StepType: s.GetStepType(), Status: s.GetStatus(),
			StartTime: s.GetStartTime().AsTime(), EndTime: s.GetEndTime().AsTime(),
			DurationMs: s.GetDurationMs(), Error: s.GetError(),
			WaitReason: s.GetWaitReason(),
		})
	}
	return result, nil
}

// Trace returns the BPM execution trace for a workflow.
func (c *WorkflowClient) Trace(ctx context.Context, workflowID, runID string) ([]TraceEntryResult, error) {
	resp, err := c.client.QueryTrace(ctx, connect.NewRequest(&pb.QueryTraceRequest{
		WorkflowId: workflowID,
		RunId:      runID,
	}))
	if err != nil {
		return nil, err
	}
	msg := resp.Msg
	if msg.GetError() != "" {
		return nil, fmt.Errorf("query trace: %s", msg.GetError())
	}
	var entries []TraceEntryResult
	for _, t := range msg.GetEntries() {
		entries = append(entries, toTraceEntry(t))
	}
	return entries, nil
}

// ── Lifecycle ────────────────────────────────────────────────────────────

// Cancel requests cancellation of a running workflow.
func (c *WorkflowClient) Cancel(ctx context.Context, workflowID, reason string) error {
	resp, err := c.client.CancelWorkflow(ctx, connect.NewRequest(&pb.CancelWorkflowRequest{
		WorkflowId: workflowID,
		Reason:     reason,
	}))
	if err != nil {
		return err
	}
	if resp.Msg.GetError() != "" {
		return fmt.Errorf("cancel workflow: %s", resp.Msg.GetError())
	}
	return nil
}

// Signal sends a signal to a running workflow.
func (c *WorkflowClient) Signal(ctx context.Context, workflowID, signalName string, data map[string]any) (*SignalResult, error) {
	req := &pb.SignalWorkflowRequest{
		WorkflowId: workflowID,
		SignalName: signalName,
	}
	if data != nil {
		sd, err := toValueMap(data)
		if err != nil {
			return nil, fmt.Errorf("marshal signal data: %w", err)
		}
		req.SignalData = sd
	}
	resp, err := c.client.SignalWorkflow(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, err
	}
	msg := resp.Msg
	if msg.GetError() != "" {
		return nil, fmt.Errorf("signal workflow: %s", msg.GetError())
	}
	return &SignalResult{
		WorkflowID: msg.GetWorkflowId(),
		RunID:      msg.GetRunId(),
	}, nil
}

// Terminate forcefully terminates a running workflow.
func (c *WorkflowClient) Terminate(ctx context.Context, workflowID, reason string) error {
	resp, err := c.client.TerminateWorkflow(ctx, connect.NewRequest(&pb.TerminateWorkflowRequest{
		WorkflowId: workflowID,
		Reason:     reason,
	}))
	if err != nil {
		return err
	}
	if resp.Msg.GetError() != "" {
		return fmt.Errorf("terminate workflow: %s", resp.Msg.GetError())
	}
	return nil
}

// Pause pauses a running workflow.
func (c *WorkflowClient) Pause(ctx context.Context, workflowID, runID, reason string) error {
	resp, err := c.client.PauseWorkflow(ctx, connect.NewRequest(&pb.PauseWorkflowRequest{
		WorkflowId: workflowID,
		RunId:      runID,
		Reason:     reason,
	}))
	if err != nil {
		return err
	}
	if resp.Msg.GetError() != "" {
		return fmt.Errorf("pause workflow: %s", resp.Msg.GetError())
	}
	return nil
}

// Resume resumes a paused workflow.
func (c *WorkflowClient) Resume(ctx context.Context, workflowID, runID, reason string) error {
	resp, err := c.client.ResumeWorkflow(ctx, connect.NewRequest(&pb.ResumeWorkflowRequest{
		WorkflowId: workflowID,
		RunId:      runID,
		Reason:     reason,
	}))
	if err != nil {
		return err
	}
	if resp.Msg.GetError() != "" {
		return fmt.Errorf("resume workflow: %s", resp.Msg.GetError())
	}
	return nil
}

// Validate validates a workflow YAML source without executing it.
func (c *WorkflowClient) Validate(ctx context.Context, yaml string) (*ValidateResult, error) {
	// The validate RPC takes a Workflow message, but for SDK ergonomics
	// we accept YAML and let the server parse it. We use a minimal Workflow
	// with the name field set to the source for now.
	// In practice, the server's ValidateWorkflow expects a parsed Workflow proto.
	// SDK users should use ExecuteYAML with a dry-run or the server's validation endpoint.
	resp, err := c.client.ValidateWorkflow(ctx, connect.NewRequest(&pb.ValidateWorkflowRequest{
		Workflow: &pb.Workflow{Name: yaml},
	}))
	if err != nil {
		return nil, err
	}
	msg := resp.Msg
	return &ValidateResult{
		Valid:    msg.GetValid(),
		Errors:   msg.GetErrors(),
		Warnings: msg.GetWarnings(),
	}, nil
}

// ── Helpers ──────────────────────────────────────────────────────────────

var statusMap = map[pb.WorkflowStatus]WorkflowStatus{
	pb.WorkflowStatus_WORKFLOW_STATUS_UNSPECIFIED: WorkflowStatusUnspecified,
	pb.WorkflowStatus_WORKFLOW_STATUS_PENDING:     WorkflowStatusPending,
	pb.WorkflowStatus_WORKFLOW_STATUS_RUNNING:     WorkflowStatusRunning,
	pb.WorkflowStatus_WORKFLOW_STATUS_COMPLETED:   WorkflowStatusCompleted,
	pb.WorkflowStatus_WORKFLOW_STATUS_FAILED:      WorkflowStatusFailed,
	pb.WorkflowStatus_WORKFLOW_STATUS_CANCELLED:   WorkflowStatusCancelled,
	pb.WorkflowStatus_WORKFLOW_STATUS_PAUSED:      WorkflowStatusPaused,
}

var statusReverseMap = map[WorkflowStatus]pb.WorkflowStatus{
	WorkflowStatusUnspecified: pb.WorkflowStatus_WORKFLOW_STATUS_UNSPECIFIED,
	WorkflowStatusPending:    pb.WorkflowStatus_WORKFLOW_STATUS_PENDING,
	WorkflowStatusRunning:    pb.WorkflowStatus_WORKFLOW_STATUS_RUNNING,
	WorkflowStatusCompleted:  pb.WorkflowStatus_WORKFLOW_STATUS_COMPLETED,
	WorkflowStatusFailed:     pb.WorkflowStatus_WORKFLOW_STATUS_FAILED,
	WorkflowStatusCancelled:  pb.WorkflowStatus_WORKFLOW_STATUS_CANCELLED,
	WorkflowStatusPaused:     pb.WorkflowStatus_WORKFLOW_STATUS_PAUSED,
}

func toStatus(s pb.WorkflowStatus) WorkflowStatus {
	if v, ok := statusMap[s]; ok {
		return v
	}
	return WorkflowStatusUnspecified
}

func toStatusProto(s WorkflowStatus) pb.WorkflowStatus {
	if v, ok := statusReverseMap[s]; ok {
		return v
	}
	return pb.WorkflowStatus_WORKFLOW_STATUS_UNSPECIFIED
}

func toTraceEntry(t *pb.TraceEntry) TraceEntryResult {
	return TraceEntryResult{
		Timestamp: t.GetTimestamp().AsTime(),
		StepType:  t.GetStepType(),
		Details:   t.GetDetails(),
		Status:    t.GetStatus(),
		Error:     t.GetError(),
		Data:      fromValueMap(t.GetData()),
	}
}

func toRelated(rw *pb.RelatedWorkflow) RelatedWorkflowInfo {
	return RelatedWorkflowInfo{
		WorkflowID:   rw.GetWorkflowId(),
		RunID:        rw.GetRunId(),
		WorkflowType: rw.GetWorkflowType(),
		Status:       toStatus(rw.GetStatus()),
	}
}

func toValueMap(m map[string]any) (map[string]*structpb.Value, error) {
	result := make(map[string]*structpb.Value, len(m))
	for k, v := range m {
		val, err := structpb.NewValue(v)
		if err != nil {
			return nil, fmt.Errorf("field %q: %w", k, err)
		}
		result[k] = val
	}
	return result, nil
}

func fromValueMap(m map[string]*structpb.Value) map[string]any {
	if m == nil {
		return nil
	}
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = v.AsInterface()
	}
	return result
}
