package masflowsdk

import (
	pb "github.com/mas-soft/masflow/sdk/internal/activitypb/activity"
	"google.golang.org/protobuf/types/known/anypb"
)

// Definition describes a single activity's contract and holds its handler.
type Definition struct {
	Name             string
	Description      string
	Icon             string
	InputType        string // type URL (e.g. "go/pkg.TypeName" or proto full name)
	OutputType       string // type URL
	InputSchemaJSON  []byte // JSON Schema bytes (auto-generated)
	OutputSchemaJSON []byte // JSON Schema bytes (auto-generated)
	SupportsAsync    bool
	TaskQueue        string // override module-level task queue
	Category         string
	Tags             []string
	DocumentationURL string
	handlerFunc      interface{} // actual function for Temporal registration
}

// toProto converts a Definition to its proto representation.
func (d *Definition) toProto() *pb.ActivityDefinition {
	ad := &pb.ActivityDefinition{
		Name:          d.Name,
		Description:   d.Description,
		Icon:          d.Icon,
		InputTypeUrl:  d.InputType,
		OutputTypeUrl: d.OutputType,
		SupportsAsync: d.SupportsAsync,
		TaskQueue:     d.TaskQueue,
		Metadata: &pb.ActivityMetadata{
			DocumentationUrl: d.DocumentationURL,
			Tags:             d.Tags,
			Category:         d.Category,
		},
	}

	if d.InputSchemaJSON != nil {
		ad.InputSchema = schemaToAny(d.InputSchemaJSON)
	}
	if d.OutputSchemaJSON != nil {
		ad.OutputSchema = schemaToAny(d.OutputSchemaJSON)
	}

	return ad
}

// schemaToAny wraps JSON Schema bytes in a protobuf Any.
func schemaToAny(schemaJSON []byte) *anypb.Any {
	if schemaJSON == nil {
		return nil
	}
	return &anypb.Any{
		TypeUrl: "type.googleapis.com/json-schema",
		Value:   schemaJSON,
	}
}

// ActivityOption configures a Definition.
type ActivityOption func(*Definition)

// WithDescription sets the activity description.
func WithDescription(desc string) ActivityOption {
	return func(d *Definition) { d.Description = desc }
}

// WithIcon sets the activity icon.
func WithIcon(icon string) ActivityOption {
	return func(d *Definition) { d.Icon = icon }
}

// WithCategory sets the activity category.
func WithCategory(cat string) ActivityOption {
	return func(d *Definition) { d.Category = cat }
}

// WithTags sets the activity tags.
func WithTags(tags ...string) ActivityOption {
	return func(d *Definition) { d.Tags = tags }
}

// WithTaskQueue overrides the module-level task queue for this activity.
func WithTaskQueue(tq string) ActivityOption {
	return func(d *Definition) { d.TaskQueue = tq }
}

// WithDocumentationURL sets the documentation URL for this activity.
func WithDocumentationURL(url string) ActivityOption {
	return func(d *Definition) { d.DocumentationURL = url }
}
