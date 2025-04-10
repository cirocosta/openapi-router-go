// package model contains the data models for the sample router application
package model

import (
	"time"
)

// Todo represents a todo item in the system
type Todo struct {
	ID          string    `json:"id" doc:"Unique identifier for the todo item" example:"123e4567-e89b-12d3-a456-426614174000"`
	Title       string    `json:"title" doc:"Title of the todo item" example:"Buy groceries"`
	Description string    `json:"description,omitempty" doc:"Detailed description of the todo item" example:"Need to buy milk, eggs, and bread"`
	Completed   bool      `json:"completed" doc:"Whether the todo item is completed" example:"false"`
	CreatedAt   time.Time `json:"created_at" doc:"When the todo item was created" example:"2023-01-01T12:00:00Z"`
	UpdatedAt   time.Time `json:"updated_at" doc:"When the todo item was last updated" example:"2023-01-02T12:00:00Z"`
}

// CreateTodoRequest is used when creating a new todo item
type CreateTodoRequest struct {
	Title       string `json:"title" doc:"Title of the todo item" example:"Buy groceries"`
	Description string `json:"description,omitempty" doc:"Detailed description of the todo item" example:"Need to buy milk, eggs, and bread"`
}

// UpdateTodoRequest is used when updating an existing todo item
type UpdateTodoRequest struct {
	Title       string `json:"title,omitempty" doc:"Title of the todo item" example:"Buy groceries"`
	Description string `json:"description,omitempty" doc:"Detailed description of the todo item" example:"Need to buy milk, eggs, and bread"`
	Completed   bool   `json:"completed,omitempty" doc:"Whether the todo item is completed" example:"true"`
}

// TodoResponse is used for responses with a single todo item
type TodoResponse struct {
	Todo Todo `json:"todo" doc:"A todo item"`
}

// TodoListResponse is used for responses with multiple todo items
type TodoListResponse struct {
	Todos []Todo `json:"todos" doc:"List of todo items"`
}

// ErrorResponse represents an error returned by the API
type ErrorResponse struct {
	Error string `json:"error" doc:"Error message" example:"Invalid todo ID"`
}
