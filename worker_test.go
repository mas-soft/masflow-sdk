package masflowsdk

import (
	"strings"
	"testing"

	"github.com/nexus-rpc/sdk-go/nexus"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

type recordingWorker struct {
	activityNames []string
	registerFn    func(a interface{}, options activity.RegisterOptions)
}

func (w *recordingWorker) RegisterActivity(interface{}) {}

func (w *recordingWorker) RegisterActivityWithOptions(a interface{}, options activity.RegisterOptions) {
	if w.registerFn != nil {
		w.registerFn(a, options)
		return
	}
	w.activityNames = append(w.activityNames, options.Name)
}

func (w *recordingWorker) RegisterDynamicActivity(interface{}, activity.DynamicRegisterOptions) {}

func (w *recordingWorker) RegisterWorkflow(interface{}) {}

func (w *recordingWorker) RegisterWorkflowWithOptions(interface{}, workflow.RegisterOptions) {}

func (w *recordingWorker) RegisterDynamicWorkflow(interface{}, workflow.DynamicRegisterOptions) {}

func (w *recordingWorker) RegisterNexusService(*nexus.Service) {}

func (w *recordingWorker) Start() error { return nil }

func (w *recordingWorker) Run(<-chan interface{}) error { return nil }

func (w *recordingWorker) Stop() {}

var _ worker.Worker = (*recordingWorker)(nil)

func TestRegisterAllRegistersActivities(t *testing.T) {
	mod := NewModule("test-module")
	if err := Register(mod, "processOrder", processOrder); err != nil {
		t.Fatalf("Register failed: %v", err)
	}
	if err := RegisterVoid(mod, "logEvent", logEvent); err != nil {
		t.Fatalf("RegisterVoid failed: %v", err)
	}

	w := &recordingWorker{}
	if err := RegisterAll(w, mod); err != nil {
		t.Fatalf("RegisterAll failed: %v", err)
	}

	if len(w.activityNames) != 2 {
		t.Fatalf("expected 2 activities to be registered, got %d", len(w.activityNames))
	}

	registered := map[string]bool{}
	for _, name := range w.activityNames {
		registered[name] = true
	}
	if !registered["processOrder"] || !registered["logEvent"] {
		t.Fatalf("unexpected registered activities: %v", w.activityNames)
	}
}

func TestRegisterAllRejectsMissingHandler(t *testing.T) {
	mod := NewModule("test-module")
	mod.addActivity(&Definition{Name: "broken"})

	err := RegisterAll(&recordingWorker{}, mod)
	if err == nil {
		t.Fatal("expected RegisterAll to fail for missing handler")
	}
	if !strings.Contains(err.Error(), "activity \"broken\" has no handler") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRegisterAllRecoversRegistrationPanic(t *testing.T) {
	mod := NewModule("test-module")
	if err := Register(mod, "processOrder", processOrder); err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	w := &recordingWorker{
		registerFn: func(interface{}, activity.RegisterOptions) {
			panic("boom")
		},
	}

	err := RegisterAll(w, mod)
	if err == nil {
		t.Fatal("expected RegisterAll to convert registration panic into an error")
	}
	if !strings.Contains(err.Error(), "register activity with Temporal worker: boom") {
		t.Fatalf("unexpected error: %v", err)
	}
}
