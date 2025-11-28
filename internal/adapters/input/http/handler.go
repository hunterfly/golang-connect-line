package http

import (
	"golang-template/internal/domain"
	"golang-template/internal/ports/input"
	"golang-template/pkg/validator"

	"gorm.io/gorm"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// HTTPHandler struct - Primary/Driving adapter for HTTP
type HTTPHandler struct {
	srv       input.TodoService
	db        *gorm.DB
	validator validator.Validator
}

// New func - Creates new HTTP handler
func New(srv input.TodoService, db *gorm.DB) *HTTPHandler {
	return &HTTPHandler{
		srv:       srv,
		db:        db,
		validator: validator.New(),
	}
}

// HealthCheck func
func (hdl *HTTPHandler) HealthCheck(c *fiber.Ctx) error {
	sqlDB, err := hdl.db.DB()
	if err != nil {
		logrus.Errorln(err)
		return c.Status(fiber.StatusInternalServerError).JSON(ResponseBody{Status: InternalServerError})
	}

	err = sqlDB.Ping()
	if err != nil {
		logrus.Errorln(err)
		return c.Status(fiber.StatusInternalServerError).JSON(ResponseBody{Status: InternalServerError})
	}
	return c.Status(fiber.StatusOK).JSON(ResponseBody{Status: Success, Data: ""})
}

// CreateTodo func
/* create todo */
// CreateTodo godoc
// @Summary Create todo
// @Description Create todo
// @Tags TODO
// @Accept application/json
// @Success 200 {object} map[string]interface{}
// @Router /v1/api/todo	[post]
// @Produce json
// @param CreateTodo body TodoRequest true "CreateTodo"
func (hdl *HTTPHandler) CreateTodo(c *fiber.Ctx) error {
	var request TodoRequest
	if err := c.BodyParser(&request); err != nil {
		logrus.Errorln(err)
		return c.Status(fiber.StatusBadRequest).JSON(ResponseBody{Status: BadRequest})
	}
	if err := hdl.validator.ValidateStruct(request); err != nil {
		msg := ResponseBody{
			Status: BadRequest,
		}
		msg.Status.Message = []string{
			err.Error(),
		}
		return c.Status(fiber.StatusBadRequest).JSON(msg)
	}
	// Convert HTTP request to domain request
	domainReq := domain.TodoRequest{
		ID:          request.ID,
		Title:       request.Title,
		Description: request.Description,
		Date:        request.Date,
		Image:       request.Image,
		Status:      (*domain.TodoStatus)(request.Status),
	}
	response, err := hdl.srv.CreateTodo(domainReq)
	if err != nil {
		logrus.Errorln(err)
		msg := ResponseBody{
			Status: InternalServerError,
		}
		msg.Status.Message = []string{
			err.Error(),
		}
		return c.Status(fiber.StatusInternalServerError).JSON(msg)
	}
	return c.Status(fiber.StatusOK).JSON(ResponseBody{Status: Success, Data: response})
}

// UpdateTodo func
/* update todo */
// UpdateTodo godoc
// @Summary Update todo
// @Description Update todo
// @Tags TODO
// @Accept application/json
// @Success 200 {object} map[string]interface{}
// @Router /v1/api/todo	[put]
// @Produce json
// @param UpdateTodo body TodoRequest true "UpdateTodo"
func (hdl *HTTPHandler) UpdateTodo(c *fiber.Ctx) error {
	var request TodoRequest
	if err := c.BodyParser(&request); err != nil {
		logrus.Errorln(err)
		return c.Status(fiber.StatusBadRequest).JSON(ResponseBody{Status: BadRequest})
	}
	if err := hdl.validator.ValidateStruct(request); err != nil {
		logrus.Errorln(err)
		return c.Status(fiber.StatusBadRequest).JSON(ResponseBody{Status: BadRequest})
	}
	if request.ID == nil {
		return c.Status(fiber.StatusBadRequest).JSON(ResponseBody{Status: BadRequest})
	}
	// Convert HTTP request to domain request
	domainReq := domain.TodoRequest{
		ID:          request.ID,
		Title:       request.Title,
		Description: request.Description,
		Date:        request.Date,
		Image:       request.Image,
		Status:      (*domain.TodoStatus)(request.Status),
	}
	response, err := hdl.srv.UpdateTodo(domainReq)
	if err != nil {
		msg := ResponseBody{
			Status: InternalServerError,
		}
		msg.Status.Message = []string{
			err.Error(),
		}
		return c.Status(fiber.StatusInternalServerError).JSON(msg)
	}
	return c.Status(fiber.StatusOK).JSON(ResponseBody{Status: Success, Data: response})
}

// DeleteTodo func
/* delete todo */
// DeleteTodo godoc
// @Summary Delete todo
// @Description Delete todo
// @Tags TODO
// @Accept application/json
// @Success 200 {object} map[string]interface{}
// @Router /v1/api/todo/{id}	[delete]
// @Produce json
// @param id path string true "uuid"
func (hdl *HTTPHandler) DeleteTodo(c *fiber.Ctx) error {
	var (
		uid uuid.UUID
		err error
	)
	id := c.Params("id")
	if id != "" {
		uid, err = uuid.Parse(id)
		if err != nil {
			logrus.Errorln(err)
			return c.Status(fiber.StatusBadRequest).JSON(ResponseBody{Status: BadRequest})
		}
	}

	var request TodoRequest
	request.ID = &uid
	// Convert HTTP request to domain request
	domainReq := domain.TodoRequest{
		ID: &uid,
	}
	response, err := hdl.srv.DeleteTodo(domainReq)
	if err != nil {
		msg := ResponseBody{
			Status: InternalServerError,
		}
		msg.Status.Message = []string{
			err.Error(),
		}
		return c.Status(fiber.StatusInternalServerError).JSON(msg)
	}
	return c.Status(fiber.StatusOK).JSON(ResponseBody{Status: Success, Data: response})
}

// GetTodo func
/* delete todo */
// DeleteTodo godoc
// @Summary Delete todo
// @Description Delete todo
// @Tags TODO
// @Accept application/json
// @Success 200 {object} map[string]interface{}
// @Router /v1/api/todo	[get]
// @Produce json
// @param id query string false "uuid"
// @param page query int false "page"
// @param limit query int false "limit"
// @param orderBy query string false "order_by"
// @param asc query bool false "asc"
// @param title query string false "title"
// @param description query string false "description"
// @param status query string false "status"
func (hdl *HTTPHandler) GetTodo(c *fiber.Ctx) error {
	var (
		uid  uuid.UUID
		err  error
		data []TodoResponse
	)
	condition := QueryTodoRequest{}
	err = c.QueryParser(&condition)
	if err != nil {
		logrus.Errorln(err)
		return c.Status(fiber.StatusBadRequest).JSON(ResponseBody{Status: BadRequest})
	}

	err = hdl.validator.ValidateStruct(condition)
	if err != nil {
		logrus.Errorln(err)
		return c.Status(fiber.StatusBadRequest).JSON(ResponseBody{Status: BadRequest})
	}

	id := c.Params("id")
	if id != "" {
		uid, err = uuid.Parse(id)
		if err != nil {
			logrus.Errorln(err)
			return c.Status(fiber.StatusBadRequest).JSON(ResponseBody{Status: BadRequest})
		}
		condition.ID = &uid
	}
	// Convert HTTP query request to domain query request
	domainCondition := domain.QueryTodoRequest{
		ID:          condition.ID,
		Title:       condition.Title,
		Description: condition.Description,
		Status:      condition.Status,
		Limit:       condition.Limit,
		Page:        condition.Page,
		OrderBy:     condition.OrderBy,
		Asc:         condition.Asc,
	}
	result, err := hdl.srv.GetTodo(domainCondition)
	if err != nil {
		logrus.Errorln(err)
		return c.Status(fiber.StatusInternalServerError).JSON(ResponseBody{Status: InternalServerError})
	}
	// Convert domain response to HTTP response
	if result.Todos == nil {
		data = make([]TodoResponse, 0)
	} else {
		for _, todo := range result.Todos {
			httpTodo := TodoResponse{
				ID:          todo.ID,
				Title:       todo.Title,
				Description: todo.Description,
				Date:        todo.Date,
				Image:       todo.Image,
				Status:      (*TodoStatus)(todo.Status),
				CreatedAt:   todo.CreatedAt,
				UpdatedAt:   todo.UpdatedAt,
				DeletedAt:   todo.DeletedAt,
			}
			data = append(data, httpTodo)
		}
	}

	return c.Status(fiber.StatusOK).JSON(ResponseBody{
		Status:      Success,
		Data:        data,
		CurrentPage: result.CurrentPage,
		PerPage:     result.PerPage,
		TotalItem:   result.TotalItem,
	})
}
