package router

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test types for schema generation
type simpleType struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type withOptionalField struct {
	ID      string `json:"id"`
	Comment string `json:"comment,omitempty"`
}

type withDocAndExample struct {
	Status string `json:"status" doc:"Status of the resource" example:"active" enum:"active,inactive,pending"`
}

type withTime struct {
	CreatedAt time.Time `json:"createdAt"`
}

type withRawJSON struct {
	Data json.RawMessage `json:"data"`
}

type withArray struct {
	Tags  []string     `json:"tags"`
	Items []simpleType `json:"items"`
}

type withMap struct {
	Properties map[string]string     `json:"properties"`
	Objects    map[string]simpleType `json:"objects"`
}

// Circular reference test types
type parent struct {
	Name     string  `json:"name"`
	Children []child `json:"children,omitempty"`
}

type child struct {
	Name   string  `json:"name"`
	Parent *parent `json:"parent,omitempty"`
}

func TestJsonSchema(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		input    any
		expected map[string]any
	}{
		"simple type": {
			input: simpleType{},
			expected: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{
						"type": "string",
					},
					"age": map[string]any{
						"type": "integer",
					},
				},
				"required": []string{"name", "age"},
			},
		},
		"with optional field": {
			input: withOptionalField{},
			expected: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{
						"type": "string",
					},
					"comment": map[string]any{
						"type": "string",
					},
				},
				"required": []string{"id"},
			},
		},
		"with doc and example": {
			input: withDocAndExample{},
			expected: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"status": map[string]any{
						"type":        "string",
						"description": "Status of the resource",
						"example":     "active",
						"enum":        []string{"active", "inactive", "pending"},
					},
				},
				"required": []string{"status"},
			},
		},
		"with time.Time": {
			input: withTime{},
			expected: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"createdAt": map[string]any{
						"type":   "string",
						"format": "date-time",
					},
				},
				"required": []string{"createdAt"},
			},
		},
		"with json.RawMessage": {
			input: withRawJSON{},
			expected: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"data": map[string]any{
						"type": "object",
					},
				},
				"required": []string{"data"},
			},
		},
		"with array": {
			input: withArray{},
			expected: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"tags": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "string",
						},
					},
					"items": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"name": map[string]any{
									"type": "string",
								},
								"age": map[string]any{
									"type": "integer",
								},
							},
							"required": []string{"name", "age"},
						},
					},
				},
				"required": []string{"tags", "items"},
			},
		},
		"with map": {
			input: withMap{},
			expected: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"properties": map[string]any{
						"type": "object",
						"additionalProperties": map[string]any{
							"type": "string",
						},
					},
					"objects": map[string]any{
						"type": "object",
						"additionalProperties": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"name": map[string]any{
									"type": "string",
								},
								"age": map[string]any{
									"type": "integer",
								},
							},
							"required": []string{"name", "age"},
						},
					},
				},
				"required": []string{"properties", "objects"},
			},
		},
		"boolean type": {
			input: true,
			expected: map[string]any{
				"type": "boolean",
			},
		},
		"integer type": {
			input: 42,
			expected: map[string]any{
				"type": "integer",
			},
		},
		"float type": {
			input: 3.14,
			expected: map[string]any{
				"type": "number",
			},
		},
		"string type": {
			input: "test",
			expected: map[string]any{
				"type": "string",
			},
		},
	} {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			actual := jsonSchema(tc.input)

			if diff := cmp.Diff(tc.expected, actual); diff != "" {
				t.Errorf("schema mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCircularReferenceHandling(t *testing.T) {
	t.Parallel()

	// Create a circular reference
	p := &parent{Name: "Parent"}
	c := child{Name: "Child", Parent: p}
	p.Children = []child{c}

	// Generate schema with circular reference
	schema := jsonSchema(p)

	// Verify schema is of object type
	assert.Equal(t, "object", schema["type"], "Schema should be an object type")

	// Verify children property exists
	properties, ok := schema["properties"].(map[string]any)
	require.True(t, ok, "Schema should have properties")

	children, ok := properties["children"].(map[string]any)
	require.True(t, ok, "Schema should have children property")

	assert.Equal(t, "array", children["type"], "Children should be an array")

	// Check that we can serialize the schema without infinite recursion
	_, err := json.Marshal(schema)
	assert.NoError(t, err, "Schema with circular reference should serialize without error")
}

func TestParseJsonTag(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		jsonTag   string
		fieldName string
		wantName  string
		required  bool
	}{
		"empty tag": {
			jsonTag:   "",
			fieldName: "Field",
			wantName:  "Field",
			required:  true,
		},
		"with name": {
			jsonTag:   "customName",
			fieldName: "Field",
			wantName:  "customName",
			required:  true,
		},
		"with omitempty": {
			jsonTag:   "field,omitempty",
			fieldName: "Field",
			wantName:  "field",
			required:  false,
		},
		"with empty name": {
			jsonTag:   ",omitempty",
			fieldName: "Field",
			wantName:  "Field",
			required:  false,
		},
		"with other options": {
			jsonTag:   "field,string",
			fieldName: "Field",
			wantName:  "field",
			required:  true,
		},
	} {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			gotName, isRequired := parseJsonTag(tc.jsonTag, tc.fieldName)

			assert.Equal(t, tc.wantName, gotName, "Field name should match expected")
			assert.Equal(t, tc.required, isRequired, "Required flag should match expected")
		})
	}
}

func TestGetTypeName(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		input    any
		expected string
	}{
		"nil": {
			input:    nil,
			expected: "",
		},
		"simple type": {
			input:    simpleType{},
			expected: "simpleType",
		},
		"pointer to type": {
			input:    &simpleType{},
			expected: "simpleType",
		},
		"primitive": {
			input:    "string",
			expected: "string",
		},
		"anonymous struct": {
			input:    struct{ A int }{},
			expected: "",
		},
	} {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			actual := getTypeName(tc.input)

			assert.Equal(t, tc.expected, actual, "Type name should match expected")
		})
	}
}

func TestSchemaGenerator(t *testing.T) {
	t.Parallel()

	// Test the generator directly
	g := newSchemaGenerator()

	// Verify correct struct processing
	result := g.processStruct(reflect.TypeOf(simpleType{}))

	expected := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{
				"type": "string",
			},
			"age": map[string]any{
				"type": "integer",
			},
		},
		"required": []string{"name", "age"},
	}

	if diff := cmp.Diff(expected, result); diff != "" {
		t.Errorf("processStruct mismatch (-want +got):\n%s", diff)
	}
}

func TestExtractNestedTypes(t *testing.T) {
	t.Parallel()

	// Create test schema with nested types
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"nested": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"field": map[string]any{
						"type": "string",
					},
				},
			},
			"items": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"name": map[string]any{
							"type": "string",
						},
					},
				},
			},
		},
	}

	// Create registry and extract nested types
	registry := newSchemaRegistry()
	extractNestedTypes(schema, "Test", registry)

	// Verify nested types were extracted and references created
	schemas := registry.getSchemas()

	// Check nested object was extracted
	_, exists := schemas["TestNested"]
	assert.True(t, exists, "TestNested schema should be registered")

	// Check array item was extracted
	_, exists = schemas["TestItemsItem"]
	assert.True(t, exists, "TestItemsItem schema should be registered")

	// Check references were updated
	properties := schema["properties"].(map[string]any)
	nestedProp := properties["nested"].(map[string]any)

	assert.Contains(t, nestedProp, "$ref", "Nested property should have $ref")
	assert.Equal(t, "#/components/schemas/TestNested", nestedProp["$ref"], "Reference should point to extracted schema")
}
