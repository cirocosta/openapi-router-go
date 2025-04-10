# openapi-router-go

> auto-openapi generation for Go HTTP services

- code-first documentation: define your api in go code and automatically generate documentation
- type safety: schema generation is based on actual go types
- fluent api: intuitive builder pattern for configuring routes
- standard library integration: built on top of go's standard `http.servemux`
- zero external dependencies: relies only on go's standard library

> note: go 1.22+ is a pre-requisite given the use of http.servemux

for instance:

```go
// create a new router
router := router.NewDocRouter()

// define a route with full documentation
router.Route("GET", "/users/{id}", getUserHandler).
    WithName("Get User").
    WithDescription("Retrieve a user by their unique identifier").
    WithResponse(UserResponse{}).
    WithErrorResponse("404", "User not found", ErrorResponse{}, 
        router.Example{
            ContentType: "application/json",
            Value: `{"error": "user not found"}`,
        },
    ).
    WithTags("Users").
    Register()


// generate openapi documentation
generator := router.NewOpenAPIGenerator(
    "User API",
    "API for managing users",
    "1.0.0",
    router.GetRoutes(),
)

data, _ := json.MarshalIndent(generator.Generate(), "", "  ")
_ := os.WriteFile("openapi.json", data, 0644)
```


## route definition workflow

```mermaid
sequenceDiagram
    participant App as Application
    participant DR as DocRouter
    participant RC as RouteConfig
    participant SM as ServeMux
    
    App->>DR: NewDocRouter()
    Note over DR: Creates new router
    
    App->>DR: Route("GET", "/users/{id}", handler)
    DR->>RC: Creates RouteConfig
    Note over RC: Starts building route config
    
    App->>RC: WithName("Get User")
    App->>RC: WithDescription("Get a user by ID")
    App->>RC: WithResponse(UserResponse{})
    App->>RC: WithErrorResponse("404", "User not found", ErrorResponse{})
    App->>RC: WithTags("Users")
    
    App->>RC: Register()
    RC->>SM: Handle("GET /users/{id}", handler)
    RC->>DR: Add RouteInfo to routes collection
    
    Note over App: Later in the application
    
    App->>DR: GetRoutes()
    DR-->>App: Returns []RouteInfo
    App->>+OpenAPIGenerator: Generate()
    OpenAPIGenerator-->>-App: Returns OpenAPI spec
```

## schema gen

```mermaid
flowchart TD
    A[Go Type] --> B[schemaGenerator]
    B --> C{Type Kind?}
    
    C -->|Basic Type| D[basicTypeSchema]
    C -->|Struct| E[processStruct]
    C -->|Array/Slice| F[processArrayField]
    C -->|Map| G[processMapField]
    
    E --> H[Extract Fields]
    H --> I[Parse JSON Tags]
    H --> J[Process Field Types]
    
    J --> K{Field Kind?}
    K -->|Basic| L[basicTypeSchema]
    K -->|Struct| M[Generate Nested Schema]
    K -->|Array| N[processArrayField]
    K -->|Map| O[processMapField]
    
    D --> P[Schema Object]
    E --> P
    F --> P
    G --> P
    
    P --> Q[Add Field Metadata]
    Q --> R[Final Schema]
```

note that here we include special handling for:

- circular references to prevent infinite recursion
- special types like `time.Time` and `json.RawMessage`
- field metadata from struct tags (`json`, `doc`, `example`, `enum`)
- required vs. optional fields (based on `json:"field,omitempty"` tags)

## openapi spec gen

```mermaid
flowchart TD
    A[Routes Collection] --> B[OpenAPIGenerator]
    B --> C[Generate]
    
    C --> D[Generate Paths]
    C --> E[Generate Components]
    
    D --> F[For Each Route]
    F --> G[Extract Path Parameters]
    F --> H[Generate Operation Object]
    H --> I[Add Request Body]
    H --> J[Generate Responses]
    
    E --> K[Extract Schemas]
    E --> L[Register Common Responses]
    
    I --> M[Schema Reference System]
    J --> M
    K --> M
    
    M --> N[Schema Registry]
    N --> O[Reuse Common Types]
    
    O --> P[Final OpenAPI Spec]
```

## License

MIT (do whatever)
