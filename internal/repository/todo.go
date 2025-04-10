// package repository provides data access interfaces and implementations
package repository

import (
	"context"
	"sync"
	"time"

	"github.com/cirocosta/openapi-router-go/internal/model"
)

// TodoRepository defines the interface for todo data access
type TodoRepository interface {
	// FindAll returns all todos
	FindAll(ctx context.Context) ([]model.Todo, error)

	// FindByID returns a specific todo by ID
	FindByID(ctx context.Context, id string) (model.Todo, error)

	// Create adds a new todo
	Create(ctx context.Context, todo model.Todo) (model.Todo, error)

	// Update modifies an existing todo
	Update(ctx context.Context, id string, todo model.Todo) (model.Todo, error)

	// Delete removes a todo
	Delete(ctx context.Context, id string) error
}

// InMemoryTodoRepository implements TodoRepository with an in-memory map
type InMemoryTodoRepository struct {
	todos map[string]model.Todo
	mutex sync.RWMutex
}

// NewInMemoryTodoRepository creates a new in-memory todo repository with optional initial data
func NewInMemoryTodoRepository() *InMemoryTodoRepository {
	repo := &InMemoryTodoRepository{
		todos: make(map[string]model.Todo),
		mutex: sync.RWMutex{},
	}

	// add a sample todo
	sampleTodo := model.Todo{
		ID:          "sample-todo-1",
		Title:       "Sample Todo",
		Description: "This is a sample todo item",
		Completed:   false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	repo.todos[sampleTodo.ID] = sampleTodo

	return repo
}

// FindAll returns all todos
func (r *InMemoryTodoRepository) FindAll(ctx context.Context) ([]model.Todo, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	todos := make([]model.Todo, 0, len(r.todos))
	for _, todo := range r.todos {
		todos = append(todos, todo)
	}

	return todos, nil
}

// FindByID returns a specific todo by ID
func (r *InMemoryTodoRepository) FindByID(ctx context.Context, id string) (model.Todo, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	todo, exists := r.todos[id]
	if !exists {
		return model.Todo{}, ErrTodoNotFound{ID: id}
	}

	return todo, nil
}

// Create adds a new todo
func (r *InMemoryTodoRepository) Create(ctx context.Context, todo model.Todo) (model.Todo, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.todos[todo.ID] = todo
	return todo, nil
}

// Update modifies an existing todo
func (r *InMemoryTodoRepository) Update(ctx context.Context, id string, todo model.Todo) (model.Todo, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	_, exists := r.todos[id]
	if !exists {
		return model.Todo{}, ErrTodoNotFound{ID: id}
	}

	// ensure ID doesn't change
	todo.ID = id
	r.todos[id] = todo

	return todo, nil
}

// Delete removes a todo
func (r *InMemoryTodoRepository) Delete(ctx context.Context, id string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	_, exists := r.todos[id]
	if !exists {
		return ErrTodoNotFound{ID: id}
	}

	delete(r.todos, id)
	return nil
}
