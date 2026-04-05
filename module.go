package masflowsdk

import (
	"time"

	pb "github.com/mas-soft/masflow/sdk/internal/pb/activity"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Module groups activities from a single provider.
type Module struct {
	Name        string
	Description string
	Version     string
	Icon        string
	TaskQueue   string
	Author      string
	Category    string
	Tags        []string
	activities  map[string]*Definition
}

// NewModule creates a new module with the given name and options.
func NewModule(name string, opts ...ModuleOption) *Module {
	m := &Module{
		Name:       name,
		activities: make(map[string]*Definition),
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// Activities returns all activity definitions in this module.
func (m *Module) Activities() map[string]*Definition {
	return m.activities
}

// GetActivity returns an activity definition by name.
func (m *Module) GetActivity(name string) (*Definition, bool) {
	d, ok := m.activities[name]
	return d, ok
}

// addActivity adds an activity definition to the module.
func (m *Module) addActivity(d *Definition) {
	m.activities[d.Name] = d
}

// toProto converts a Module to its proto representation for platform registration.
func (m *Module) toProto() *pb.Module {
	pbMod := &pb.Module{
		Name:        m.Name,
		Description: m.Description,
		Version:     m.Version,
		Icon:        m.Icon,
		TaskQueue:   m.TaskQueue,
		Metadata: &pb.ModuleMetadata{
			Author:       m.Author,
			Tags:         m.Tags,
			Category:     m.Category,
			RegisteredAt: timestamppb.New(time.Now()),
		},
	}

	for _, def := range m.activities {
		pbMod.Activities = append(pbMod.Activities, def.toProto())
	}

	return pbMod
}

// ModuleOption configures a Module.
type ModuleOption func(*Module)

// WithModuleDescription sets the module description.
func WithModuleDescription(desc string) ModuleOption {
	return func(m *Module) { m.Description = desc }
}

// WithModuleVersion sets the module version.
func WithModuleVersion(ver string) ModuleOption {
	return func(m *Module) { m.Version = ver }
}

// WithModuleIcon sets the module icon.
func WithModuleIcon(icon string) ModuleOption {
	return func(m *Module) { m.Icon = icon }
}

// WithModuleTaskQueue sets the Temporal task queue for all activities in this module.
func WithModuleTaskQueue(tq string) ModuleOption {
	return func(m *Module) { m.TaskQueue = tq }
}

// WithModuleAuthor sets the module author.
func WithModuleAuthor(author string) ModuleOption {
	return func(m *Module) { m.Author = author }
}

// WithModuleCategory sets the module category.
func WithModuleCategory(cat string) ModuleOption {
	return func(m *Module) { m.Category = cat }
}

// WithModuleTags sets the module tags.
func WithModuleTags(tags ...string) ModuleOption {
	return func(m *Module) { m.Tags = tags }
}
