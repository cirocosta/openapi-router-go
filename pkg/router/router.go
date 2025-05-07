// package router provides a router wrapper that captures documentation data
// and generates OpenAPI specifications
package router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
)

// RouteResponse represents a documented response for a specific HTTP status code
type RouteResponse struct {
	StatusCode  string    // HTTP status code (e.g., "200", "400")
	Description string    // Description of the response
	Schema      any       // Response schema/type (optional)
	Examples    []Example // Example responses (optional)
}

// Example represents an example response for documentation
type Example struct {
	ContentType string // Content type of the example (e.g., "application/json")
	Value       string // Example value as string
}

// RouteInfo stores documentation for a route
type RouteInfo struct {
	Method       string                   // HTTP method (GET, POST, etc.)
	Path         string                   // URL path
	Name         string                   // Friendly name for the endpoint
	Description  string                   // Description of what the endpoint does
	Handler      http.Handler             // The actual handler function
	RequestType  any                      // Example request type (for schema generation)
	ResponseType any                      // Example success response type (for schema generation)
	Responses    map[string]RouteResponse // Map of HTTP status codes to responses
	Tags         []string                 // Tags for grouping endpoints
	Secured      bool                     // Whether this route requires authentication
}

// Server represents an OpenAPI server configuration
type Server struct {
	URL         string // Server URL
	Description string // Server description
}

// TagInfo represents an OpenAPI tag description
type TagInfo struct {
	Name        string // Tag name
	Description string // Tag description
}

// RouteConfig is a builder for route configuration
type RouteConfig struct {
	router       *DocRouter
	method       string
	path         string
	handler      http.HandlerFunc
	name         string
	description  string
	requestType  any
	responseType any
	responses    map[string]RouteResponse
	tags         []string
	secured      bool
}

// DocRouter wraps http.ServeMux to add documentation capabilities and OpenAPI generation
type DocRouter struct {
	mux             *http.ServeMux
	routes          []RouteInfo
	title           string
	description     string
	version         string
	servers         []Server
	tags            []TagInfo
	useBearerAuth   bool
	schemaRegistry  *schemaRegistry
	customResponses map[string]map[string]any
	routeResponses  map[string]map[string]string // Maps routeID -> statusCode -> responseName
}

// NewDocRouter creates a new documented router with optional API metadata
func NewDocRouter(title, description, version string) *DocRouter {
	return &DocRouter{
		mux:             http.NewServeMux(),
		routes:          []RouteInfo{},
		title:           title,
		description:     description,
		version:         version,
		servers:         []Server{},
		tags:            []TagInfo{},
		useBearerAuth:   false,
		schemaRegistry:  newSchemaRegistry(),
		customResponses: make(map[string]map[string]any),
		routeResponses:  make(map[string]map[string]string),
	}
}

// WithServer adds a server to the OpenAPI specification and returns the router for chaining
func (dr *DocRouter) WithServer(url, description string) *DocRouter {
	dr.servers = append(dr.servers, Server{
		URL:         url,
		Description: description,
	})
	return dr
}

// WithBearerAuth enables JWT Bearer token authentication in the OpenAPI specification
func (dr *DocRouter) WithBearerAuth() *DocRouter {
	dr.useBearerAuth = true
	return dr
}

// WithTag adds a tag definition to the OpenAPI specification and returns the router for chaining
func (dr *DocRouter) WithTag(name, description string) *DocRouter {
	dr.tags = append(dr.tags, TagInfo{
		Name:        name,
		Description: description,
	})
	return dr
}

// RegisterResponse adds a custom response pattern that can be referenced in routes
func (dr *DocRouter) RegisterResponse(name string, response map[string]any) {
	dr.customResponses[name] = response
}

// RegisterRouteResponse associates a named response with a specific route and status code
func (dr *DocRouter) RegisterRouteResponse(routePath, method, statusCode, responseName string) {
	routeID := fmt.Sprintf("%s:%s", strings.ToLower(method), routePath)

	// Initialize the map for this route if it doesn't exist
	if _, exists := dr.routeResponses[routeID]; !exists {
		dr.routeResponses[routeID] = make(map[string]string)
	}

	// Associate the response name with the status code for this route
	dr.routeResponses[routeID][statusCode] = responseName
}

// Route starts a route configuration chain
func (dr *DocRouter) Route(method, path string, handler http.HandlerFunc) *RouteConfig {
	return &RouteConfig{
		router:    dr,
		method:    method,
		path:      path,
		handler:   handler,
		responses: make(map[string]RouteResponse),
	}
}

// WithName adds a name to the route
func (rc *RouteConfig) WithName(name string) *RouteConfig {
	rc.name = name
	return rc
}

// WithDescription adds a description to the route
func (rc *RouteConfig) WithDescription(description string) *RouteConfig {
	rc.description = description
	return rc
}

// WithRequest adds a request type to the route
func (rc *RouteConfig) WithRequest(requestType any) *RouteConfig {
	rc.requestType = requestType
	return rc
}

// WithResponse adds a success response type to the route
func (rc *RouteConfig) WithResponse(responseType any) *RouteConfig {
	rc.responseType = responseType
	return rc
}

// WithErrorResponse adds an error response to the route
func (rc *RouteConfig) WithErrorResponse(statusCode, description string, schema any, examples ...Example) *RouteConfig {
	rc.responses[statusCode] = RouteResponse{
		StatusCode:  statusCode,
		Description: description,
		Schema:      schema,
		Examples:    examples,
	}
	return rc
}

// WithTags adds tags to the route
func (rc *RouteConfig) WithTags(tags ...string) *RouteConfig {
	rc.tags = tags
	return rc
}

// WithSecurity marks this route as requiring authentication
func (rc *RouteConfig) WithSecurity() *RouteConfig {
	rc.secured = true
	return rc
}

// Register finalizes the route configuration and registers it with the router
func (rc *RouteConfig) Register() {
	// Create the Go 1.22 pattern with method
	pattern := rc.method + " " + rc.path

	// Register the handler with ServeMux
	rc.router.mux.Handle(pattern, rc.handler)

	// Add documentation
	rc.router.routes = append(rc.router.routes, RouteInfo{
		Method:       rc.method,
		Path:         rc.path,
		Name:         rc.name,
		Description:  rc.description,
		Handler:      rc.handler,
		RequestType:  rc.requestType,
		ResponseType: rc.responseType,
		Responses:    rc.responses,
		Tags:         rc.tags,
		Secured:      rc.secured,
	})
}

// GetRoutes returns all documented routes
func (dr *DocRouter) GetRoutes() []RouteInfo {
	return dr.routes
}

// ServeHTTP makes DocRouter implement the http.Handler interface
func (dr *DocRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	dr.mux.ServeHTTP(w, r)
}

// Use allows adding middleware to the router
func (dr *DocRouter) Use(middleware ...func(http.Handler) http.Handler) {
	// Create a chain of middleware
	var handler http.Handler = dr.mux
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}

	// Create a new mux that forwards requests to the middleware chain
	dr.mux = http.NewServeMux()

	// Add a catch-all handler that forwards to the middleware chain
	dr.mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, r)
	}))
}

// OpenAPI generates an OpenAPI specification from the router
func (dr *DocRouter) OpenAPI() map[string]any {
	spec := map[string]any{
		"openapi": "3.0.0",
		"info": map[string]any{
			"title":       dr.title,
			"description": dr.description,
			"version":     dr.version,
		},
		"paths":      dr.generatePaths(),
		"components": dr.generateComponents(),
	}

	// Add servers if defined
	if len(dr.servers) > 0 {
		servers := make([]map[string]any, 0, len(dr.servers))
		for _, server := range dr.servers {
			serverObj := map[string]any{
				"url":         server.URL,
				"description": server.Description,
			}
			servers = append(servers, serverObj)
		}
		spec["servers"] = servers
	}

	// Add tags if defined
	if len(dr.tags) > 0 {
		tags := make([]map[string]any, 0, len(dr.tags))
		for _, tag := range dr.tags {
			tagObj := map[string]any{
				"name":        tag.Name,
				"description": tag.Description,
			}
			tags = append(tags, tagObj)
		}
		spec["tags"] = tags
	}

	// Add global security requirements for Bearer auth if enabled
	if dr.useBearerAuth {
		spec["security"] = []map[string][]string{
			{"bearerAuth": {}},
		}
	}

	return spec
}

// OpenAPIJSON returns the OpenAPI specification as JSON
func (dr *DocRouter) OpenAPIJSON() ([]byte, error) {
	return json.MarshalIndent(dr.OpenAPI(), "", "  ")
}

// generatePaths creates the paths section of the OpenAPI document
func (dr *DocRouter) generatePaths() map[string]any {
	paths := map[string]any{}

	for _, route := range dr.routes {
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
			"responses":   dr.generateResponses(route),
		}

		// Add tags if specified in the route
		if len(route.Tags) > 0 {
			operation["tags"] = route.Tags
		}

		// Add security requirements if route has secured flag
		if route.Secured && dr.useBearerAuth {
			operation["security"] = []map[string][]string{
				{"bearerAuth": {}},
			}
		}

		// Add path parameters if any exist
		if len(pathParams) > 0 {
			operation["parameters"] = generatePathParameters(pathParams)
		}

		// add request body for POST, PUT, PATCH
		if route.RequestType != nil && (method == "post" || method == "put" || method == "patch") {
			operation["requestBody"] = dr.generateRequestBody(route)
		}

		pathItem[method] = operation
	}

	return paths
}

// generateResponses creates response documentation
func (dr *DocRouter) generateResponses(route RouteInfo) map[string]any {
	responses := map[string]any{}

	// Add success response if specified
	if route.ResponseType != nil {
		// Check if the response type is an array/slice
		respType := reflect.TypeOf(route.ResponseType)
		if respType.Kind() == reflect.Slice || respType.Kind() == reflect.Array {
			// For array types, we need to register the array and its element type
			elemType := respType.Elem()

			// First, register the element type if it's a named struct
			var elemTypeName string
			if elemType.Kind() == reflect.Struct && elemType.Name() != "" {
				elemTypeName = elemType.Name()
				// Only register if not already registered
				schemas := dr.schemaRegistry.getSchemas()
				if _, exists := schemas[elemTypeName]; !exists {
					elemInstance := reflect.New(elemType).Elem().Interface()
					dr.schemaRef(elemInstance)
				}
			}

			// Create the array schema reference
			arrayTypeName := fmt.Sprintf("array_%s", elemTypeName)

			// Register the array type if not already registered
			schemas := dr.schemaRegistry.getSchemas()
			if _, exists := schemas[arrayTypeName]; !exists {
				arraySchema := map[string]any{
					"type": "array",
					"items": map[string]any{
						"$ref": fmt.Sprintf("#/components/schemas/%s", elemTypeName),
					},
				}
				dr.schemaRegistry.register(arrayTypeName, arraySchema)
			}

			// Use the array schema reference
			schema := map[string]any{"$ref": fmt.Sprintf("#/components/schemas/%s", arrayTypeName)}

			responses["200"] = map[string]any{
				"description": "Successful response",
				"content": map[string]any{
					"application/json": map[string]any{
						"schema": schema,
					},
				},
			}
		} else {
			// Non-array types handled normally
			schema := dr.schemaRef(route.ResponseType)

			responses["200"] = map[string]any{
				"description": "Successful response",
				"content": map[string]any{
					"application/json": map[string]any{
						"schema": schema,
					},
				},
			}
		}
	}

	// Add documented error responses
	for statusCode, response := range route.Responses {
		// If schema is provided, use it
		if response.Schema != nil {
			schema := dr.schemaRef(response.Schema)

			// Set up the response object
			respObj := map[string]any{
				"description": response.Description,
				"content": map[string]any{
					"application/json": map[string]any{
						"schema": schema,
					},
				},
			}

			// Add examples if provided
			if len(response.Examples) > 0 {
				examples := map[string]any{}
				for i, example := range response.Examples {
					exampleName := fmt.Sprintf("example%d", i+1)
					examples[exampleName] = map[string]any{
						"value": example.Value,
					}
				}

				respContent := respObj["content"].(map[string]any)
				jsonContent := respContent["application/json"].(map[string]any)
				jsonContent["examples"] = examples
			}

			responses[statusCode] = respObj
		} else {
			// Simple response without schema
			responses[statusCode] = map[string]any{
				"description": response.Description,
			}
		}
	}

	// Add custom responses for this route if any are registered
	routeID := fmt.Sprintf("%s:%s", strings.ToLower(route.Method), route.Path)
	if routeResps, exists := dr.routeResponses[routeID]; exists {
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

	// Ensure there's at least one response defined (OpenAPI requirement)
	if len(responses) == 0 {
		// Add a default 200 response if none were defined
		responses["200"] = map[string]any{
			"description": "Successful operation",
		}
	}

	return responses
}

// generateRequestBody creates request body documentation
func (dr *DocRouter) generateRequestBody(route RouteInfo) map[string]any {
	schema := dr.schemaRef(route.RequestType)

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
func (dr *DocRouter) generateComponents() map[string]any {
	components := map[string]any{
		"schemas": dr.schemaRegistry.getSchemas(),
	}

	// Add custom responses section only when we have responses defined
	if len(dr.customResponses) > 0 {
		components["responses"] = dr.customResponses
	}

	// Add security schemes if Bearer auth is enabled
	if dr.useBearerAuth {
		components["securitySchemes"] = map[string]any{
			"bearerAuth": map[string]any{
				"type":         "http",
				"scheme":       "bearer",
				"bearerFormat": "JWT",
				"description":  "JWT token for authentication",
			},
		}
	}

	return components
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
