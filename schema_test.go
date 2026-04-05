package masflowsdk

import (
	"encoding/json"
	"strings"
	"testing"
)

type SchemaTestInput struct {
	Name    string   `json:"name"`
	Count   int      `json:"count"`
	Tags    []string `json:"tags,omitempty"`
	Enabled bool     `json:"enabled"`
}

func TestGenerateSchema(t *testing.T) {
	schema, err := generateSchema[SchemaTestInput]()
	if err != nil {
		t.Fatalf("generateSchema failed: %v", err)
	}
	if schema == nil {
		t.Fatal("expected non-nil schema")
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(schema, &doc); err != nil {
		t.Fatalf("schema is not valid JSON: %v", err)
	}

	// invopop/jsonschema uses $ref + $defs pattern
	defs, hasDefs := doc["$defs"]
	if !hasDefs {
		t.Fatal("expected '$defs' in schema")
	}

	defsMap, ok := defs.(map[string]interface{})
	if !ok {
		t.Fatal("expected '$defs' to be a map")
	}

	// Find the type definition
	typeDef, ok := defsMap["SchemaTestInput"]
	if !ok {
		t.Fatal("expected 'SchemaTestInput' in $defs")
	}

	typeDefMap, ok := typeDef.(map[string]interface{})
	if !ok {
		t.Fatal("expected type def to be a map")
	}

	props, ok := typeDefMap["properties"]
	if !ok {
		t.Fatal("expected 'properties' in type definition")
	}

	propsMap, ok := props.(map[string]interface{})
	if !ok {
		t.Fatal("expected 'properties' to be a map")
	}

	for _, field := range []string{"name", "count", "tags", "enabled"} {
		if _, ok := propsMap[field]; !ok {
			t.Errorf("expected field %q in schema properties", field)
		}
	}
}

func TestGenerateSchemaEmpty(t *testing.T) {
	// Non-struct types should return nil
	schema, err := generateSchema[string]()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if schema != nil {
		t.Error("expected nil schema for string type")
	}
}

func TestTypeURL(t *testing.T) {
	url := typeURL[SchemaTestInput]()
	if url == "" {
		t.Fatal("expected non-empty type URL")
	}
	if !strings.HasPrefix(url, "go/") {
		t.Errorf("expected type URL to start with 'go/', got %q", url)
	}
	if !strings.Contains(url, "SchemaTestInput") {
		t.Errorf("expected type URL to contain 'SchemaTestInput', got %q", url)
	}
}
