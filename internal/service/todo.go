// package service implements business logic for the application
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/cirocosta/openapi-router-go/internal/model"
	"github.com/cirocosta/openapi-router-go/internal/repository"
)

// TodoService handles business logic for todo operations
type TodoService struct {
	repo repository.TodoRepository
}

// NewTodoService creates a new todo service with the given repository
func NewTodoService(repo repository.TodoRepository) *TodoService {
	return &TodoService{
		repo: repo,
	}
}

// ListTodos returns all todos
func (s *TodoService) ListTodos(ctx context.Context) ([]model.Todo, error) {
	return s.repo.FindAll(ctx)
}

// GetTodo returns a todo by ID
func (s *TodoService) GetTodo(ctx context.Context, id string) (model.Todo, error) {
	if id == "" {
		return model.Todo{}, fmt.Errorf("todo id is required")
	}

	return s.repo.FindByID(ctx, id)
}

// CreateTodo creates a new todo
func (s *TodoService) CreateTodo(ctx context.Context, req model.CreateTodoRequest) (model.Todo, error) {
	// validate input
	if req.Title == "" {
		return model.Todo{}, fmt.Errorf("title is required")
	}

	// create the todo
	now := time.Now()
	todo := model.Todo{
		ID:          fmt.Sprintf("todo-%d", now.UnixNano()),
		Title:       req.Title,
		Description: req.Description,
		Completed:   false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	return s.repo.Create(ctx, todo)
}

// UpdateTodo updates an existing todo
func (s *TodoService) UpdateTodo(ctx context.Context, id string, req model.UpdateTodoRequest) (model.Todo, error) {
	if id == "" {
		return model.Todo{}, fmt.Errorf("todo id is required")
	}

	// get existing todo
	existingTodo, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return model.Todo{}, err
	}

	// apply updates
	if req.Title != "" {
		existingTodo.Title = req.Title
	}
	if req.Description != "" {
		existingTodo.Description = req.Description
	}

	existingTodo.Completed = req.Completed
	existingTodo.UpdatedAt = time.Now()

	return s.repo.Update(ctx, id, existingTodo)
}

// DeleteTodo deletes a todo
func (s *TodoService) DeleteTodo(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("todo id is required")
	}

	return s.repo.Delete(ctx, id)
}
