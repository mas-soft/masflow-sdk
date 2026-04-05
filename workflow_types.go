package masflowsdk

import "time"

// WorkflowStatus represents workflow execution status.
type WorkflowStatus string

const (
	WorkflowStatusUnspecified WorkflowStatus = ""
	WorkflowStatusPending    WorkflowStatus = "PENDING"
	WorkflowStatusRunning    WorkflowStatus = "RUNNING"
	WorkflowStatusCompleted  WorkflowStatus = "COMPLETED"
	WorkflowStatusFailed     WorkflowStatus = "FAILED"
	WorkflowStatusCancelled  WorkflowStatus = "CANCELLED"
	WorkflowStatusPaused     WorkflowStatus = "PAUSED"
)

// ExecuteResult is returned after starting a workflow execution.
type ExecuteResult struct {
	WorkflowID string
	Status     WorkflowStatus
	Result     map[string]any
	Error      string
}

// ExecuteDeclarationResult is returned after executing a saved workflow declaration.
type ExecuteDeclarationResult struct {
	WorkflowID      string
	Status          WorkflowStatus
	DeclarationID   string
	DeclarationName string
	Error           string
}

// WorkflowStatusResult holds the current status of a workflow.
type WorkflowStatusResult struct {
	WorkflowID string
	Status     WorkflowStatus
	Trace      []TraceEntryResult
}

// TraceEntryResult represents a single BPM execution trace entry.
type TraceEntryResult struct {
	Timestamp time.Time
	StepType  string
	Details   string
	Status    string
	Error     string
	Data      map[string]any
}

// WorkflowInfo holds detailed information about a workflow execution.
type WorkflowInfo struct {
	WorkflowID             string
	RunID                  string
	WorkflowType           string
	Status                 WorkflowStatus
	StartTime              time.Time
	CloseTime              time.Time
	ExecutionTime          time.Duration
	ParentWorkflowID       string
	ParentRunID            string
	HasErrors              bool
	ErrorMessage           string
	Attempt                int64
	TaskQueue              string
	HistoryLength          int64
	PendingActivitiesCount int32
	PendingChildrenCount   int32
	Namespace              string
	DeclarationID          string
	DeclarationName        string
	ChildWorkflows         []RelatedWorkflowInfo
	SubWorkflows           []RelatedWorkflowInfo
}

// RelatedWorkflowInfo is a lightweight reference to a related workflow.
type RelatedWorkflowInfo struct {
	WorkflowID   string
	RunID        string
	WorkflowType string
	Status       WorkflowStatus
}

// WorkflowSummaryResult holds a summary of a workflow for listing.
type WorkflowSummaryResult struct {
	WorkflowID       string
	RunID            string
	WorkflowType     string
	Status           WorkflowStatus
	StartTime        time.Time
	CloseTime        time.Time
	ExecutionTime    time.Duration
	ParentWorkflowID string
	ParentRunID      string
	HasErrors        bool
	DeclarationName  string
}

// ListWorkflowsResult holds a page of workflow summaries.
type ListWorkflowsResult struct {
	Workflows     []WorkflowSummaryResult
	NextPageToken string
	TotalCount    int32
}

// QueryResult holds the result of a workflow query.
type QueryResult struct {
	WorkflowID string
	RunID      string
	Result     map[string]any
	QueryTime  time.Time
}

// MonitorStatus holds real-time monitoring data for a workflow.
type MonitorStatus struct {
	WorkflowID string
	Status     string
	StartTime  time.Time
	ElapsedMs  int64
	Progress   ProgressInfo
	StepCounts StepCounts
	Steps      []StepMonitor
	LastError  string
}

// ProgressInfo describes how far execution has progressed.
type ProgressInfo struct {
	CurrentStep int32
	TotalSteps  int32
	Percentage  float64
}

// StepCounts holds aggregated step-status counts.
type StepCounts struct {
	Total     int32
	Pending   int32
	Running   int32
	Waiting   int32
	Completed int32
	Failed    int32
	Skipped   int32
	Cancelled int32
}

// StepMonitor holds real-time state for a single step.
type StepMonitor struct {
	Index      int32
	Name       string
	StepType   string
	Status     string
	StartTime  time.Time
	EndTime    time.Time
	DurationMs int64
	Error      string
	WaitReason string
}

// SearchResult holds a page of workflow metadata from search.
type SearchResult struct {
	Workflows     []WorkflowMetadataResult
	NextPageToken string
	TotalCount    int32
	FacetCounts   map[string]int32
}

// WorkflowMetadataResult holds metadata for a workflow.
type WorkflowMetadataResult struct {
	WorkflowID  string
	Name        string
	Description string
	Version     string
	Author      string
	Tags        []string
	Category    string
	Status      WorkflowStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// SignalResult holds the result of signaling a workflow.
type SignalResult struct {
	WorkflowID string
	RunID      string
}

// LifecycleResult holds the result of a lifecycle operation (cancel, terminate, pause, resume).
type LifecycleResult struct {
	WorkflowID string
	RunID      string
}

// ValidateResult holds the result of workflow validation.
type ValidateResult struct {
	Valid    bool
	Errors   []string
	Warnings []string
}

// ExecuteSourceOptions configures a workflow execution from YAML/JSON source.
type ExecuteSourceOptions struct {
	WorkflowID       string
	Variables        map[string]any
	TaskQueue        string
	WorkflowTimeout  time.Duration
	Context          map[string]string
}

// ExecuteDeclarationOptions configures execution of a saved declaration.
type ExecuteDeclarationOptions struct {
	WorkflowID      string
	Variables       map[string]any
	WorkflowTimeout time.Duration
}

// ListWorkflowsOptions configures a workflow listing request.
type ListWorkflowsOptions struct {
	WorkflowType  string
	Status        WorkflowStatus
	Query         string
	Tags          []string
	Category      string
	PageSize      int32
	NextPageToken string
	OrderBy       string
}

// SearchWorkflowsOptions configures a workflow search request.
type SearchWorkflowsOptions struct {
	Query         string
	Tags          []string
	Categories    []string
	Status        WorkflowStatus
	PageSize      int32
	NextPageToken string
	SortBy        string
	SortOrder     string
}
