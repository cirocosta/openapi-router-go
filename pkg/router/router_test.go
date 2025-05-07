package router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
)

// Test types
type Address struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	Country string `json:"country"`
	ZipCode string `json:"zip_code"`
}

type User struct {
	ID       int     `json:"id"`
	Username string  `json:"username"`
	Email    string  `json:"email"`
	Address  Address `json:"address"`
}

type CreateUserRequest struct {
	Username string `json:"username" doc:"Username for the new user"`
	Email    string `json:"email" doc:"Email address for the new user"`
	Password string `json:"password" doc:"Password for the new user"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func noopHandler(w http.ResponseWriter, r *http.Request) {}

func TestNewDocRouter(t *testing.T) {
	title := "Test API"
	description := "API for testing"
	version := "1.0.0"

	router := NewDocRouter(title, description, version)

	assert.NotNil(t, router)
	assert.Equal(t, title, router.title)
	assert.Equal(t, description, router.description)
	assert.Equal(t, version, router.version)
	assert.NotNil(t, router.mux)
	assert.Empty(t, router.routes)
	assert.Empty(t, router.servers)
	assert.Empty(t, router.tags)
	assert.False(t, router.useBearerAuth)
	assert.NotNil(t, router.schemaRegistry)
	assert.Empty(t, router.customResponses)
	assert.Empty(t, router.routeResponses)
}

func TestWithServer(t *testing.T) {
	router := NewDocRouter("Test API", "API for testing", "1.0.0")

	url := "https://api.example.com"
	description := "Production server"

	result := router.WithServer(url, description)

	assert.Equal(t, router, result, "WithServer should return the router for chaining")
	assert.Len(t, router.servers, 1)
	assert.Equal(t, url, router.servers[0].URL)
	assert.Equal(t, description, router.servers[0].Description)
}

func TestWithBearerAuth(t *testing.T) {
	router := NewDocRouter("Test API", "API for testing", "1.0.0")

	result := router.WithBearerAuth()

	assert.Equal(t, router, result, "WithBearerAuth should return the router for chaining")
	assert.True(t, router.useBearerAuth)
}

func TestWithTag(t *testing.T) {
	router := NewDocRouter("Test API", "API for testing", "1.0.0")

	name := "users"
	description := "User operations"

	result := router.WithTag(name, description)

	assert.Equal(t, router, result, "WithTag should return the router for chaining")
	assert.Len(t, router.tags, 1)
	assert.Equal(t, name, router.tags[0].Name)
	assert.Equal(t, description, router.tags[0].Description)
}

func TestRegisterResponse(t *testing.T) {
	router := NewDocRouter("Test API", "API for testing", "1.0.0")

	name := "StandardError"
	response := map[string]any{
		"description": "Standard error response",
		"content": map[string]any{
			"application/json": map[string]any{
				"schema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"code":    map[string]any{"type": "string"},
						"message": map[string]any{"type": "string"},
					},
				},
			},
		},
	}

	router.RegisterResponse(name, response)

	assert.Len(t, router.customResponses, 1)
	assert.Equal(t, response, router.customResponses[name])
}

func TestRegisterRouteResponse(t *testing.T) {
	router := NewDocRouter("Test API", "API for testing", "1.0.0")

	path := "/users"
	method := "GET"
	statusCode := "400"
	responseName := "StandardError"

	router.RegisterRouteResponse(path, method, statusCode, responseName)

	routeID := "get:/users"
	assert.Len(t, router.routeResponses, 1)
	assert.Contains(t, router.routeResponses, routeID)
	assert.Equal(t, responseName, router.routeResponses[routeID][statusCode])
}

func TestRouteConfigChain(t *testing.T) {
	router := NewDocRouter("Test API", "API for testing", "1.0.0")

	var configuredRoutes []RouteInfo
	originalRoutes := router.routes

	// Store the original implementation
	router.routes = configuredRoutes

	// Setup route
	config := router.Route("GET", "/users/{id}", noopHandler).
		WithName("Get User").
		WithDescription("Get a user by ID").
		WithResponse(User{}).
		WithRequest(nil).
		WithErrorResponse("404", "User not found", ErrorResponse{}).
		WithTags("users").
		WithSecurity()

	// Register the route
	config.Register()

	// Restore original routes
	configuredRoutes = router.routes
	router.routes = originalRoutes

	assert.Len(t, configuredRoutes, 1)
	route := configuredRoutes[0]

	assert.Equal(t, "GET", route.Method)
	assert.Equal(t, "/users/{id}", route.Path)
	assert.Equal(t, "Get User", route.Name)
	assert.Equal(t, "Get a user by ID", route.Description)
	assert.NotNil(t, route.Handler)
	assert.IsType(t, User{}, route.ResponseType)
	assert.Nil(t, route.RequestType)
	assert.Len(t, route.Responses, 1)
	assert.Contains(t, route.Responses, "404")
	assert.Equal(t, "User not found", route.Responses["404"].Description)
	assert.Len(t, route.Tags, 1)
	assert.Equal(t, "users", route.Tags[0])
	assert.True(t, route.Secured)
}

// assertOpenAPIMatches is a helper to make OpenAPI testing more declarative
func assertOpenAPIMatches(t *testing.T, router *DocRouter, expectedSpecPath string) {
	t.Helper()

	// Generate OpenAPI spec
	generatedSpec := router.OpenAPI()

	// Load expected spec from testdata
	expectedBytes, err := os.ReadFile(expectedSpecPath)
	assert.NoError(t, err, "Failed to read expected OpenAPI spec file")

	var expectedSpec map[string]any
	err = json.Unmarshal(expectedBytes, &expectedSpec)
	assert.NoError(t, err, "Failed to unmarshal expected OpenAPI spec")

	// Normalize operationId fields in paths
	normalizePaths(generatedSpec)

	// Convert both to JSON for comparison
	expectedJSON, err := json.MarshalIndent(expectedSpec, "", "  ")
	assert.NoError(t, err, "Failed to marshal expected spec to JSON")

	generatedJSON, err := json.MarshalIndent(generatedSpec, "", "  ")
	assert.NoError(t, err, "Failed to marshal generated spec to JSON")

	// Compare JSON strings for better error messages
	if !assert.Equal(t, string(expectedJSON), string(generatedJSON)) {
		// Write the actual generated spec to a file for easier debugging
		debugFilePath := strings.Replace(expectedSpecPath, "expected", "actual", 1)
		err = os.WriteFile(debugFilePath, generatedJSON, 0644)
		assert.NoError(t, err, "Failed to write actual spec to debug file")

		t.Logf("Generated OpenAPI spec doesn't match expected. Actual spec written to %s", debugFilePath)
		t.Logf("Diff: %s", cmp.Diff(string(expectedJSON), string(generatedJSON)))
	}
}

// normalizePaths normalizes operationId fields for deterministic testing
func normalizePaths(spec map[string]any) {
	if paths, ok := spec["paths"].(map[string]any); ok {
		for path, pathItemObj := range paths {
			if pathItem, ok := pathItemObj.(map[string]any); ok {
				for method, opObj := range pathItem {
					if op, ok := opObj.(map[string]any); ok {
						// Normalize operationId to method_path format
						normalizedPath := strings.ReplaceAll(path, "/", "_")
						if normalizedPath == "" {
							normalizedPath = "root"
						}
						op["operationId"] = fmt.Sprintf("%s%s", method, normalizedPath)
					}
				}
			}
		}
	}
}

func TestOpenAPI(t *testing.T) {
	// Create a router with all the features
	router := NewDocRouter("Test API", "API for testing", "1.0.0").
		WithServer("https://api.example.com", "Production server").
		WithBearerAuth().
		WithTag("users", "User operations")

	// Register a custom response
	errorResponse := map[string]any{
		"description": "Standard error response",
		"content": map[string]any{
			"application/json": map[string]any{
				"schema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"code":    map[string]any{"type": "string"},
						"message": map[string]any{"type": "string"},
					},
					"required": []string{"code", "message"},
				},
			},
		},
	}
	router.RegisterResponse("StandardError", errorResponse)

	// Register routes with the response
	router.Route("GET", "/users", noopHandler).
		WithName("List Users").
		WithDescription("Get all users").
		WithResponse([]User{}).
		WithTags("users").
		Register()

	router.Route("POST", "/users", noopHandler).
		WithName("Create User").
		WithDescription("Create a new user").
		WithRequest(CreateUserRequest{}).
		WithResponse(User{}).
		WithErrorResponse("400", "Invalid request", ErrorResponse{}).
		WithTags("users").
		WithSecurity().
		Register()

	router.RegisterRouteResponse("/users", "GET", "500", "StandardError")

	// Use the helper to compare against expected spec
	assertOpenAPIMatches(t, router, "testdata/expected_openapi.json")
}

// TestMinimalOpenAPI tests a minimal API configuration
func TestMinimalOpenAPI(t *testing.T) {
	// Create a minimal router without additional features
	router := NewDocRouter("Minimal API", "Basic API for testing", "1.0.0")

	// Register a single route
	router.Route("GET", "/health", noopHandler).
		WithName("Health Check").
		WithDescription("Check API health").
		Register()

	// Generate and save expected spec if it doesn't exist yet
	expectedPath := "testdata/expected_minimal_openapi.json"
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		spec := router.OpenAPI()
		normalizePaths(spec)
		jsonBytes, err := json.MarshalIndent(spec, "", "  ")
		assert.NoError(t, err)
		err = os.WriteFile(expectedPath, jsonBytes, 0644)
		assert.NoError(t, err)
	}

	// Compare against expected spec
	assertOpenAPIMatches(t, router, expectedPath)
}

func TestOpenAPIJSON(t *testing.T) {
	router := NewDocRouter("Test API", "API for testing", "1.0.0")

	jsonBytes, err := router.OpenAPIJSON()

	assert.NoError(t, err)
	assert.NotEmpty(t, jsonBytes)
	assert.Contains(t, string(jsonBytes), "Test API")
}

func TestExtractPathParams(t *testing.T) {
	tests := map[string]struct {
		path     string
		expected []string
	}{
		"no params": {
			path:     "/users",
			expected: nil,
		},
		"one param": {
			path:     "/users/{id}",
			expected: []string{"id"},
		},
		"multiple params": {
			path:     "/users/{id}/posts/{postId}",
			expected: []string{"id", "postId"},
		},
		"trailing slash": {
			path:     "/users/{id}/",
			expected: []string{"id"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result := extractPathParams(tc.path)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestGeneratePathParameters(t *testing.T) {
	params := []string{"id", "name"}

	result := generatePathParameters(params)

	assert.Len(t, result, 2)

	param1 := result[0].(map[string]any)
	assert.Equal(t, "id", param1["name"])
	assert.Equal(t, "path", param1["in"])
	assert.True(t, param1["required"].(bool))

	param2 := result[1].(map[string]any)
	assert.Equal(t, "name", param2["name"])
}

func TestNestedRouteRegister(t *testing.T) {
	router := NewDocRouter("Test API", "API for testing", "1.0.0")

	handler := http.HandlerFunc(noopHandler)

	// Create a route using the builder pattern
	router.Route("GET", "/nested/path", handler).
		WithName("Nested Route").
		WithDescription("A nested route").
		Register()

	// Check that the route was added correctly
	assert.Len(t, router.routes, 1)
	assert.Equal(t, "GET", router.routes[0].Method)
	assert.Equal(t, "/nested/path", router.routes[0].Path)
}

func TestGenerateResponses(t *testing.T) {
	router := NewDocRouter("Test API", "API for testing", "1.0.0")

	// Create a route with various responses
	route := RouteInfo{
		Method:       "GET",
		Path:         "/test",
		ResponseType: User{},
		Responses: map[string]RouteResponse{
			"400": {
				StatusCode:  "400",
				Description: "Bad Request",
				Schema:      ErrorResponse{},
				Examples: []Example{
					{
						ContentType: "application/json",
						Value:       `{"code":"invalid_input","message":"Invalid input"}`,
					},
				},
			},
			"404": {
				StatusCode:  "404",
				Description: "Not Found",
				Schema:      nil,
			},
		},
	}

	responses := router.generateResponses(route)

	// Check 200 response
	assert.Contains(t, responses, "200")
	resp200 := responses["200"].(map[string]any)
	assert.Equal(t, "Successful response", resp200["description"])
	assert.Contains(t, resp200, "content")

	// Check 400 response with schema and examples
	assert.Contains(t, responses, "400")
	resp400 := responses["400"].(map[string]any)
	assert.Equal(t, "Bad Request", resp400["description"])
	assert.Contains(t, resp400, "content")

	// Content should have examples
	content400 := resp400["content"].(map[string]any)
	jsonContent := content400["application/json"].(map[string]any)
	assert.Contains(t, jsonContent, "examples")

	// Check 404 response without schema
	assert.Contains(t, responses, "404")
	resp404 := responses["404"].(map[string]any)
	assert.Equal(t, "Not Found", resp404["description"])
	assert.NotContains(t, resp404, "content")
}
