package masflowsdk

import (
	"fmt"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/worker"
)

// RegisterAll registers all activities from the module with a Temporal worker.
func RegisterAll(w worker.Worker, m *Module) (err error) {
	if w == nil {
		return fmt.Errorf("worker is required")
	}
	if m == nil {
		return fmt.Errorf("module is required")
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("register activity with Temporal worker: %v", recovered)
		}
	}()

	for name, def := range m.Activities() {
		if def == nil {
			return fmt.Errorf("activity %q definition is nil", name)
		}
		if def.Name == "" {
			return fmt.Errorf("activity %q has empty definition name", name)
		}
		if def.Name != name {
			return fmt.Errorf("activity %q definition name mismatch: %q", name, def.Name)
		}
		if def.handlerFunc == nil {
			return fmt.Errorf("activity %q has no handler", name)
		}
		w.RegisterActivityWithOptions(def.handlerFunc, activity.RegisterOptions{
			Name: name,
		})
	}

	return nil
}
