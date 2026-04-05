package masflowsdk

import "fmt"

// Register adds a typed sync activity to a module.
// It auto-generates JSON Schema from the Go input/output types and infers type URLs.
func Register[TReq, TRes any](m *Module, name string, handler Handler[TReq, TRes], opts ...ActivityOption) error {
	if name == "" {
		return fmt.Errorf("activity name is required")
	}
	if _, exists := m.activities[name]; exists {
		return fmt.Errorf("activity %q is already registered in module %q", name, m.Name)
	}

	def := &Definition{
		Name:        name,
		InputType:   typeURL[TReq](),
		OutputType:  typeURL[TRes](),
		handlerFunc: handler,
	}

	for _, opt := range opts {
		opt(def)
	}

	if inputSchema, err := generateSchema[TReq](); err == nil {
		def.InputSchemaJSON = inputSchema
	}
	if outputSchema, err := generateSchema[TRes](); err == nil {
		def.OutputSchemaJSON = outputSchema
	}

	m.addActivity(def)
	return nil
}

// RegisterVoid adds a typed activity that returns only an error (no response value).
func RegisterVoid[TReq any](m *Module, name string, handler VoidHandler[TReq], opts ...ActivityOption) error {
	if name == "" {
		return fmt.Errorf("activity name is required")
	}
	if _, exists := m.activities[name]; exists {
		return fmt.Errorf("activity %q is already registered in module %q", name, m.Name)
	}

	def := &Definition{
		Name:        name,
		InputType:   typeURL[TReq](),
		OutputType:  "",
		handlerFunc: handler,
	}

	for _, opt := range opts {
		opt(def)
	}

	if inputSchema, err := generateSchema[TReq](); err == nil {
		def.InputSchemaJSON = inputSchema
	}

	m.addActivity(def)
	return nil
}

// RegisterAsync adds a typed async-capable activity to a module.
// Async activities receive an AsyncCallbackInfo with workflow_id, run_id, and callback_signal.
func RegisterAsync[TReq, TRes any](m *Module, name string, handler AsyncHandler[TReq, TRes], opts ...ActivityOption) error {
	if name == "" {
		return fmt.Errorf("activity name is required")
	}
	if _, exists := m.activities[name]; exists {
		return fmt.Errorf("activity %q is already registered in module %q", name, m.Name)
	}

	def := &Definition{
		Name:          name,
		InputType:     typeURL[TReq](),
		OutputType:    typeURL[TRes](),
		SupportsAsync: true,
		handlerFunc:   handler,
	}

	for _, opt := range opts {
		opt(def)
	}

	if inputSchema, err := generateSchema[TReq](); err == nil {
		def.InputSchemaJSON = inputSchema
	}
	if outputSchema, err := generateSchema[TRes](); err == nil {
		def.OutputSchemaJSON = outputSchema
	}

	m.addActivity(def)
	return nil
}
