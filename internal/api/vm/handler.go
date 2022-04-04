package vm

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"rightsizing-api-server/internal/api/common/query"
	_ "rightsizing-api-server/internal/api/common/resource"
)

type VMHandler struct {
	vs     VMService
	logger *zap.Logger
}

func VMRouter(route fiber.Router, vs VMService, logger *zap.Logger) {
	handler := &VMHandler{
		vs:     vs,
		logger: logger,
	}

	// route.Use(auth.JWTMiddleware(), auth.GetDataFromJWT)
	route.Get("/vms/resource-quota", handler.getAllQuota)
	route.Get("/vms/:name/resource-quota", handler.getQuota)

	// resource usage history
	route.Get("/vms/:name", handler.getHistory)
	// resource usage history
	route.Post("/vms/:name/forecast", handler.forecast)
	route.Get("/vms/:name/forecast", handler.forecast)
	route.Get("/vms/:name/forecast/status", handler.getForecastStatus)
	route.Get("/vms/:name/forecast/result", handler.getForecastResult)
	route.Get("/vms/:uuid/forecast/status", handler.getForecastStatusByID)
	route.Get("/vms/:uuid/forecast/result", handler.getForecastResultByID)
}

// @Summary Get all vm resource quota
// @Description Get all vm resource quota information from TimescaleDB
// @Accept  json
// @Produce json
// @Success 200 {object} Vm list
// @Failure 400 {object} nil
// @Failure 404 {object} nil
// @Failure 500 {object} nil
// @Router /api/v1/vms/resource-quota [get]
func (h *VMHandler) getAllQuota(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON("")
}

// @Summary Get vm resource quota
// @Description Get vm resource quota information from TimescaleDB
// @Accept  json
// @Produce json
// @Param name path  string  true  "the name of vm"
// @Success 200 {object} object
// @Failure 400 {object} nil
// @Failure 404 {object} nil
// @Failure 500 {object} nil
// @Router /api/v1/vms/{name}/resource-quota [get]
func (h *VMHandler) getQuota(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON("")
}

// @Summary Get vm usage history and optimization usage
// @Description Get all resource usage history and optimization usage value
// @Accept  json
// @Produce json
// @Param name 	path  string  true  "name of the vm"
// @Param start query string  false "start time"
// @Param end   query string  false "end time"
// @Success 200 {object} object
// @Failure 400 {object} nil
// @Failure 404 {object} nil
// @Failure 500 {object} nil
// @Router /api/v1/vms/{name} [get]
func (h *VMHandler) getHistory(c *fiber.Ctx) error {
	query, err := query.ParseAndValidate(c)
	if err != nil {
		h.logger.Debug("query parser error", zap.Error(err))
	}

	vm, err := h.vs.GetVm(query)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(err)
	}
	if vm == nil {
		return c.Status(fiber.StatusNotFound).JSON(nil)
	}

	return c.Status(fiber.StatusOK).JSON(vm)
}

// @Summary Post vm forecast task
// @Description Create forecast task and result task UUID
// @Accept  json
// @Produce json
// @Param name      path string true "the name of vm"
// @Success 200 {object} object
// @Failure 400 {object} nil
// @Failure 404 {object} nil
// @Failure 500 {object} nil
// @Router /api/v1/vms/{name}/forecast [get]
func (h *VMHandler) forecast(c *fiber.Ctx) error {
	query, err := query.ParseAndValidate(c)
	if err != nil {
		h.logger.Debug("query parser error", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(err)
	}

	uuid, err := h.vs.Forecast(query)
	if err != nil {
		h.logger.Debug("query parser error", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(err)
	}

	return c.Status(fiber.StatusOK).JSON(map[string]interface{}{
		"uuid": uuid,
	})
}

// @Summary Get vm forecast task status
// @Description Get forecast task from TimescaleDB
// @Accept  json
// @Produce json
// @Param name path string true "the name of vm"
// @Success 200 {object} object
// @Failure 400 {object} nil
// @Failure 404 {object} nil
// @Failure 500 {object} nil
// @Router /api/v1/vms/{name}/forecast/status [get]
func (h *VMHandler) getForecastStatus(c *fiber.Ctx) error {
	var (
		id   = c.Locals("requestid").(string)
		name = c.Params("name")
	)

	id, err := h.vs.GetForecastStatus(name)
	if err != nil {
		h.logger.Debug("failed to get status", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(err)
	}

	return c.Status(fiber.StatusOK).JSON(map[string]interface{}{
		"uuid": id,
	})
}

// @Summary Get vm forecast result
// @Description Get forecast result from TimescaleDB
// @Accept  json
// @Produce json
// @Param name path string true "the name of vm"
// @Success 200 {object} object
// @Failure 400 {object} nil
// @Failure 404 {object} nil
// @Failure 500 {object} nil
// @Router /api/v1/vms/{name}/forecast [get]
func (h *VMHandler) getForecastResult(c *fiber.Ctx) error {
	var (
		id   = c.Locals("requestid").(string)
		name = c.Params("name")
	)

	forecastUsage, err := h.vs.GetForecastResult(name)
	if err != nil {
		h.logger.Debug("failed to get result", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(err)
	}
	if forecastUsage == nil {
		c.Status(fiber.StatusNoContent)
	}

	return c.Status(fiber.StatusOK).JSON(map[string]interface{}{
		"uuid":   id,
		"result": forecastUsage,
	})
}

// @Summary Get vm forecast task status by UUID
// @Description Get forecast task status by UUID
// @Accept  json
// @Produce json
// @Param uuid path string true "the uuid of forecast task"
// @Success 200 {object} object
// @Failure 400 {object} nil
// @Failure 404 {object} nil
// @Failure 500 {object} nil
// @Router /api/v1/vms/{uuid}/forecast/status [get]
func (h *VMHandler) getForecastStatusByID(c *fiber.Ctx) error {
	var (
		id   = c.Locals("requestid").(string)
		uuid = c.Params("uuid")
	)

	forecastUsage, err := h.vs.GetForecastStatusByID(uuid)
	if err != nil {
		h.logger.Debug("failed to get result", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(err)
	}

	return c.Status(fiber.StatusOK).JSON(map[string]interface{}{
		"id":     id,
		"result": forecastUsage,
	})
}

// @Summary Get vm forecast result
// @Description Get forecast result
// @Accept  json
// @Produce json
// @Param uuid path string true "the uuid of forecast task"
// @Success 200 {object} object
// @Failure 400 {object} nil
// @Failure 404 {object} nil
// @Failure 500 {object} nil
// @Router /api/v1/pods/{uuid}/forecast/result [get]
func (h *VMHandler) getForecastResultByID(c *fiber.Ctx) error {
	var (
		id   = c.Locals("requestid").(string)
		uuid = c.Params("uuid")
	)

	forecastUsage, err := h.vs.GetForecastResultByID(uuid)
	if err != nil {
		h.logger.Debug("failed to get result", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(err)
	}
	if forecastUsage == nil {
		c.Status(fiber.StatusNoContent)
	}

	return c.Status(fiber.StatusOK).JSON(map[string]interface{}{
		"id":     id,
		"result": forecastUsage,
	})
}
