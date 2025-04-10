package router

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"
)

// schemaGenerator handles the conversion of Go types to JSON Schema
type schemaGenerator struct {
	// processed tracks types already processed to detect circular references
	processed map[reflect.Type]bool
}

// newSchemaGenerator creates a new schema generator
func newSchemaGenerator() *schemaGenerator {
	return &schemaGenerator{
		processed: make(map[reflect.Type]bool),
	}
}

// generate converts a Go type to a JSON Schema
func (g *schemaGenerator) generate(t any) map[string]any {
	if t == nil {
		return nil
	}

	typ := reflect.TypeOf(t)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	// handle non-struct types
	if typ.Kind() != reflect.Struct {
		return basicTypeSchema(typ.Kind())
	}

	// handle circular references
	if g.processed[typ] {
		return map[string]any{
			"type":        "object",
			"description": "circular reference to " + typ.Name(),
		}
	}

	// mark as processed and process the type
	g.processed[typ] = true
	schema := g.processStruct(typ)
	delete(g.processed, typ)

	return schema
}

// processStruct converts a struct type to a JSON Schema
func (g *schemaGenerator) processStruct(typ reflect.Type) map[string]any {
	properties := make(map[string]any)
	required := []string{}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		// get field name from json tag or field name
		name, isRequired := parseJsonTag(jsonTag, field.Name)
		if isRequired {
			required = append(required, name)
		}

		// process field schema
		fieldSchema := g.processField(field)
		if fieldSchema != nil {
			properties[name] = fieldSchema
		}
	}

	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}

// parseJsonTag extracts name and required status from a json tag
func parseJsonTag(jsonTag, fieldName string) (string, bool) {
	if jsonTag == "" {
		return fieldName, true
	}

	parts := strings.Split(jsonTag, ",")
	name := parts[0]
	if name == "" {
		name = fieldName
	}

	return name, !slices.Contains(parts[1:], "omitempty")
}

// processField converts a struct field to a JSON Schema
func (g *schemaGenerator) processField(field reflect.StructField) map[string]any {
	fieldType := field.Type
	if fieldType.Kind() == reflect.Ptr {
		fieldType = fieldType.Elem()
	}

	// Check for special types first
	switch {
	case fieldType == reflect.TypeOf(time.Time{}):
		return map[string]any{
			"type":   "string",
			"format": "date-time",
		}
	case fieldType == reflect.TypeOf(json.RawMessage{}):
		return map[string]any{
			"type": "object",
		}
	}

	// Then check for basic types
	if schema := basicTypeSchema(fieldType.Kind()); schema != nil {
		addFieldMetadata(schema, field)
		return schema
	}

	// Handle different complex types
	switch fieldType.Kind() {
	case reflect.Struct:
		fieldValue := reflect.New(fieldType).Elem().Interface()
		return g.generate(fieldValue)
	case reflect.Slice, reflect.Array:
		return g.processArrayField(fieldType)
	case reflect.Map:
		return g.processMapField(fieldType)
	default:
		return map[string]any{"type": "object"}
	}
}

// processArrayField handles array and slice fields
func (g *schemaGenerator) processArrayField(fieldType reflect.Type) map[string]any {
	elemType := fieldType.Elem()
	var items map[string]any

	switch {
	case basicTypeSchema(elemType.Kind()) != nil:
		items = basicTypeSchema(elemType.Kind())
	case elemType.Kind() == reflect.Struct:
		elemValue := reflect.New(elemType).Elem().Interface()
		items = g.generate(elemValue)
	default:
		items = map[string]any{"type": "object"}
	}

	return map[string]any{
		"type":  "array",
		"items": items,
	}
}

// processMapField handles map fields
func (g *schemaGenerator) processMapField(fieldType reflect.Type) map[string]any {
	valueType := fieldType.Elem()
	var additionalProperties map[string]any

	switch {
	case basicTypeSchema(valueType.Kind()) != nil:
		additionalProperties = basicTypeSchema(valueType.Kind())
	case valueType.Kind() == reflect.Struct:
		valueObj := reflect.New(valueType).Elem().Interface()
		additionalProperties = g.generate(valueObj)
	default:
		additionalProperties = map[string]any{"type": "object"}
	}

	return map[string]any{
		"type":                 "object",
		"additionalProperties": additionalProperties,
	}
}

// addFieldMetadata adds documentation from struct tags to a schema
func addFieldMetadata(schema map[string]any, field reflect.StructField) {
	if docTag := field.Tag.Get("doc"); docTag != "" {
		schema["description"] = docTag
	}

	if exampleTag := field.Tag.Get("example"); exampleTag != "" {
		schema["example"] = exampleTag
	}

	if enumTag := field.Tag.Get("enum"); enumTag != "" {
		schema["enum"] = strings.Split(enumTag, ",")
	}
}

// basicTypeSchema maps Go basic types to OpenAPI schema types
func basicTypeSchema(kind reflect.Kind) map[string]any {
	switch kind {
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return map[string]any{"type": "integer"}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.String:
		return map[string]any{"type": "string"}
	}
	return nil
}

// jsonSchema generates an OpenAPI schema from a Go type
func jsonSchema(t any) map[string]any {
	return newSchemaGenerator().generate(t)
}

// schemaRef generates a reference to a schema if possible
func (g *OpenAPIGenerator) schemaRef(t any) map[string]any {
	typeName := getTypeName(t)

	// if we can't determine the type name, fall back to inline schema
	if typeName == "" {
		schema := jsonSchema(t)
		extractNestedTypes(schema, "Anonymous", g.schemaRegistry)
		return schema
	}

	// register the schema if not already registered
	if _, exists := g.schemaRegistry.schemas[typeName]; !exists {
		schema := jsonSchema(t)
		g.schemaRegistry.register(typeName, schema)
		extractNestedTypes(schema, typeName, g.schemaRegistry)
	}

	// return a reference to the schema
	return map[string]any{
		"$ref": fmt.Sprintf("#/components/schemas/%s", typeName),
	}
}

// getTypeName extracts the Go type name from an interface
func getTypeName(t any) string {
	if t == nil {
		return ""
	}

	typ := reflect.TypeOf(t)
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	return typ.Name() // returns "" for anonymous structs
}

// extractNestedTypes processes a schema to identify nested types that should be extracted
func extractNestedTypes(schema map[string]any, path string, registry *schemaRegistry) {
	// only process object schemas
	if schema["type"] != "object" {
		return
	}

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		return
	}

	for propName, propSchema := range props {
		propSchemaMap, ok := propSchema.(map[string]any)
		if !ok {
			continue
		}

		// Capitalize first letter of property name for type name
		propertyTitle := strings.Title(propName)
		typeName := path + propertyTitle

		// handle object properties
		if propSchemaMap["type"] == "object" && propSchemaMap["properties"] != nil {
			registry.register(typeName, propSchemaMap)
			props[propName] = map[string]any{
				"$ref": fmt.Sprintf("#/components/schemas/%s", typeName),
			}
			extractNestedTypes(propSchemaMap, typeName, registry)
		}

		// handle array properties
		if propSchemaMap["type"] == "array" {
			if items, ok := propSchemaMap["items"].(map[string]any); ok {
				if items["type"] == "object" && items["properties"] != nil {
					itemTypeName := typeName + "Item"
					registry.register(itemTypeName, items)
					propSchemaMap["items"] = map[string]any{
						"$ref": fmt.Sprintf("#/components/schemas/%s", itemTypeName),
					}
					extractNestedTypes(items, itemTypeName, registry)
				}
			}
		}
	}
}
