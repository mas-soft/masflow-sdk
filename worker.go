package masflowsdk

import (
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/worker"
)

// RegisterAll registers all activities from the module with a Temporal worker.
func RegisterAll(w worker.Worker, m *Module) {
	for name, def := range m.Activities() {
		if def.handlerFunc == nil {
			continue
		}
		w.RegisterActivityWithOptions(def.handlerFunc, activity.RegisterOptions{
			Name: name,
		})
	}
}
