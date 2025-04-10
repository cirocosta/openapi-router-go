package router

import (
	"fmt"
	"strings"
)

// schemaRegistry tracks schema definitions to enable reuse
type schemaRegistry struct {
	schemas map[string]map[string]any
}

// newSchemaRegistry creates a new schema registry
func newSchemaRegistry() *schemaRegistry {
	return &schemaRegistry{
		schemas: make(map[string]map[string]any),
	}
}

// register adds a schema to the registry
func (r *schemaRegistry) register(typeName string, schema map[string]any) {
	r.schemas[typeName] = schema
}

// getSchemas returns all registered schemas
func (r *schemaRegistry) getSchemas() map[string]any {
	result := make(map[string]any)
	for name, schema := range r.schemas {
		result[name] = schema
	}
	return result
}

// OpenAPIGenerator generates OpenAPI specs from route info
type OpenAPIGenerator struct {
	Title           string
	Description     string
	Version         string
	Routes          []RouteInfo
	schemaRegistry  *schemaRegistry
	customResponses map[string]map[string]any
	routeResponses  map[string]map[string]string // Maps routeID -> statusCode -> responseName
}

// NewOpenAPIGenerator creates a new OpenAPI generator
func NewOpenAPIGenerator(title, description, version string, routes []RouteInfo) *OpenAPIGenerator {
	return &OpenAPIGenerator{
		Title:           title,
		Description:     description,
		Version:         version,
		Routes:          routes,
		schemaRegistry:  newSchemaRegistry(),
		customResponses: make(map[string]map[string]any),
		routeResponses:  make(map[string]map[string]string),
	}
}

// RegisterResponse adds a custom response pattern that can be referenced in routes
func (g *OpenAPIGenerator) RegisterResponse(name string, response map[string]any) {
	g.customResponses[name] = response
}

// RegisterRouteResponse associates a named response with a specific route and status code
func (g *OpenAPIGenerator) RegisterRouteResponse(routePath, method, statusCode, responseName string) {
	routeID := fmt.Sprintf("%s:%s", strings.ToLower(method), routePath)

	// Initialize the map for this route if it doesn't exist
	if _, exists := g.routeResponses[routeID]; !exists {
		g.routeResponses[routeID] = make(map[string]string)
	}

	// Associate the response name with the status code for this route
	g.routeResponses[routeID][statusCode] = responseName
}

// Generate creates and returns an OpenAPI specification
func (g *OpenAPIGenerator) Generate() map[string]any {
	spec := map[string]any{
		"openapi": "3.0.0",
		"info": map[string]any{
			"title":       g.Title,
			"description": g.Description,
			"version":     g.Version,
		},
		"paths":      g.generatePaths(),
		"components": g.generateComponents(),
	}

	return spec
}

// extractPathParams gets path parameters from a URL path
func extractPathParams(path string) []string {
	var params []string
	parts := strings.Split(path, "/")

	for _, part := range parts {
		if len(part) > 0 && part[0] == '{' && part[len(part)-1] == '}' {
			// Extract the parameter name without braces
			paramName := part[1 : len(part)-1]
			params = append(params, paramName)
		}
	}

	return params
}

// generatePathParameters creates parameter objects for path parameters
func generatePathParameters(params []string) []any {
	var parameters []any

	for _, param := range params {
		parameters = append(parameters, map[string]any{
			"name":     param,
			"in":       "path",
			"required": true,
			"schema": map[string]any{
				"type": "string",
			},
			"description": fmt.Sprintf("%s parameter", param),
		})
	}

	return parameters
}

// generatePaths creates the paths section of the OpenAPI spec
func (g *OpenAPIGenerator) generatePaths() map[string]any {
	paths := map[string]any{}

	for _, route := range g.Routes {
		// skip if the path contains regex patterns (not easily mappable to OpenAPI)
		if strings.Contains(route.Path, "^") || strings.Contains(route.Path, "(") {
			continue
		}

		// convert Chi path params from {param} to OpenAPI {param}
		path := route.Path

		// add the path if it doesn't exist
		if _, exists := paths[path]; !exists {
			paths[path] = map[string]any{}
		}

		pathItem := paths[path].(map[string]any)
		method := strings.ToLower(route.Method)

		// Extract path parameters
		pathParams := extractPathParams(path)

		operation := map[string]any{
			"summary":     route.Name,
			"description": route.Description,
			"operationId": fmt.Sprintf("%s_%s", method, strings.ReplaceAll(route.Path, "/", "_")),
			"responses":   g.generateResponses(route),
		}

		// Add path parameters if any exist
		if len(pathParams) > 0 {
			operation["parameters"] = generatePathParameters(pathParams)
		}

		// add request body for POST, PUT, PATCH
		if route.RequestType != nil && (method == "post" || method == "put" || method == "patch") {
			operation["requestBody"] = g.generateRequestBody(route)
		}

		pathItem[method] = operation
	}

	return paths
}

// generateResponses creates response documentation
func (g *OpenAPIGenerator) generateResponses(route RouteInfo) map[string]any {
	responses := map[string]any{}

	// Add custom responses from the route definition
	if route.Responses != nil {
		for statusCode, routeResponse := range route.Responses {
			responseContent := map[string]any{}

			// Add schema if available
			if routeResponse.Schema != nil {
				schema := g.schemaRef(routeResponse.Schema)
				responseContent["schema"] = schema
			}

			// Add examples if available
			if len(routeResponse.Examples) > 0 {
				examples := map[string]any{}
				for _, example := range routeResponse.Examples {
					examples[example.ContentType] = map[string]any{
						"value": example.Value,
					}
				}
				if len(examples) > 0 {
					responseContent["examples"] = examples
				}
			}

			// Create response object
			response := map[string]any{
				"description": routeResponse.Description,
			}

			// Add content if we have schema or examples
			if len(responseContent) > 0 {
				response["content"] = map[string]any{
					"application/json": responseContent,
				}
			}

			responses[statusCode] = response
		}
	}

	// Add success response if it wasn't overridden by a custom response
	if _, exists := responses["200"]; !exists {
		if route.ResponseType != nil {
			schema := g.schemaRef(route.ResponseType)

			responses["200"] = map[string]any{
				"description": "successful operation",
				"content": map[string]any{
					"application/json": map[string]any{
						"schema": schema,
					},
				},
			}
		} else {
			// generic success response if no type provided
			responses["200"] = map[string]any{
				"description": "successful operation",
			}
		}
	}

	// Add custom responses for this route if any exist in the global registry
	routeID := fmt.Sprintf("%s:%s", strings.ToLower(route.Method), route.Path)
	if routeResps, exists := g.routeResponses[routeID]; exists {
		for statusCode, responseName := range routeResps {
			// Skip if this status code is already defined in the route's responses
			if _, exists := responses[statusCode]; exists {
				continue
			}

			// Reference the custom response
			responses[statusCode] = map[string]any{
				"$ref": fmt.Sprintf("#/components/responses/%s", responseName),
			}
		}
	}

	return responses
}

// generateRequestBody creates request body documentation
func (g *OpenAPIGenerator) generateRequestBody(route RouteInfo) map[string]any {
	schema := g.schemaRef(route.RequestType)

	return map[string]any{
		"description": fmt.Sprintf("request body for %s", route.Name),
		"required":    true,
		"content": map[string]any{
			"application/json": map[string]any{
				"schema": schema,
			},
		},
	}
}

// generateComponents creates reusable components
func (g *OpenAPIGenerator) generateComponents() map[string]any {
	components := map[string]any{
		"schemas": g.schemaRegistry.getSchemas(),
	}

	// Add custom responses section only when we have responses defined
	if len(g.customResponses) > 0 {
		components["responses"] = g.customResponses
	}

	return components
}
