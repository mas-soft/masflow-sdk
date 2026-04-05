package masflowsdk

import (
	"context"
	"testing"
)

// --- Test types ---

type OrderInput struct {
	OrderID  string  `json:"order_id"`
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

type OrderOutput struct {
	Status    string `json:"status"`
	Reference string `json:"reference"`
}

type LogInput struct {
	Message string `json:"message"`
	Level   string `json:"level"`
}

func processOrder(_ context.Context, input OrderInput) (OrderOutput, error) {
	return OrderOutput{Status: "processed", Reference: input.OrderID}, nil
}

func logEvent(_ context.Context, input LogInput) error {
	return nil
}

func asyncProcess(_ context.Context, input OrderInput, async *AsyncCallbackInfo) (OrderOutput, error) {
	return OrderOutput{Status: "pending", Reference: async.WorkflowID}, nil
}

// --- Tests ---

func TestRegister(t *testing.T) {
	mod := NewModule("test-module",
		WithModuleDescription("Test module"),
		WithModuleVersion("1.0.0"),
		WithModuleTaskQueue("test-queue"),
	)

	err := Register(mod, "processOrder", processOrder,
		WithDescription("Process an order"),
		WithCategory("orders"),
		WithTags("order", "payment"),
	)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	def, ok := mod.GetActivity("processOrder")
	if !ok {
		t.Fatal("activity not found after registration")
	}

	if def.Name != "processOrder" {
		t.Errorf("expected name 'processOrder', got %q", def.Name)
	}
	if def.Description != "Process an order" {
		t.Errorf("expected description 'Process an order', got %q", def.Description)
	}
	if def.Category != "orders" {
		t.Errorf("expected category 'orders', got %q", def.Category)
	}
	if def.InputType == "" {
		t.Error("expected InputType to be set")
	}
	if def.OutputType == "" {
		t.Error("expected OutputType to be set")
	}
	if def.InputSchemaJSON == nil {
		t.Error("expected InputSchemaJSON to be auto-generated")
	}
	if def.OutputSchemaJSON == nil {
		t.Error("expected OutputSchemaJSON to be auto-generated")
	}
	if def.handlerFunc == nil {
		t.Error("expected handlerFunc to be set")
	}
}

func TestRegisterVoid(t *testing.T) {
	mod := NewModule("test-module", WithModuleTaskQueue("test-queue"))

	err := RegisterVoid(mod, "logEvent", logEvent,
		WithDescription("Log an event"),
	)
	if err != nil {
		t.Fatalf("RegisterVoid failed: %v", err)
	}

	def, ok := mod.GetActivity("logEvent")
	if !ok {
		t.Fatal("activity not found after registration")
	}

	if def.OutputType != "" {
		t.Errorf("expected empty OutputType for void handler, got %q", def.OutputType)
	}
	if def.OutputSchemaJSON != nil {
		t.Error("expected nil OutputSchemaJSON for void handler")
	}
}

func TestRegisterAsync(t *testing.T) {
	mod := NewModule("test-module", WithModuleTaskQueue("test-queue"))

	err := RegisterAsync(mod, "asyncProcess", asyncProcess,
		WithDescription("Async process"),
	)
	if err != nil {
		t.Fatalf("RegisterAsync failed: %v", err)
	}

	def, ok := mod.GetActivity("asyncProcess")
	if !ok {
		t.Fatal("activity not found after registration")
	}

	if !def.SupportsAsync {
		t.Error("expected SupportsAsync to be true")
	}
}

func TestRegisterDuplicate(t *testing.T) {
	mod := NewModule("test-module", WithModuleTaskQueue("test-queue"))

	err := Register(mod, "processOrder", processOrder)
	if err != nil {
		t.Fatalf("first Register failed: %v", err)
	}

	err = Register(mod, "processOrder", processOrder)
	if err == nil {
		t.Fatal("expected error on duplicate registration")
	}
}

func TestRegisterEmptyName(t *testing.T) {
	mod := NewModule("test-module", WithModuleTaskQueue("test-queue"))

	err := Register(mod, "", processOrder)
	if err == nil {
		t.Fatal("expected error on empty activity name")
	}
}

func TestModuleOptions(t *testing.T) {
	mod := NewModule("my-module",
		WithModuleDescription("A test module"),
		WithModuleVersion("2.0.0"),
		WithModuleIcon("star"),
		WithModuleTaskQueue("my-queue"),
		WithModuleAuthor("tester"),
		WithModuleCategory("testing"),
		WithModuleTags("tag1", "tag2"),
	)

	if mod.Name != "my-module" {
		t.Errorf("expected name 'my-module', got %q", mod.Name)
	}
	if mod.Description != "A test module" {
		t.Errorf("expected description, got %q", mod.Description)
	}
	if mod.Version != "2.0.0" {
		t.Errorf("expected version '2.0.0', got %q", mod.Version)
	}
	if mod.TaskQueue != "my-queue" {
		t.Errorf("expected task queue 'my-queue', got %q", mod.TaskQueue)
	}
	if mod.Author != "tester" {
		t.Errorf("expected author 'tester', got %q", mod.Author)
	}
	if mod.Category != "testing" {
		t.Errorf("expected category 'testing', got %q", mod.Category)
	}
	if len(mod.Tags) != 2 || mod.Tags[0] != "tag1" {
		t.Errorf("expected tags [tag1, tag2], got %v", mod.Tags)
	}
}

func TestModuleToProto(t *testing.T) {
	mod := NewModule("proto-test",
		WithModuleDescription("Proto conversion test"),
		WithModuleVersion("1.0.0"),
		WithModuleTaskQueue("proto-queue"),
		WithModuleAuthor("tester"),
		WithModuleCategory("testing"),
		WithModuleTags("proto"),
	)

	Register(mod, "processOrder", processOrder,
		WithDescription("Process an order"),
		WithCategory("orders"),
	)

	pb := mod.toProto()

	if pb.GetName() != "proto-test" {
		t.Errorf("expected proto name 'proto-test', got %q", pb.GetName())
	}
	if pb.GetTaskQueue() != "proto-queue" {
		t.Errorf("expected proto task_queue 'proto-queue', got %q", pb.GetTaskQueue())
	}
	if len(pb.GetActivities()) != 1 {
		t.Errorf("expected 1 activity in proto, got %d", len(pb.GetActivities()))
	}
	if pb.GetMetadata().GetAuthor() != "tester" {
		t.Errorf("expected author 'tester', got %q", pb.GetMetadata().GetAuthor())
	}
}
