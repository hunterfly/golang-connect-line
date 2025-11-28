package input

import "golang-template/internal/domain"

// TodoService interface - Input port (use case)
// Defines what the application can do with todos
type TodoService interface {
	CreateTodo(request domain.TodoRequest) (*domain.TodoResponse, error)
	UpdateTodo(request domain.TodoRequest) (*domain.TodoResponse, error)
	DeleteTodo(request domain.TodoRequest) (*domain.TodoResponse, error)
	GetTodo(condition domain.QueryTodoRequest) (*domain.TodoListResponse, error)
}
