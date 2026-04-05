package masflowsdk

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/invopop/jsonschema"
	"google.golang.org/protobuf/proto"
)

// generateSchema produces a JSON Schema for the given Go type.
func generateSchema[T any]() ([]byte, error) {
	var zero T
	t := reflect.TypeOf(zero)
	if t == nil {
		return nil, nil
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return nil, nil
	}

	r := &jsonschema.Reflector{}
	schema := r.Reflect(zero)
	if schema == nil {
		return nil, nil
	}

	return json.Marshal(schema)
}

// typeURL returns a type URL for a Go type.
// For proto.Message types, uses the proto full name.
// For plain Go structs, uses "go/<pkgpath>.<TypeName>".
func typeURL[T any]() string {
	var zero T
	t := reflect.TypeOf(zero)
	if t == nil {
		return ""
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	// Check if it implements proto.Message
	if t.Kind() == reflect.Struct {
		ptrType := reflect.PointerTo(t)
		if ptrType.Implements(reflect.TypeOf((*proto.Message)(nil)).Elem()) {
			msg := reflect.New(t).Interface().(proto.Message)
			return string(msg.ProtoReflect().Descriptor().FullName())
		}
	}

	return fmt.Sprintf("go/%s.%s", t.PkgPath(), t.Name())
}
