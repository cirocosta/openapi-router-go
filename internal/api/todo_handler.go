// package api provides the HTTP API for the application
package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/cirocosta/openapi-router-go/internal/model"
	"github.com/cirocosta/openapi-router-go/internal/repository"
)

// TodoHandler handles HTTP requests for todo operations
type TodoHandler struct {
	todoService TodoService
}

// NewTodoHandler creates a new todo handler with the given service
func NewTodoHandler(todoService TodoService) *TodoHandler {
	return &TodoHandler{
		todoService: todoService,
	}
}

// ListTodos handles GET /todos
func (h *TodoHandler) ListTodos(w http.ResponseWriter, r *http.Request) {
	todos, err := h.todoService.ListTodos(r.Context())
	if err != nil {
		writeError(w, "error listing todos", http.StatusInternalServerError)
		return
	}

	response := model.TodoListResponse{
		Todos: todos,
	}

	writeJSON(w, response, http.StatusOK)
}

// GetTodo handles GET /todos/{id}
func (h *TodoHandler) GetTodo(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	todo, err := h.todoService.GetTodo(r.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrTodoNotFound{ID: id}) {
			writeError(w, "todo not found", http.StatusNotFound)
			return
		}
		writeError(w, "error getting todo", http.StatusInternalServerError)
		return
	}

	response := model.TodoResponse{
		Todo: todo,
	}

	writeJSON(w, response, http.StatusOK)
}

// CreateTodo handles POST /todos
func (h *TodoHandler) CreateTodo(w http.ResponseWriter, r *http.Request) {
	var req model.CreateTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request format", http.StatusBadRequest)
		return
	}

	todo, err := h.todoService.CreateTodo(r.Context(), req)
	if err != nil {
		if err.Error() == "title is required" {
			writeError(w, err.Error(), http.StatusUnprocessableEntity)
			return
		}
		writeError(w, "error creating todo", http.StatusInternalServerError)
		return
	}

	response := model.TodoResponse{
		Todo: todo,
	}

	writeJSON(w, response, http.StatusCreated)
}

// UpdateTodo handles PUT /todos/{id}
func (h *TodoHandler) UpdateTodo(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req model.UpdateTodoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "invalid request format", http.StatusBadRequest)
		return
	}

	todo, err := h.todoService.UpdateTodo(r.Context(), id, req)
	if err != nil {
		var notFoundErr repository.ErrTodoNotFound
		if errors.As(err, &notFoundErr) {
			writeError(w, "todo not found", http.StatusNotFound)
			return
		}
		writeError(w, "error updating todo", http.StatusInternalServerError)
		return
	}

	response := model.TodoResponse{
		Todo: todo,
	}

	writeJSON(w, response, http.StatusOK)
}

// DeleteTodo handles DELETE /todos/{id}
func (h *TodoHandler) DeleteTodo(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	err := h.todoService.DeleteTodo(r.Context(), id)
	if err != nil {
		var notFoundErr repository.ErrTodoNotFound
		if errors.As(err, &notFoundErr) {
			writeError(w, "todo not found", http.StatusNotFound)
			return
		}
		writeError(w, "error deleting todo", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// writeJSON writes a JSON response with the given status code
func writeJSON(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "error encoding response", http.StatusInternalServerError)
	}
}

// writeError writes an error response with the given status code
func writeError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(model.ErrorResponse{
		Error: message,
	})
}
