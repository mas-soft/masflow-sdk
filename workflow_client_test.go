package masflowsdk

import (
	"testing"

	pb "github.com/mas-soft/masflow/sdk/internal/pb/workflow"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestNewWorkflowClient(t *testing.T) {
	c := NewWorkflowClient("http://localhost:10000")
	if c == nil {
		t.Fatal("expected non-nil client")
	}
	if c.baseURL != "http://localhost:10000" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "http://localhost:10000")
	}
}

func TestToStatus(t *testing.T) {
	tests := []struct {
		proto  pb.WorkflowStatus
		expect WorkflowStatus
	}{
		{pb.WorkflowStatus_WORKFLOW_STATUS_UNSPECIFIED, WorkflowStatusUnspecified},
		{pb.WorkflowStatus_WORKFLOW_STATUS_PENDING, WorkflowStatusPending},
		{pb.WorkflowStatus_WORKFLOW_STATUS_RUNNING, WorkflowStatusRunning},
		{pb.WorkflowStatus_WORKFLOW_STATUS_COMPLETED, WorkflowStatusCompleted},
		{pb.WorkflowStatus_WORKFLOW_STATUS_FAILED, WorkflowStatusFailed},
		{pb.WorkflowStatus_WORKFLOW_STATUS_CANCELLED, WorkflowStatusCancelled},
		{pb.WorkflowStatus_WORKFLOW_STATUS_PAUSED, WorkflowStatusPaused},
		{pb.WorkflowStatus(999), WorkflowStatusUnspecified}, // unknown
	}
	for _, tt := range tests {
		got := toStatus(tt.proto)
		if got != tt.expect {
			t.Errorf("toStatus(%v) = %q, want %q", tt.proto, got, tt.expect)
		}
	}
}

func TestToStatusProto(t *testing.T) {
	tests := []struct {
		sdk    WorkflowStatus
		expect pb.WorkflowStatus
	}{
		{WorkflowStatusPending, pb.WorkflowStatus_WORKFLOW_STATUS_PENDING},
		{WorkflowStatusRunning, pb.WorkflowStatus_WORKFLOW_STATUS_RUNNING},
		{WorkflowStatusCompleted, pb.WorkflowStatus_WORKFLOW_STATUS_COMPLETED},
		{WorkflowStatus("UNKNOWN"), pb.WorkflowStatus_WORKFLOW_STATUS_UNSPECIFIED},
	}
	for _, tt := range tests {
		got := toStatusProto(tt.sdk)
		if got != tt.expect {
			t.Errorf("toStatusProto(%q) = %v, want %v", tt.sdk, got, tt.expect)
		}
	}
}

func TestToValueMap(t *testing.T) {
	input := map[string]any{
		"name":  "test",
		"count": float64(42),
		"flag":  true,
	}
	result, err := toValueMap(input)
	if err != nil {
		t.Fatalf("toValueMap: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(result))
	}
	if result["name"].GetStringValue() != "test" {
		t.Errorf("name = %v, want %q", result["name"], "test")
	}
	if result["count"].GetNumberValue() != 42 {
		t.Errorf("count = %v, want 42", result["count"])
	}
	if result["flag"].GetBoolValue() != true {
		t.Errorf("flag = %v, want true", result["flag"])
	}
}

func TestToValueMapInvalid(t *testing.T) {
	// channels can't be converted to structpb.Value
	input := map[string]any{"bad": make(chan int)}
	_, err := toValueMap(input)
	if err == nil {
		t.Fatal("expected error for unconvertible value")
	}
}

func TestFromValueMap(t *testing.T) {
	if fromValueMap(nil) != nil {
		t.Fatal("expected nil for nil input")
	}

	input := map[string]*structpb.Value{
		"x": structpb.NewStringValue("hello"),
		"y": structpb.NewNumberValue(3.14),
	}
	result := fromValueMap(input)
	if result["x"] != "hello" {
		t.Errorf("x = %v, want %q", result["x"], "hello")
	}
	if result["y"] != 3.14 {
		t.Errorf("y = %v, want 3.14", result["y"])
	}
}

func TestToTraceEntry(t *testing.T) {
	ts := timestamppb.Now()
	data := map[string]*structpb.Value{
		"key": structpb.NewStringValue("val"),
	}
	entry := toTraceEntry(&pb.TraceEntry{
		Timestamp: ts,
		StepType:  "activity",
		Details:   "running step 1",
		Status:    "completed",
		Error:     "",
		Data:      data,
	})
	if entry.StepType != "activity" {
		t.Errorf("StepType = %q, want %q", entry.StepType, "activity")
	}
	if entry.Details != "running step 1" {
		t.Errorf("Details = %q, want %q", entry.Details, "running step 1")
	}
	if entry.Status != "completed" {
		t.Errorf("Status = %q, want %q", entry.Status, "completed")
	}
	if entry.Data["key"] != "val" {
		t.Errorf("Data[key] = %v, want %q", entry.Data["key"], "val")
	}
}

func TestToRelated(t *testing.T) {
	rw := &pb.RelatedWorkflow{
		WorkflowId:   "wf-123",
		RunId:        "run-456",
		WorkflowType: "OrderProcessor",
		Status:       pb.WorkflowStatus_WORKFLOW_STATUS_RUNNING,
	}
	got := toRelated(rw)
	if got.WorkflowID != "wf-123" {
		t.Errorf("WorkflowID = %q, want %q", got.WorkflowID, "wf-123")
	}
	if got.RunID != "run-456" {
		t.Errorf("RunID = %q, want %q", got.RunID, "run-456")
	}
	if got.WorkflowType != "OrderProcessor" {
		t.Errorf("WorkflowType = %q, want %q", got.WorkflowType, "OrderProcessor")
	}
	if got.Status != WorkflowStatusRunning {
		t.Errorf("Status = %q, want %q", got.Status, WorkflowStatusRunning)
	}
}

func TestRunnerWorkflows(t *testing.T) {
	m := NewModule("test", WithModuleTaskQueue("test-queue"))

	// Without WorkflowURL → nil
	r, err := NewRunner(m, WithPlatformURL("http://localhost:10000"))
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	if r.Workflows() != nil {
		t.Error("expected nil Workflows() without WithWorkflowURL")
	}

	// With WorkflowURL → non-nil
	r2, err := NewRunner(m,
		WithPlatformURL("http://localhost:10000"),
		WithWorkflowURL("http://localhost:10000"),
	)
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}
	wc := r2.Workflows()
	if wc == nil {
		t.Fatal("expected non-nil Workflows() with WithWorkflowURL")
	}
	if wc.baseURL != "http://localhost:10000" {
		t.Errorf("Workflows().baseURL = %q, want %q", wc.baseURL, "http://localhost:10000")
	}
}

func TestNewRunnerRequiresPlatformURL(t *testing.T) {
	m := NewModule("test", WithModuleTaskQueue("test-queue"))
	_, err := NewRunner(m)
	if err == nil {
		t.Fatal("expected error when platformURL is not set")
	}
}
