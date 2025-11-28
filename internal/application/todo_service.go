package application

import (
	"golang-template/internal/domain"
	"golang-template/internal/ports/output"

	"github.com/sirupsen/logrus"
)

// TodoService struct - Application service implementing use cases
type TodoService struct {
	repo output.TodoRepository
}

// NewTodoService func - Creates new todo service
func NewTodoService(repo output.TodoRepository) *TodoService {
	return &TodoService{
		repo: repo,
	}
}

// CreateTodo func - Use case: Create a new todo
func (s *TodoService) CreateTodo(request domain.TodoRequest) (*domain.TodoResponse, error) {
	result, err := s.repo.CreateTodo(request)
	if err != nil {
		logrus.Errorln(err)
		return nil, err
	}
	return result, nil
}

// UpdateTodo func - Use case: Update an existing todo
func (s *TodoService) UpdateTodo(request domain.TodoRequest) (*domain.TodoResponse, error) {
	return s.repo.UpdateTodo(request)
}

// DeleteTodo func - Use case: Delete a todo
func (s *TodoService) DeleteTodo(request domain.TodoRequest) (*domain.TodoResponse, error) {
	return s.repo.DeleteTodo(request)
}

// GetTodo func - Use case: Get todo(s) with pagination and filtering
func (s *TodoService) GetTodo(condition domain.QueryTodoRequest) (*domain.TodoListResponse, error) {
	var (
		page    int
		perPage int
		offset  int
	)
	if condition.Page != nil {
		page = *condition.Page
	} else {
		page = 1
		condition.Page = &page
	}
	if condition.Limit != nil {
		perPage = *condition.Limit
	} else {
		perPage = 100
		condition.Limit = &perPage
	}
	offset = (page - 1) * perPage
	condition.Pagination = &domain.Pagination{
		Limit:  perPage,
		Offset: offset,
	}
	if condition.OrderBy != nil {
		asc := true
		if condition.Asc != nil {
			asc = *condition.Asc
		}
		condition.SortMethod = &domain.SortMethod{
			Asc:     asc,
			OrderBy: *condition.OrderBy,
		}
	} else {
		asc := true
		if condition.Asc != nil {
			asc = *condition.Asc
		}
		condition.SortMethod = &domain.SortMethod{
			Asc:     asc,
			OrderBy: "ID",
		}
	}
	return s.repo.GetTodo(condition)
}
