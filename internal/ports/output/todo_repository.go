package output

import "golang-template/internal/domain"

// TodoRepository interface - Output port
// Defines what the application needs from data persistence
type TodoRepository interface {
	CreateTodo(request domain.TodoRequest) (*domain.TodoResponse, error)
	UpdateTodo(request domain.TodoRequest) (*domain.TodoResponse, error)
	DeleteTodo(request domain.TodoRequest) (*domain.TodoResponse, error)
	GetTodo(condition domain.QueryTodoRequest) (*domain.TodoListResponse, error)
}
