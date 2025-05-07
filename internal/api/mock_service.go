package api

import (
	"context"

	"github.com/cirocosta/openapi-router-go/internal/model"
)

// MockTodoService is a mock implementation of TodoService that does nothing
// and is used solely for OpenAPI documentation generation
type MockTodoService struct{}

// NewMockTodoService creates a new mock todo service
func NewMockTodoService() *MockTodoService {
	return &MockTodoService{}
}

// ListTodos implements TodoService
func (s *MockTodoService) ListTodos(ctx context.Context) ([]model.Todo, error) {
	return nil, nil
}

// GetTodo implements TodoService
func (s *MockTodoService) GetTodo(ctx context.Context, id string) (model.Todo, error) {
	return model.Todo{}, nil
}

// CreateTodo implements TodoService
func (s *MockTodoService) CreateTodo(ctx context.Context, req model.CreateTodoRequest) (model.Todo, error) {
	return model.Todo{}, nil
}

// UpdateTodo implements TodoService
func (s *MockTodoService) UpdateTodo(ctx context.Context, id string, req model.UpdateTodoRequest) (model.Todo, error) {
	return model.Todo{}, nil
}

// DeleteTodo implements TodoService
func (s *MockTodoService) DeleteTodo(ctx context.Context, id string) error {
	return nil
}
