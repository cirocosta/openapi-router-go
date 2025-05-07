// package api provides the HTTP API for the application
package api

import (
	"context"
	"net/http"

	"github.com/cirocosta/openapi-router-go/internal/model"
	"github.com/cirocosta/openapi-router-go/pkg/router"
)

// TodoService defines the minimal interface needed by the API
type TodoService interface {
	// ListTodos returns all todos
	ListTodos(ctx context.Context) ([]model.Todo, error)

	// GetTodo returns a todo by ID
	GetTodo(ctx context.Context, id string) (model.Todo, error)

	// CreateTodo creates a new todo
	CreateTodo(ctx context.Context, req model.CreateTodoRequest) (model.Todo, error)

	// UpdateTodo updates an existing todo
	UpdateTodo(ctx context.Context, id string, req model.UpdateTodoRequest) (model.Todo, error)

	// DeleteTodo deletes a todo
	DeleteTodo(ctx context.Context, id string) error
}

// errorSchema is used for documentation of error responses
type errorSchema struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// API holds the components needed to register routes
type API struct {
	router      *router.DocRouter
	todoHandler *TodoHandler
}

// NewRouter creates a new router with all routes configured
func NewRouter(todoService TodoService) *router.DocRouter {
	r := router.NewDocRouter("Sample API",
		"A sample API using the custom router wrapper",
		"1.0.0",
	)

	// Add middlewares
	r.Use(loggerMiddleware)
	r.Use(recovererMiddleware)

	api := &API{router: r, todoHandler: NewTodoHandler(todoService)}

	// Define routes
	api.registerRoutes()

	return r
}

// registerRoutes configures all API routes with documentation
func (api *API) registerRoutes() {
	// Error schema for documentation
	errSchema := &errorSchema{}

	api.router = api.router.WithServer("https://api.hellofresh.com/v1", "Production server").
		WithServer("https://api-staging.hellofresh.com/v1", "Staging server").
		WithTag("Todo", "Operations related to todo items").
		WithTag("Core", "Core API endpoints").
		WithBearerAuth()

	// Home and health routes with declarative API
	api.router.Route("GET", "/", homeHandler).
		WithName("Home").
		WithDescription("Home page").
		WithResponse(nil).
		WithTags("Core").
		Register()

	api.router.Route("GET", "/health", healthHandler).
		WithName("Health Check").
		WithDescription("API health check endpoint").
		WithResponse(nil).
		WithTags("Core").
		Register()

	// Todo routes with new declarative API
	api.router.Route("GET", "/todos", api.todoHandler.ListTodos).
		WithName("List Todos").
		WithDescription("Get all todo items").
		WithResponse(&model.TodoListResponse{}).
		WithErrorResponse("400", "Bad Request", errSchema,
			router.Example{
				ContentType: "application/json",
				Value:       `{"code": 400, "message": "invalid query parameters"}`,
			}).
		WithErrorResponse("401", "Unauthorized", errSchema,
			router.Example{
				ContentType: "application/json",
				Value:       `{"code": 401, "message": "authentication required"}`,
			}).
		WithErrorResponse("500", "Internal Server Error", errSchema).
		WithTags("Todos").
		WithSecurity().
		Register()

	api.router.Route("POST", "/todos", api.todoHandler.CreateTodo).
		WithName("Create Todo").
		WithDescription("Create a new todo item").
		WithRequest(&model.CreateTodoRequest{}).
		WithResponse(&model.TodoResponse{}).
		WithErrorResponse("400", "Bad Request", errSchema,
			router.Example{
				ContentType: "application/json",
				Value:       `{"code": 400, "message": "invalid request format"}`,
			}).
		WithErrorResponse("401", "Unauthorized", errSchema).
		WithErrorResponse("422", "Unprocessable Entity", errSchema,
			router.Example{
				ContentType: "application/json",
				Value:       `{"code": 422, "message": "title is required"}`,
			}).
		WithTags("Todos").
		WithSecurity().
		Register()

	api.router.Route("GET", "/todos/{id}", api.todoHandler.GetTodo).
		WithName("Get Todo").
		WithDescription("Get a todo item by ID").
		WithResponse(&model.TodoResponse{}).
		WithErrorResponse("400", "Bad Request", errSchema).
		WithErrorResponse("401", "Unauthorized", errSchema).
		WithErrorResponse("404", "Not Found", errSchema,
			router.Example{
				ContentType: "application/json",
				Value:       `{"code": 404, "message": "todo item not found"}`,
			}).
		WithTags("Todos").
		WithSecurity().
		Register()

	api.router.Route("PUT", "/todos/{id}", api.todoHandler.UpdateTodo).
		WithName("Update Todo").
		WithDescription("Update a todo item").
		WithRequest(&model.UpdateTodoRequest{}).
		WithResponse(&model.TodoResponse{}).
		WithErrorResponse("400", "Bad Request", errSchema).
		WithErrorResponse("401", "Unauthorized", errSchema).
		WithErrorResponse("404", "Not Found", errSchema).
		WithErrorResponse("422", "Unprocessable Entity", errSchema).
		WithTags("Todos").
		WithSecurity().
		Register()

	api.router.Route("DELETE", "/todos/{id}", api.todoHandler.DeleteTodo).
		WithName("Delete Todo").
		WithDescription("Delete a todo item").
		WithErrorResponse("400", "Bad Request", errSchema).
		WithErrorResponse("401", "Unauthorized", errSchema).
		WithErrorResponse("404", "Not Found", errSchema).
		WithTags("Todos").
		WithSecurity().
		Register()
}

// homeHandler handles the home page
func homeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("Welcome to the Todo API"))
}

// healthHandler handles the health check endpoint
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}
