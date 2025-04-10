// package router provides a router wrapper that captures documentation data
package router

import (
	"net/http"
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
}

// DocRouter wraps http.ServeMux to add documentation capabilities
type DocRouter struct {
	mux    *http.ServeMux
	routes []RouteInfo
}

// NewDocRouter creates a new documented router
func NewDocRouter() *DocRouter {
	return &DocRouter{
		mux:    http.NewServeMux(),
		routes: []RouteInfo{},
	}
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
