package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/cirocosta/openapi-router-go/internal/model"
	"github.com/cirocosta/openapi-router-go/internal/repository"
)

// mockTodoService is a mock implementation of TodoService
type mockTodoService struct {
	mock.Mock
}

func (m *mockTodoService) ListTodos(ctx context.Context) ([]model.Todo, error) {
	args := m.Called(ctx)
	return args.Get(0).([]model.Todo), args.Error(1)
}

func (m *mockTodoService) GetTodo(ctx context.Context, id string) (model.Todo, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(model.Todo), args.Error(1)
}

func (m *mockTodoService) CreateTodo(ctx context.Context, req model.CreateTodoRequest) (model.Todo, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(model.Todo), args.Error(1)
}

func (m *mockTodoService) UpdateTodo(ctx context.Context, id string, req model.UpdateTodoRequest) (model.Todo, error) {
	args := m.Called(ctx, id, req)
	return args.Get(0).(model.Todo), args.Error(1)
}

func (m *mockTodoService) DeleteTodo(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func TestListTodos(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		setupMock    func(m *mockTodoService)
		wantStatus   int
		wantResponse model.TodoListResponse
		wantErr      string
	}{
		"success": {
			setupMock: func(m *mockTodoService) {
				todos := []model.Todo{
					{ID: "1", Title: "Todo 1", Completed: false},
					{ID: "2", Title: "Todo 2", Completed: true},
				}
				m.On("ListTodos", mock.Anything).Return(todos, nil)
			},
			wantStatus: http.StatusOK,
			wantResponse: model.TodoListResponse{
				Todos: []model.Todo{
					{ID: "1", Title: "Todo 1", Completed: false},
					{ID: "2", Title: "Todo 2", Completed: true},
				},
			},
		},
		"service error": {
			setupMock: func(m *mockTodoService) {
				m.On("ListTodos", mock.Anything).Return([]model.Todo{}, errors.New("database error"))
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    "error listing todos",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			mockService := new(mockTodoService)
			tc.setupMock(mockService)

			handler := NewTodoHandler(mockService)
			req := httptest.NewRequest(http.MethodGet, "/todos", nil).WithContext(ctx)
			rec := httptest.NewRecorder()

			handler.ListTodos(rec, req)

			assert.Equal(t, tc.wantStatus, rec.Code)

			if tc.wantErr != "" {
				var errResp model.ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &errResp)
				require.NoError(t, err)
				assert.Equal(t, tc.wantErr, errResp.Error)
				return
			}

			var gotResp model.TodoListResponse
			err := json.Unmarshal(rec.Body.Bytes(), &gotResp)
			require.NoError(t, err)

			if diff := cmp.Diff(tc.wantResponse, gotResp); diff != "" {
				t.Errorf("response mismatch (-want +got):\n%s", diff)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestGetTodo(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		todoID     string
		setupMock  func(m *mockTodoService)
		wantStatus int
		wantTodo   model.Todo
		wantErr    string
	}{
		"success": {
			todoID: "123",
			setupMock: func(m *mockTodoService) {
				todo := model.Todo{ID: "123", Title: "Test Todo", Completed: false}
				m.On("GetTodo", mock.Anything, "123").Return(todo, nil)
			},
			wantStatus: http.StatusOK,
			wantTodo:   model.Todo{ID: "123", Title: "Test Todo", Completed: false},
		},
		"not found": {
			todoID: "999",
			setupMock: func(m *mockTodoService) {
				m.On("GetTodo", mock.Anything, "999").Return(model.Todo{}, repository.ErrTodoNotFound{ID: "999"})
			},
			wantStatus: http.StatusNotFound,
			wantErr:    "todo not found",
		},
		"service error": {
			todoID: "123",
			setupMock: func(m *mockTodoService) {
				m.On("GetTodo", mock.Anything, "123").Return(model.Todo{}, errors.New("database error"))
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    "error getting todo",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mockService := new(mockTodoService)
			tc.setupMock(mockService)

			handler := NewTodoHandler(mockService)

			// Create a custom request with URL parameters that can be accessed by r.PathValue()
			req := httptest.NewRequest(http.MethodGet, "/todos/"+tc.todoID, nil)

			// Create a custom request context to mock URL parameters
			ctx := req.Context()
			req = req.WithContext(ctx)

			// Use a testing ResponseRecorder
			rec := httptest.NewRecorder()

			// Create a request with the URL pattern
			reqWithParams := req.Clone(req.Context())
			reqWithParams.SetPathValue("id", tc.todoID)

			// Call the handler directly
			handler.GetTodo(rec, reqWithParams)

			assert.Equal(t, tc.wantStatus, rec.Code)

			if tc.wantErr != "" {
				var errResp model.ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &errResp)
				require.NoError(t, err)
				assert.Equal(t, tc.wantErr, errResp.Error)
				return
			}

			var gotResp model.TodoResponse
			err := json.Unmarshal(rec.Body.Bytes(), &gotResp)
			require.NoError(t, err)

			if diff := cmp.Diff(tc.wantTodo, gotResp.Todo); diff != "" {
				t.Errorf("todo mismatch (-want +got):\n%s", diff)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestCreateTodo(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		requestBody string
		setupMock   func(m *mockTodoService)
		wantStatus  int
		wantTodo    model.Todo
		wantErr     string
	}{
		"success": {
			requestBody: `{"title": "New Todo"}`,
			setupMock: func(m *mockTodoService) {
				expectedReq := model.CreateTodoRequest{Title: "New Todo"}
				createdTodo := model.Todo{ID: "new-id", Title: "New Todo", Completed: false}
				m.On("CreateTodo", mock.Anything, expectedReq).Return(createdTodo, nil)
			},
			wantStatus: http.StatusCreated,
			wantTodo:   model.Todo{ID: "new-id", Title: "New Todo", Completed: false},
		},
		"invalid json": {
			requestBody: `{invalid json`,
			setupMock:   func(m *mockTodoService) {},
			wantStatus:  http.StatusBadRequest,
			wantErr:     "invalid request format",
		},
		"missing title": {
			requestBody: `{"completed": true}`,
			setupMock: func(m *mockTodoService) {
				expectedReq := model.CreateTodoRequest{Title: ""}
				m.On("CreateTodo", mock.Anything, expectedReq).Return(model.Todo{}, errors.New("title is required"))
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantErr:    "title is required",
		},
		"service error": {
			requestBody: `{"title": "New Todo"}`,
			setupMock: func(m *mockTodoService) {
				expectedReq := model.CreateTodoRequest{Title: "New Todo"}
				m.On("CreateTodo", mock.Anything, expectedReq).Return(model.Todo{}, errors.New("database error"))
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    "error creating todo",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			mockService := new(mockTodoService)
			tc.setupMock(mockService)

			handler := NewTodoHandler(mockService)

			req := httptest.NewRequest(http.MethodPost, "/todos", strings.NewReader(tc.requestBody)).WithContext(ctx)
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			handler.CreateTodo(rec, req)

			assert.Equal(t, tc.wantStatus, rec.Code)

			if tc.wantErr != "" {
				var errResp model.ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &errResp)
				require.NoError(t, err)
				assert.Equal(t, tc.wantErr, errResp.Error)
				return
			}

			var gotResp model.TodoResponse
			err := json.Unmarshal(rec.Body.Bytes(), &gotResp)
			require.NoError(t, err)

			if diff := cmp.Diff(tc.wantTodo, gotResp.Todo); diff != "" {
				t.Errorf("todo mismatch (-want +got):\n%s", diff)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestUpdateTodo(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		todoID      string
		requestBody string
		setupMock   func(m *mockTodoService)
		wantStatus  int
		wantTodo    model.Todo
		wantErr     string
	}{
		"success": {
			todoID:      "123",
			requestBody: `{"title": "Updated Todo", "completed": true}`,
			setupMock: func(m *mockTodoService) {
				expectedReq := model.UpdateTodoRequest{Title: "Updated Todo", Completed: true}
				updatedTodo := model.Todo{ID: "123", Title: "Updated Todo", Completed: true}
				m.On("UpdateTodo", mock.Anything, "123", expectedReq).Return(updatedTodo, nil)
			},
			wantStatus: http.StatusOK,
			wantTodo:   model.Todo{ID: "123", Title: "Updated Todo", Completed: true},
		},
		"invalid json": {
			todoID:      "123",
			requestBody: `{invalid json`,
			setupMock:   func(m *mockTodoService) {},
			wantStatus:  http.StatusBadRequest,
			wantErr:     "invalid request format",
		},
		"not found": {
			todoID:      "999",
			requestBody: `{"title": "Updated Todo", "completed": true}`,
			setupMock: func(m *mockTodoService) {
				expectedReq := model.UpdateTodoRequest{Title: "Updated Todo", Completed: true}
				m.On("UpdateTodo", mock.Anything, "999", expectedReq).Return(model.Todo{}, repository.ErrTodoNotFound{ID: "999"})
			},
			wantStatus: http.StatusNotFound,
			wantErr:    "todo not found",
		},
		"service error": {
			todoID:      "123",
			requestBody: `{"title": "Updated Todo", "completed": true}`,
			setupMock: func(m *mockTodoService) {
				expectedReq := model.UpdateTodoRequest{Title: "Updated Todo", Completed: true}
				m.On("UpdateTodo", mock.Anything, "123", expectedReq).Return(model.Todo{}, errors.New("database error"))
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    "error updating todo",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mockService := new(mockTodoService)
			tc.setupMock(mockService)

			handler := NewTodoHandler(mockService)

			// Create request with body and URL parameter
			req := httptest.NewRequest(http.MethodPut, "/todos/"+tc.todoID, strings.NewReader(tc.requestBody))
			req.Header.Set("Content-Type", "application/json")

			// Set path value
			reqWithParams := req.Clone(req.Context())
			reqWithParams.SetPathValue("id", tc.todoID)

			rec := httptest.NewRecorder()

			// Call handler directly
			handler.UpdateTodo(rec, reqWithParams)

			assert.Equal(t, tc.wantStatus, rec.Code)

			if tc.wantErr != "" {
				var errResp model.ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &errResp)
				require.NoError(t, err)
				assert.Equal(t, tc.wantErr, errResp.Error)
				return
			}

			var gotResp model.TodoResponse
			err := json.Unmarshal(rec.Body.Bytes(), &gotResp)
			require.NoError(t, err)

			if diff := cmp.Diff(tc.wantTodo, gotResp.Todo); diff != "" {
				t.Errorf("todo mismatch (-want +got):\n%s", diff)
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestDeleteTodo(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		todoID     string
		setupMock  func(m *mockTodoService)
		wantStatus int
		wantErr    string
	}{
		"success": {
			todoID: "123",
			setupMock: func(m *mockTodoService) {
				m.On("DeleteTodo", mock.Anything, "123").Return(nil)
			},
			wantStatus: http.StatusNoContent,
		},
		"not found": {
			todoID: "999",
			setupMock: func(m *mockTodoService) {
				m.On("DeleteTodo", mock.Anything, "999").Return(repository.ErrTodoNotFound{ID: "999"})
			},
			wantStatus: http.StatusNotFound,
			wantErr:    "todo not found",
		},
		"service error": {
			todoID: "123",
			setupMock: func(m *mockTodoService) {
				m.On("DeleteTodo", mock.Anything, "123").Return(errors.New("database error"))
			},
			wantStatus: http.StatusInternalServerError,
			wantErr:    "error deleting todo",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mockService := new(mockTodoService)
			tc.setupMock(mockService)

			handler := NewTodoHandler(mockService)

			// Create request with URL parameter
			req := httptest.NewRequest(http.MethodDelete, "/todos/"+tc.todoID, nil)

			// Set path value for ID
			reqWithParams := req.Clone(req.Context())
			reqWithParams.SetPathValue("id", tc.todoID)

			rec := httptest.NewRecorder()

			// Call handler directly
			handler.DeleteTodo(rec, reqWithParams)

			assert.Equal(t, tc.wantStatus, rec.Code)

			if tc.wantErr != "" {
				var errResp model.ErrorResponse
				err := json.Unmarshal(rec.Body.Bytes(), &errResp)
				require.NoError(t, err)
				assert.Equal(t, tc.wantErr, errResp.Error)
			} else if rec.Body.Len() > 0 {
				t.Errorf("expected empty response body for success case, got: %s", rec.Body.String())
			}

			mockService.AssertExpectations(t)
		})
	}
}

// TestHelperFunctions tests the writeJSON and writeError functions
func TestHelperFunctions(t *testing.T) {
	t.Parallel()

	t.Run("writeJSON", func(t *testing.T) {
		t.Parallel()

		data := map[string]string{"key": "value"}
		rec := httptest.NewRecorder()

		writeJSON(rec, data, http.StatusOK)

		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

		var result map[string]string
		err := json.Unmarshal(rec.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, data, result)
	})

	t.Run("writeError", func(t *testing.T) {
		t.Parallel()

		rec := httptest.NewRecorder()
		errorMsg := "test error"

		writeError(rec, errorMsg, http.StatusBadRequest)

		assert.Equal(t, http.StatusBadRequest, rec.Code)
		assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

		var result model.ErrorResponse
		err := json.Unmarshal(rec.Body.Bytes(), &result)
		require.NoError(t, err)
		assert.Equal(t, errorMsg, result.Error)
	})
}
