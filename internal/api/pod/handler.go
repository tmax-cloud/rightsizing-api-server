package pod

import (
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"rightsizing-api-server/internal/api/common/query"
	_ "rightsizing-api-server/internal/api/common/resource"
)

type PodHandler struct {
	ps     PodService
	logger *zap.Logger
}

func PodRouter(route fiber.Router, ps PodService, logger *zap.Logger) {
	handler := &PodHandler{
		ps:     ps,
		logger: logger,
	}

	// route.Use(auth.JWTMiddleware(), auth.GetDataFromJWT)

	// resource-quota
	route.Get("/resource-quota", handler.getAllQuota)
	route.Get("/:namespace/:name/resource-quota", handler.getQuota)
	// resource usage history
	route.Get("/:namespace/:name", handler.getHistory)
	// resource usage history
	route.Post("/:namespace/:name/forecast", handler.forecast)
	route.Get("/:namespace/:name/forecast", handler.forecast)
	route.Get("/:namespace/:name/forecast/status", handler.getForecastStatus)
	route.Get("/:namespace/:name/forecast/result", handler.getForecastResult)
	route.Get("/:uuid/forecast/status", handler.getForecastStatusByID)
	route.Get("/:uuid/forecast/result", handler.getForecastResultByID)
}

// @Summary Get all pod resource quota
// @Description Get all pod resource quota information from TimescaleDB
// @Accept  json
// @Produce json
// @Param start query string  false "start time"
// @Param end   query string  false "end time"
// @Success 200 {object} Pod list
// @Failure 400 {object} nil
// @Failure 404 {object} nil
// @Failure 500 {object} nil
// @Router /api/v1/pods/resource-quota [get]
func (h *PodHandler) getAllQuota(c *fiber.Ctx) error {
	query, err := query.ParseAndValidate(c)
	if err != nil {
		h.logger.Debug("query parser error", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(err)
	}

	pods, err := h.ps.GetAllPodQuota(query)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(err)
	}
	if pods == nil {
		return c.Status(fiber.StatusNotFound).JSON(nil)
	}
	return c.Status(fiber.StatusOK).JSON(pods)
}

// @Summary Get pod resource quota
// @Description Get pod resource quota information from TimescaleDB
// @Accept  json
// @Produce json
// @Param name      path  string  true  "the name of pod"
// @Param namespace path  string  true  "the namespace of pod"
// @Param start     query string  false "start time"
// @Param end       query string  false "end time"
// @Success 200 {object} Pod
// @Failure 400 {object} nil
// @Failure 404 {object} nil
// @Failure 500 {object} nil
// @Router /api/v1/pods/{namespace}/{name}/resource-quota [get]
func (h *PodHandler) getQuota(c *fiber.Ctx) error {
	query, err := query.ParseAndValidate(c)
	if err != nil {
		h.logger.Debug("query parser error", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(err)
	}

	pod, err := h.ps.GetPodQuota(query)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(err)
	}
	if pod == nil {
		return c.Status(fiber.StatusNotFound).JSON(nil)
	}
	return c.Status(fiber.StatusOK).JSON(pod)
}

// @Summary Get usage history and optimization usage
// @Description Get all resource usage history and optimization usage value
// @Accept  json
// @Produce json
// @Param name      path  string  true  "the name of pod"
// @Param namespace path  string  true  "the namespace of pod"
// @Param start     query string  false "start time"
// @Param end       query string  false "end time"
// @Success 200 {object} Pod
// @Failure 400 {object} nil
// @Failure 404 {object} nil
// @Failure 500 {object} nil
// @Router /api/v1/pods/{namespace}/{name} [get]
func (h *PodHandler) getHistory(c *fiber.Ctx) error {
	query, err := query.ParseAndValidate(c)
	if err != nil {
		h.logger.Debug("query parser error", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(err)
	}

	pod, err := h.ps.GetPod(query)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(err)
	}
	if pod == nil {
		return c.Status(fiber.StatusNotFound).JSON(nil)
	}
	return c.Status(fiber.StatusOK).JSON(pod)
}

// @Summary Post pod forecast task
// @Description Create forecast task and result task UUID
// @Accept  json
// @Produce json
// @Param namespace path string true "the namespace of pod"
// @Param name      path string true "the name of pod"
// @Success 200 {object} object
// @Failure 400 {object} nil
// @Failure 404 {object} nil
// @Failure 500 {object} nil
// @Router /api/v1/pods/{namespace}/{name}/forecast [get]
func (h *PodHandler) forecast(c *fiber.Ctx) error {
	query, err := query.ParseAndValidate(c)
	if err != nil {
		h.logger.Debug("query parser error", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(err)
	}

	uuid, err := h.ps.Forecast(query)
	if err != nil {
		h.logger.Debug("query parser error", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(err)
	}

	return c.Status(fiber.StatusOK).JSON(map[string]interface{}{
		"uuid": uuid,
	})
}

// @Summary Get pod forecast result
// @Description Get forecast result from TimescaleDB
// @Accept  json
// @Produce json
// @Param name      path string true "the name of pod"
// @Param namespace path string true "the namespace of pod"
// @Success 200 {object} object
// @Failure 400 {object} nil
// @Failure 404 {object} nil
// @Failure 500 {object} nil
// @Router /api/v1/pods/{namespace}/{name}/forecast [get]
func (h *PodHandler) getForecastStatus(c *fiber.Ctx) error {
	var (
		id        = c.Locals("requestid").(string)
		namespace = c.Params("namespace")
		name      = c.Params("name")
	)

	id, err := h.ps.GetForecastStatus(namespace, name)
	if err != nil {
		h.logger.Debug("failed to get status", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(err)
	}

	return c.Status(fiber.StatusOK).JSON(map[string]interface{}{
		"uuid": id,
	})
}

// @Summary Get pod forecast result
// @Description Get forecast result from TimescaleDB
// @Accept  json
// @Produce json
// @Param name      path string true "the name of pod"
// @Param namespace path string true "the namespace of pod"
// @Success 200 {object} object
// @Failure 400 {object} nil
// @Failure 404 {object} nil
// @Failure 500 {object} nil
// @Router /api/v1/pods/{namespace}/{name}/forecast [get]
func (h *PodHandler) getForecastResult(c *fiber.Ctx) error {
	var (
		id        = c.Locals("requestid").(string)
		namespace = c.Params("namespace")
		name      = c.Params("name")
	)

	forecastUsage, err := h.ps.GetForecastResult(namespace, name)
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

// @Summary Get pod forecast task status
// @Description Get forecast task status
// @Accept  json
// @Produce json
// @Param uuid path string true "the uuid of forecast task"
// @Success 200 {object} object
// @Failure 400 {object} nil
// @Failure 404 {object} nil
// @Failure 500 {object} nil
// @Router /api/v1/pods/{uuid}/forecast/status [get]
func (h *PodHandler) getForecastStatusByID(c *fiber.Ctx) error {
	var (
		id   = c.Locals("requestid").(string)
		uuid = c.Params("uuid")
	)

	forecastUsage, err := h.ps.GetForecastStatusByID(uuid)
	if err != nil {
		h.logger.Debug("failed to get result", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(err)
	}

	return c.Status(fiber.StatusOK).JSON(map[string]interface{}{
		"id":     id,
		"result": forecastUsage,
	})
}

// @Summary Get pod forecast result
// @Description Get forecast result
// @Accept  json
// @Produce json
// @Param uuid path string true "the uuid of forecast task"
// @Success 200 {object} object
// @Failure 400 {object} nil
// @Failure 404 {object} nil
// @Failure 500 {object} nil
// @Router /api/v1/pods/{uuid}/forecast/result [get]
func (h *PodHandler) getForecastResultByID(c *fiber.Ctx) error {
	var (
		id   = c.Locals("requestid").(string)
		uuid = c.Params("uuid")
	)

	forecastUsage, err := h.ps.GetForecastResultByID(uuid)
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
