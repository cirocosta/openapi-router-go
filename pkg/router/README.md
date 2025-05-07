# Router Package router

The router router package is an enhanced version of the original router that integrates OpenAPI generation capabilities directly into the DocRouter. This integration simplifies API documentation by making the DocRouter the single source of truth for both routing and OpenAPI documentation.

## Key Changes

1. **Integrated OpenAPI Generation**: Added `OpenAPI()` and `OpenAPIJSON()` methods directly to DocRouter
2. **Global Configuration Options**: Moved configuration methods like `WithServer`, `WithTag`, and `WithBearerAuth` to DocRouter
3. **Enhanced Constructor**: Updated to accept API metadata (title, description, version)

## Usage Example

```go
// Create a new router with API metadata
router := router.NewDocRouter(
    "My API",
    "API Description",
    "1.0.0",
)

// Configure OpenAPI document
router = router.WithServer("https://api.example.com/v1", "Production server").
    WithServer("https://api-staging.example.com/v1", "Staging server").
    WithTag("Users", "Operations related to user management").
    WithBearerAuth()

// Define routes with documentation
router.Route("GET", "/users/{id}", getUserHandler).
    WithName("Get User").
    WithDescription("Get a user by ID").
    WithResponse(UserResponse{}).
    WithErrorResponse("404", "User not found", ErrorResponse{}).
    WithTags("Users").
    Register()

// Generate and save the OpenAPI specification
spec, err := router.OpenAPIJSON()
if err != nil {
    log.Fatalf("Error generating OpenAPI spec: %v", err)
}
err = os.WriteFile("openapi.json", spec, 0644)
```

## Benefits

1. **Simplified API**: Users only need to interact with one component (DocRouter)
2. **Single Source of Truth**: Configuration is stored in one place
3. **Improved Coherence**: Router and OpenAPI generation are naturally coupled
4. **Reduced Boilerplate**: No need to pass routes between components

## Migration from v1

To migrate from the original router package to router:

1. Update import path from `github.com/cirocosta/openapi-router-go/pkg/router` to `github.com/cirocosta/openapi-router-go/pkg/router`

2. Update router initialization:

   ```go
   // Before
   router := router.NewDocRouter()
   
   // After
   router := router.NewDocRouter(
       "API Title",
       "API Description",
       "1.0.0",
   )
   ```

3. Move OpenAPI configuration to DocRouter:

   ```go
   // Before
   generator := router.NewOpenAPIGenerator(
       "API Title",
       "API Description",
       "1.0.0",
       router.GetRoutes(),
   )
   generator.WithServer("https://api.example.com", "Production")
            .WithTag("Users", "User operations")
            .WithBearerAuth()
   spec := generator.Generate()
   
   // After
   router = router.WithServer("https://api.example.com", "Production").
       WithTag("Users", "User operations").
       WithBearerAuth()
   spec := router.OpenAPI()
   ```

4. Update response registration if used:

   ```go
   // Before
   generator.RegisterResponse("notFound", notFoundResponse)
   
   // After
   router.RegisterResponse("notFound", notFoundResponse)
   ```

## API Reference

### DocRouter Methods

- `NewDocRouter(title, description, version string) *DocRouter` - Creates a new router
- `WithServer(url, description string) *DocRouter` - Adds a server to the OpenAPI spec
- `WithTag(name, description string) *DocRouter` - Adds a tag definition
- `WithBearerAuth() *DocRouter` - Enables JWT authentication
- `RegisterResponse(name string, response map[string]any)` - Registers a reusable response
- `RegisterRouteResponse(routePath, method, statusCode, responseName string)` - Associates a response with a route
- `OpenAPI() map[string]any` - Generates the OpenAPI specification
- `OpenAPIJSON() ([]byte, error)` - Generates the OpenAPI specification as JSON

### Route Configuration Methods

Same as v1:

- `WithName(name string) *RouteConfig`
- `WithDescription(description string) *RouteConfig`
- `WithRequest(requestType any) *RouteConfig`
- `WithResponse(responseType any) *RouteConfig`
- `WithErrorResponse(statusCode, description string, schema any, examples ...Example) *RouteConfig`
- `WithTags(tags ...string) *RouteConfig`
- `WithSecurity() *RouteConfig`
- `Register()` 