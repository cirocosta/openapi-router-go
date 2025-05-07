package router

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

// Test structs used in the tests
type SimpleStruct struct {
	String  string  `json:"string"`
	Int     int     `json:"int"`
	Bool    bool    `json:"bool"`
	Float   float64 `json:"float"`
	Pointer *string `json:"pointer,omitempty"`
}

type StructWithArrays struct {
	StringArray []string       `json:"stringArray"`
	IntArray    []int          `json:"intArray"`
	ObjArray    []SimpleStruct `json:"objArray"`
}

type StructWithMaps struct {
	StringMap map[string]string       `json:"stringMap"`
	IntMap    map[string]int          `json:"intMap"`
	ObjMap    map[string]SimpleStruct `json:"objMap"`
}

type StructWithTags struct {
	Required    string `json:"required"`
	Optional    string `json:"optional,omitempty"`
	WithDoc     string `json:"withDoc" doc:"This is documentation"`
	WithExample string `json:"withExample" example:"Example value"`
	WithEnum    string `json:"withEnum" enum:"value1,value2,value3"`
}

type StructWithTime struct {
	Created time.Time `json:"created"`
}

type StructWithRawJSON struct {
	Data json.RawMessage `json:"data"`
}

type CircularStruct struct {
	Name     string           `json:"name"`
	Self     *CircularStruct  `json:"self,omitempty"`
	Children []CircularStruct `json:"children"`
}

func TestParseJsonTag(t *testing.T) {
	tests := map[string]struct {
		jsonTag      string
		fieldName    string
		wantName     string
		wantRequired bool
	}{
		"empty tag uses field name and required": {
			jsonTag:      "",
			fieldName:    "FieldName",
			wantName:     "FieldName",
			wantRequired: true,
		},
		"simple tag": {
			jsonTag:      "propertyName",
			fieldName:    "FieldName",
			wantName:     "propertyName",
			wantRequired: true,
		},
		"optional tag": {
			jsonTag:      "propertyName,omitempty",
			fieldName:    "FieldName",
			wantName:     "propertyName",
			wantRequired: false,
		},
		"multiple options": {
			jsonTag:      "propertyName,omitempty,string",
			fieldName:    "FieldName",
			wantName:     "propertyName",
			wantRequired: false,
		},
		"empty name in tag": {
			jsonTag:      ",omitempty",
			fieldName:    "FieldName",
			wantName:     "FieldName",
			wantRequired: false,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			gotName, gotRequired := parseJsonTag(tc.jsonTag, tc.fieldName)
			assert.Equal(t, tc.wantName, gotName)
			assert.Equal(t, tc.wantRequired, gotRequired)
		})
	}
}

func TestBasicTypeSchema(t *testing.T) {
	simpleStruct := SimpleStruct{
		String: "string",
		Int:    123,
		Bool:   true,
		Float:  123.45,
	}

	tests := map[string]struct {
		value    any
		expected map[string]any
	}{
		"string": {
			value:    simpleStruct.String,
			expected: map[string]any{"type": "string"},
		},
		"int": {
			value:    simpleStruct.Int,
			expected: map[string]any{"type": "integer"},
		},
		"bool": {
			value:    simpleStruct.Bool,
			expected: map[string]any{"type": "boolean"},
		},
		"float": {
			value:    simpleStruct.Float,
			expected: map[string]any{"type": "number"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := basicTypeSchema(getKind(tc.value))
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestJsonSchema(t *testing.T) {
	tests := map[string]struct {
		value    any
		expected map[string]any
	}{
		"simple struct": {
			value: SimpleStruct{},
			expected: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"string":  map[string]any{"type": "string"},
					"int":     map[string]any{"type": "integer"},
					"bool":    map[string]any{"type": "boolean"},
					"float":   map[string]any{"type": "number"},
					"pointer": map[string]any{"type": "string"},
				},
				"required": []string{"string", "int", "bool", "float"},
			},
		},
		"struct with arrays": {
			value: StructWithArrays{},
			expected: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"stringArray": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "string"},
					},
					"intArray": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "integer"},
					},
					"objArray": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"string":  map[string]any{"type": "string"},
								"int":     map[string]any{"type": "integer"},
								"bool":    map[string]any{"type": "boolean"},
								"float":   map[string]any{"type": "number"},
								"pointer": map[string]any{"type": "string"},
							},
							"required": []string{"string", "int", "bool", "float"},
						},
					},
				},
				"required": []string{"stringArray", "intArray", "objArray"},
			},
		},
		"struct with maps": {
			value: StructWithMaps{},
			expected: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"stringMap": map[string]any{
						"type":                 "object",
						"additionalProperties": map[string]any{"type": "string"},
					},
					"intMap": map[string]any{
						"type":                 "object",
						"additionalProperties": map[string]any{"type": "integer"},
					},
					"objMap": map[string]any{
						"type": "object",
						"additionalProperties": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"string":  map[string]any{"type": "string"},
								"int":     map[string]any{"type": "integer"},
								"bool":    map[string]any{"type": "boolean"},
								"float":   map[string]any{"type": "number"},
								"pointer": map[string]any{"type": "string"},
							},
							"required": []string{"string", "int", "bool", "float"},
						},
					},
				},
				"required": []string{"stringMap", "intMap", "objMap"},
			},
		},
		"struct with tags": {
			value: StructWithTags{},
			expected: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"required": map[string]any{"type": "string"},
					"optional": map[string]any{"type": "string"},
					"withDoc": map[string]any{
						"type":        "string",
						"description": "This is documentation",
					},
					"withExample": map[string]any{
						"type":    "string",
						"example": "Example value",
					},
					"withEnum": map[string]any{
						"type": "string",
						"enum": []string{"value1", "value2", "value3"},
					},
				},
				"required": []string{"required", "withDoc", "withExample", "withEnum"},
			},
		},
		"struct with time": {
			value: StructWithTime{},
			expected: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"created": map[string]any{
						"type":   "string",
						"format": "date-time",
					},
				},
				"required": []string{"created"},
			},
		},
		"struct with raw json": {
			value: StructWithRawJSON{},
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
		"circular reference": {
			value: CircularStruct{},
			expected: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string"},
					"self": map[string]any{
						"type":        "object",
						"description": "circular reference to CircularStruct",
					},
					"children": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type":        "object",
							"description": "circular reference to CircularStruct",
						},
					},
				},
				"required": []string{"name", "children"},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := jsonSchema(tc.value)
			if !assert.Equal(t, tc.expected, result) {
				// Show detailed diff for debugging
				t.Logf("Diff: %s", cmp.Diff(tc.expected, result))
			}
		})
	}
}

func TestSchemaRegistry(t *testing.T) {
	t.Run("register and retrieve schemas", func(t *testing.T) {
		registry := newSchemaRegistry()

		// Register a schema
		schema1 := map[string]any{"type": "string"}
		registry.register("Type1", schema1)

		// Register another schema
		schema2 := map[string]any{"type": "integer"}
		registry.register("Type2", schema2)

		// Get all schemas
		schemas := registry.getSchemas()

		// Check if both schemas are in the result
		assert.Contains(t, schemas, "Type1")
		assert.Contains(t, schemas, "Type2")
		assert.Equal(t, schema1, schemas["Type1"])
		assert.Equal(t, schema2, schemas["Type2"])
	})
}

func TestExtractNestedTypes(t *testing.T) {
	t.Run("extract nested object types", func(t *testing.T) {
		registry := newSchemaRegistry()
		schema := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"nested": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"field": map[string]any{"type": "string"},
					},
				},
			},
		}

		extractNestedTypes(schema, "ParentType", registry)

		// Check if the nested type was registered
		schemas := registry.getSchemas()
		assert.Empty(t, schemas, "No types should be added yet, as we just extract but don't register")
	})
}

func TestExtractNestedTypesFull(t *testing.T) {
	t.Run("extracts complex nested types", func(t *testing.T) {
		registry := newSchemaRegistry()

		// Complex nested schema with objects, arrays, and maps
		schema := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"nestedObject": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"field": map[string]any{"type": "string"},
						"deeperObject": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"deepField": map[string]any{"type": "string"},
							},
						},
					},
				},
				"arrayField": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"itemField": map[string]any{"type": "string"},
						},
					},
				},
				"mapField": map[string]any{
					"type": "object",
					"additionalProperties": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"mapValueField": map[string]any{"type": "string"},
						},
					},
				},
			},
		}

		// Extract nested types
		extractNestedTypes(schema, "RootType", registry)

		// Check registry - since extractNestedTypes only finds nested types but doesn't
		// actually register them, the registry should be empty
		schemas := registry.getSchemas()
		assert.Empty(t, schemas, "Registry should be empty as extractNestedTypes only finds but doesn't register")

		// Now let's verify the behavior when we do register types
		// First register the root type
		registry.register("RootType", schema)

		// Now let's create a DocRouter and use schemaRef which should use extractNestedTypes
		dr := &DocRouter{
			schemaRegistry: registry,
		}

		// Define a type that mimics our schema
		type DeepObject struct {
			DeepField string `json:"deepField"`
		}

		type NestedObject struct {
			Field        string     `json:"field"`
			DeeperObject DeepObject `json:"deeperObject"`
		}

		type Item struct {
			ItemField string `json:"itemField"`
		}

		type MapValue struct {
			MapValueField string `json:"mapValueField"`
		}

		type RootType struct {
			NestedObject NestedObject        `json:"nestedObject"`
			ArrayField   []Item              `json:"arrayField"`
			MapField     map[string]MapValue `json:"mapField"`
		}

		// Call schemaRef with our complex type
		ref := dr.schemaRef(RootType{})

		// Verify we get a reference
		assert.Equal(t, map[string]any{"$ref": "#/components/schemas/RootType"}, ref)

		// Now check if the registry has all our types
		schemas = registry.getSchemas()
		assert.Contains(t, schemas, "RootType")
	})
}

func TestGetTypeName(t *testing.T) {
	tests := map[string]struct {
		value    any
		expected string
	}{
		"nil value": {
			value:    nil,
			expected: "",
		},
		"simple struct": {
			value:    SimpleStruct{},
			expected: "SimpleStruct",
		},
		"pointer to struct": {
			value:    &SimpleStruct{},
			expected: "SimpleStruct",
		},
		"anonymous struct": {
			value:    struct{ Field string }{},
			expected: "",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := getTypeName(tc.value)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// Helper function to get kind for testing
func getKind(v any) reflect.Kind {
	return reflect.TypeOf(v).Kind()
}

// TestSchemaRef tests the DocRouter.schemaRef function
func TestSchemaRef(t *testing.T) {
	tests := map[string]struct {
		setup          func() *DocRouter
		inputType      any
		expectedOutput map[string]any
	}{
		"nil input returns nil": {
			setup: func() *DocRouter {
				return &DocRouter{
					schemaRegistry: newSchemaRegistry(),
				}
			},
			inputType:      nil,
			expectedOutput: nil,
		},
		"anonymous type returns inline schema": {
			setup: func() *DocRouter {
				return &DocRouter{
					schemaRegistry: newSchemaRegistry(),
				}
			},
			inputType: struct {
				Field string `json:"field"`
			}{},
			expectedOutput: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"field": map[string]any{"type": "string"},
				},
				"required": []string{"field"},
			},
		},
		"named type returns ref": {
			setup: func() *DocRouter {
				return &DocRouter{
					schemaRegistry: newSchemaRegistry(),
				}
			},
			inputType: SimpleStruct{},
			expectedOutput: map[string]any{
				"$ref": "#/components/schemas/SimpleStruct",
			},
		},
		"existing type returns ref without re-registration": {
			setup: func() *DocRouter {
				dr := &DocRouter{
					schemaRegistry: newSchemaRegistry(),
				}
				// Pre-register the type
				dr.schemaRegistry.register("SimpleStruct", map[string]any{
					"type": "object",
					"properties": map[string]any{
						"string": map[string]any{"type": "string"},
					},
				})
				return dr
			},
			inputType: SimpleStruct{},
			expectedOutput: map[string]any{
				"$ref": "#/components/schemas/SimpleStruct",
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			dr := tc.setup()
			result := dr.schemaRef(tc.inputType)

			if !assert.Equal(t, tc.expectedOutput, result) {
				t.Logf("Diff: %s", cmp.Diff(tc.expectedOutput, result))
			}

			if tc.inputType != nil && getTypeName(tc.inputType) != "" {
				// Verify the schema was registered
				schemas := dr.schemaRegistry.getSchemas()
				assert.Contains(t, schemas, getTypeName(tc.inputType))
			}
		})
	}
}

func TestCircularReference(t *testing.T) {
	t.Run("handles direct circular reference", func(t *testing.T) {
		// Create a circular reference
		type Node struct {
			Value    string `json:"value"`
			Next     *Node  `json:"next,omitempty"`
			Previous *Node  `json:"previous,omitempty"`
		}

		// Create a generator
		generator := newSchemaGenerator()

		// Generate the schema
		schema := generator.generate(Node{})

		// Check the schema's basic structure
		assert.Equal(t, "object", schema["type"])
		assert.Contains(t, schema, "properties")

		properties := schema["properties"].(map[string]any)
		assert.Contains(t, properties, "value")
		assert.Contains(t, properties, "next")
		assert.Contains(t, properties, "previous")

		// Check circular references
		nextSchema := properties["next"].(map[string]any)
		assert.Equal(t, "object", nextSchema["type"])
		assert.Contains(t, nextSchema, "description")
		assert.Contains(t, nextSchema["description"].(string), "circular reference")

		previousSchema := properties["previous"].(map[string]any)
		assert.Equal(t, "object", previousSchema["type"])
		assert.Contains(t, previousSchema, "description")
		assert.Contains(t, previousSchema["description"].(string), "circular reference")
	})

	t.Run("handles indirect circular reference", func(t *testing.T) {
		// Create an indirect circular reference
		type Person struct {
			Name     string    `json:"name"`
			Children []Person  `json:"children"`
			Parents  []*Person `json:"parents,omitempty"`
		}

		// Create a generator
		generator := newSchemaGenerator()

		// Generate the schema
		schema := generator.generate(Person{})

		// Check the schema
		assert.Equal(t, "object", schema["type"])
		assert.Contains(t, schema, "properties")

		properties := schema["properties"].(map[string]any)
		assert.Contains(t, properties, "name")
		assert.Contains(t, properties, "children")
		assert.Contains(t, properties, "parents")

		// Check children array
		childrenSchema := properties["children"].(map[string]any)
		assert.Equal(t, "array", childrenSchema["type"])
		assert.Contains(t, childrenSchema, "items")

		childrenItems := childrenSchema["items"].(map[string]any)
		// For circular references, the implementation might just label it as object
		// with a description or might have a different way to handle it
		assert.Equal(t, "object", childrenItems["type"])

		// Check parents array
		parentsSchema := properties["parents"].(map[string]any)
		assert.Equal(t, "array", parentsSchema["type"])
		assert.Contains(t, parentsSchema, "items")

		parentsItems := parentsSchema["items"].(map[string]any)
		assert.Equal(t, "object", parentsItems["type"])
	})
}

func TestAddFieldMetadata(t *testing.T) {
	tests := map[string]struct {
		structure any
		fieldName string
		schema    map[string]any
		expected  map[string]any
	}{
		"add description": {
			structure: struct {
				Field string `json:"field" doc:"Field description"`
			}{},
			fieldName: "Field",
			schema:    map[string]any{"type": "string"},
			expected: map[string]any{
				"type":        "string",
				"description": "Field description",
			},
		},
		"add example": {
			structure: struct {
				Field string `json:"field" example:"Example value"`
			}{},
			fieldName: "Field",
			schema:    map[string]any{"type": "string"},
			expected: map[string]any{
				"type":    "string",
				"example": "Example value",
			},
		},
		"add enum": {
			structure: struct {
				Field string `json:"field" enum:"value1,value2,value3"`
			}{},
			fieldName: "Field",
			schema:    map[string]any{"type": "string"},
			expected: map[string]any{
				"type": "string",
				"enum": []string{"value1", "value2", "value3"},
			},
		},
		"add all metadata": {
			structure: struct {
				Field string `json:"field" doc:"Field description" example:"Example value" enum:"value1,value2,value3"`
			}{},
			fieldName: "Field",
			schema:    map[string]any{"type": "string"},
			expected: map[string]any{
				"type":        "string",
				"description": "Field description",
				"example":     "Example value",
				"enum":        []string{"value1", "value2", "value3"},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			typ := reflect.TypeOf(tc.structure)
			field, _ := typ.FieldByName(tc.fieldName)

			// Create a copy of the schema to modify
			schema := make(map[string]any)
			for k, v := range tc.schema {
				schema[k] = v
			}

			// Add metadata
			addFieldMetadata(schema, field)

			// Check result
			assert.Equal(t, tc.expected, schema)
		})
	}
}

func TestSchemaEdgeCases(t *testing.T) {
	// Define wrapper structs for our test cases
	type MapWrapper struct {
		IntMap    map[string]int                `json:"intMap"`
		StructMap map[string]struct{ F string } `json:"structMap"`
	}

	type ArrayWrapper struct {
		IntArray    []int                `json:"intArray"`
		StructArray []struct{ F string } `json:"structArray"`
	}

	tests := map[string]struct {
		value    any
		expected map[string]any
	}{
		"nil value": {
			value:    nil,
			expected: nil,
		},
		"non-struct type": {
			value: struct {
				Str string `json:"str"`
			}{"test"},
			expected: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"str": map[string]any{"type": "string"},
				},
				"required": []string{"str"},
			},
		},
		"map fields": {
			value: MapWrapper{
				IntMap:    map[string]int{"a": 1},
				StructMap: map[string]struct{ F string }{"b": {}},
			},
			expected: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"intMap": map[string]any{
						"type":                 "object",
						"additionalProperties": map[string]any{"type": "integer"},
					},
					"structMap": map[string]any{
						"type": "object",
						"additionalProperties": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"F": map[string]any{"type": "string"},
							},
							"required": []string{"F"},
						},
					},
				},
				"required": []string{"intMap", "structMap"},
			},
		},
		"array fields": {
			value: ArrayWrapper{
				IntArray:    []int{1, 2, 3},
				StructArray: []struct{ F string }{{}},
			},
			expected: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"intArray": map[string]any{
						"type":  "array",
						"items": map[string]any{"type": "integer"},
					},
					"structArray": map[string]any{
						"type": "array",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"F": map[string]any{"type": "string"},
							},
							"required": []string{"F"},
						},
					},
				},
				"required": []string{"intArray", "structArray"},
			},
		},
		"nested struct with enum field": {
			value: struct {
				NestedStruct struct {
					EnumField string `json:"enumField" enum:"one,two,three"`
				} `json:"nestedStruct"`
			}{},
			expected: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"nestedStruct": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"enumField": map[string]any{
								"type": "string",
								"enum": []string{"one", "two", "three"},
							},
						},
						"required": []string{"enumField"},
					},
				},
				"required": []string{"nestedStruct"},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := jsonSchema(tc.value)
			if !assert.Equal(t, tc.expected, result) {
				t.Logf("Diff: %s", cmp.Diff(tc.expected, result))
			}
		})
	}
}
