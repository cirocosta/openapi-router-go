package router

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Simple test types for verifying schema generation
type SimpleType struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type NestedType struct {
	ID         string     `json:"id"`
	Properties SimpleType `json:"properties"`
}

// Circular reference test types
type Parent struct {
	Name     string  `json:"name"`
	Children []Child `json:"children,omitempty"`
}

type Child struct {
	Name   string  `json:"name"`
	Parent *Parent `json:"parent,omitempty"`
}

// Array and map test types
type ArrayType struct {
	Tags  []string     `json:"tags"`
	Items []SimpleType `json:"items"`
}

type MapType struct {
	Properties map[string]string     `json:"properties"`
	Objects    map[string]SimpleType `json:"objects"`
}

// API types for end-to-end testing
type UserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password,omitempty"`
}

type UserResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
}

type UserList struct {
	Users []UserResponse `json:"users"`
	Total int            `json:"total"`
}

func TestSchemaRef(t *testing.T) {
	t.Parallel()

	routes := []RouteInfo{}
	generator := NewOpenAPIGenerator("Test API", "Test Description", "1.0.0", routes)

	// Test simple type reference
	t.Run("simple type", func(t *testing.T) {
		t.Parallel()

		actual := generator.schemaRef(SimpleType{})
		expected := map[string]any{
			"$ref": "#/components/schemas/SimpleType",
		}

		if diff := cmp.Diff(expected, actual); diff != "" {
			t.Errorf("schema ref mismatch (-want +got):\n%s", diff)
		}

		// Verify the schema was registered
		schemas := generator.schemaRegistry.getSchemas()
		_, exists := schemas["SimpleType"]
		assert.True(t, exists, "SimpleType schema should be registered")
	})

	// Test nested type reference and extraction
	t.Run("nested type", func(t *testing.T) {
		t.Parallel()

		actual := generator.schemaRef(NestedType{})
		expected := map[string]any{
			"$ref": "#/components/schemas/NestedType",
		}

		if diff := cmp.Diff(expected, actual); diff != "" {
			t.Errorf("schema ref mismatch (-want +got):\n%s", diff)
		}

		// Verify schemas are registered
		schemas := generator.schemaRegistry.getSchemas()

		// Check if NestedType is registered
		_, exists := schemas["NestedType"]
		assert.True(t, exists, "NestedType schema should be registered")

		// Check if extracted type is registered
		_, exists = schemas["NestedTypeProperties"]
		assert.True(t, exists, "NestedTypeProperties schema should be registered")
	})
}

func TestCustomResponses(t *testing.T) {
	t.Parallel()

	// Create sample routes
	routes := []RouteInfo{
		{
			Method:       "GET",
			Path:         "/users/{id}",
			Name:         "Get User",
			Description:  "Get a user by ID",
			ResponseType: UserResponse{},
		},
		{
			Method:       "POST",
			Path:         "/users",
			Name:         "Create User",
			Description:  "Create a new user",
			RequestType:  UserRequest{},
			ResponseType: UserResponse{},
		},
	}

	// Create generator
	generator := NewOpenAPIGenerator("Test API", "API for testing", "1.0.0", routes)

	// Register custom responses
	errorResponse := map[string]any{
		"description": "Error occurred",
		"content": map[string]any{
			"application/json": map[string]any{
				"schema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"code": map[string]any{
							"type": "integer",
						},
						"message": map[string]any{
							"type": "string",
						},
					},
				},
			},
		},
	}

	notFoundResponse := map[string]any{
		"description": "Resource not found",
		"content": map[string]any{
			"application/json": map[string]any{
				"schema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"message": map[string]any{
							"type": "string",
						},
					},
				},
			},
		},
	}

	// Register the responses
	generator.RegisterResponse("Error", errorResponse)
	generator.RegisterResponse("NotFound", notFoundResponse)

	// Associate responses with routes
	generator.RegisterRouteResponse("/users/{id}", "GET", "404", "NotFound")
	generator.RegisterRouteResponse("/users/{id}", "GET", "500", "Error")
	generator.RegisterRouteResponse("/users", "POST", "400", "Error")

	// Generate spec
	spec := generator.Generate()

	// Test components section
	t.Run("components", func(t *testing.T) {
		t.Parallel()

		components, ok := spec["components"].(map[string]any)
		require.True(t, ok, "Components should be a map")

		// Check if responses exist at all
		require.Contains(t, components, "responses", "Responses section should exist in components")

		// We know the responses exist but the type is complex due to json serialization
		// Instead of casting, we'll compare directly using json serialization
		expectedJSON, err := json.Marshal(map[string]any{
			"Error":    errorResponse,
			"NotFound": notFoundResponse,
		})
		require.NoError(t, err, "Failed to marshal expected responses")

		actualJSON, err := json.Marshal(components["responses"])
		require.NoError(t, err, "Failed to marshal actual responses")

		// Compare the JSON strings for equality - this is more flexible with types
		var expected, actual any
		err = json.Unmarshal(expectedJSON, &expected)
		require.NoError(t, err, "Failed to unmarshal expected responses")

		err = json.Unmarshal(actualJSON, &actual)
		require.NoError(t, err, "Failed to unmarshal actual responses")

		// Use cmp to check equality
		if diff := cmp.Diff(expected, actual); diff != "" {
			t.Errorf("responses mismatch (-want +got):\n%s", diff)
		}
	})

	// Test user path responses
	t.Run("path responses", func(t *testing.T) {
		t.Parallel()

		paths, ok := spec["paths"].(map[string]any)
		require.True(t, ok, "Paths should be a map")

		// Check GET /users/{id} endpoint
		t.Run("GET /users/{id}", func(t *testing.T) {
			t.Parallel()

			userPath, ok := paths["/users/{id}"].(map[string]any)
			require.True(t, ok, "/users/{id} path should exist")

			getOp, ok := userPath["get"].(map[string]any)
			require.True(t, ok, "GET operation should exist")

			responses, ok := getOp["responses"].(map[string]any)
			require.True(t, ok, "Responses should be a map")

			// Check 404 response
			resp404, ok := responses["404"].(map[string]any)
			require.True(t, ok, "404 response should exist")

			expected404 := map[string]any{
				"$ref": "#/components/responses/NotFound",
			}
			if diff := cmp.Diff(expected404, resp404); diff != "" {
				t.Errorf("404 response mismatch (-want +got):\n%s", diff)
			}

			// Check 500 response
			resp500, ok := responses["500"].(map[string]any)
			require.True(t, ok, "500 response should exist")

			expected500 := map[string]any{
				"$ref": "#/components/responses/Error",
			}
			if diff := cmp.Diff(expected500, resp500); diff != "" {
				t.Errorf("500 response mismatch (-want +got):\n%s", diff)
			}
		})

		// Check POST /users endpoint
		t.Run("POST /users", func(t *testing.T) {
			t.Parallel()

			userPath, ok := paths["/users"].(map[string]any)
			require.True(t, ok, "/users path should exist")

			postOp, ok := userPath["post"].(map[string]any)
			require.True(t, ok, "POST operation should exist")

			responses, ok := postOp["responses"].(map[string]any)
			require.True(t, ok, "Responses should be a map")

			// Check 400 response
			resp400, ok := responses["400"].(map[string]any)
			require.True(t, ok, "400 response should exist")

			expected400 := map[string]any{
				"$ref": "#/components/responses/Error",
			}
			if diff := cmp.Diff(expected400, resp400); diff != "" {
				t.Errorf("400 response mismatch (-want +got):\n%s", diff)
			}
		})
	})
}

func TestEndToEndOpenAPIGeneration(t *testing.T) {
	t.Parallel()

	// Create sample routes
	routes := []RouteInfo{
		{
			Method:       "GET",
			Path:         "/users",
			Name:         "List Users",
			Description:  "Get a list of all users",
			ResponseType: UserList{},
		},
		{
			Method:       "POST",
			Path:         "/users",
			Name:         "Create User",
			Description:  "Create a new user",
			RequestType:  UserRequest{},
			ResponseType: UserResponse{},
		},
	}

	// Generate OpenAPI spec
	generator := NewOpenAPIGenerator("Test API", "API for testing", "1.0.0", routes)
	spec := generator.Generate()

	// Check basic structure
	t.Run("basic structure", func(t *testing.T) {
		t.Parallel()

		// Check OpenAPI version
		assert.Equal(t, "3.0.0", spec["openapi"], "OpenAPI version should be 3.0.0")

		// Check info section
		info, ok := spec["info"].(map[string]any)
		require.True(t, ok, "Info section should exist")
		assert.Equal(t, "Test API", info["title"], "API title should match")
		assert.Equal(t, "API for testing", info["description"], "API description should match")
		assert.Equal(t, "1.0.0", info["version"], "API version should match")

		// Check paths exist
		paths, ok := spec["paths"].(map[string]any)
		require.True(t, ok, "Paths section should exist")
		assert.Contains(t, paths, "/users", "Paths should contain /users endpoint")
	})

	// Check serialization works
	t.Run("serialization", func(t *testing.T) {
		t.Parallel()

		data, err := json.Marshal(spec)
		assert.NoError(t, err, "OpenAPI spec should serialize without error")
		assert.NotEmpty(t, data, "Serialized data should not be empty")
	})
}

func TestPathParameters(t *testing.T) {
	t.Parallel()

	t.Run("extract path params", func(t *testing.T) {
		t.Parallel()

		tests := []struct {
			path     string
			expected []string
		}{
			{
				path:     "/users",
				expected: []string{},
			},
			{
				path:     "/users/{id}",
				expected: []string{"id"},
			},
			{
				path:     "/users/{userId}/posts/{postId}",
				expected: []string{"userId", "postId"},
			},
		}

		for _, tt := range tests {
			tt := tt
			t.Run(tt.path, func(t *testing.T) {
				t.Parallel()

				actual := extractPathParams(tt.path)

				// Handle nil vs empty slice
				if len(tt.expected) == 0 && (actual == nil || len(actual) == 0) {
					// Both are effectively empty, test passes
					return
				}

				if diff := cmp.Diff(tt.expected, actual); diff != "" {
					t.Errorf("path params mismatch (-want +got):\n%s", diff)
				}
			})
		}
	})

	t.Run("parameter generation", func(t *testing.T) {
		t.Parallel()

		// Create route with parameters
		route := RouteInfo{
			Method:       "GET",
			Path:         "/users/{id}/posts/{postId}",
			Name:         "Get Post",
			Description:  "Get a post for a specific user",
			ResponseType: map[string]any{},
		}

		// Generate spec
		generator := NewOpenAPIGenerator("Test API", "API for testing", "1.0.0", []RouteInfo{route})
		spec := generator.Generate()

		// Get path operation
		paths, ok := spec["paths"].(map[string]any)
		require.True(t, ok, "Paths should exist")

		path, ok := paths["/users/{id}/posts/{postId}"].(map[string]any)
		require.True(t, ok, "Path should exist")

		getOp, ok := path["get"].(map[string]any)
		require.True(t, ok, "GET operation should exist")

		// Check parameters
		params, ok := getOp["parameters"].([]any)
		require.True(t, ok, "Parameters should exist")
		assert.Len(t, params, 2, "Should have 2 parameters")

		// Check parameter names
		paramNames := make(map[string]bool)
		for _, p := range params {
			param, ok := p.(map[string]any)
			require.True(t, ok, "Parameter should be a map")

			name, ok := param["name"].(string)
			require.True(t, ok, "Parameter name should be a string")
			paramNames[name] = true

			// Check basic parameter properties
			assert.Equal(t, "path", param["in"], "Parameter should be in path")
			assert.Equal(t, true, param["required"], "Parameter should be required")
		}

		assert.Contains(t, paramNames, "id", "Parameters should include 'id'")
		assert.Contains(t, paramNames, "postId", "Parameters should include 'postId'")
	})
}
