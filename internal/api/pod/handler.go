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

	route.Get("/pods", handler.getRightsizing)

	rg := route.Group("/pods")
	rg.Get("/clusterinfo", handler.getClusterInfo)
	// resource usage history
	rg.Post("/forecast", handler.forecast)
	rg.Get("/forecast", handler.forecast)
	rg.Get("/forecast/status", handler.getForecastStatus)
	rg.Get("/forecast/result", handler.getForecastResult)
	rg.Get("/forecast/:uuid/status", handler.getForecastStatusByID)
	rg.Get("/forecast/:uuid/result", handler.getForecastResultByID)
}

// @Summary 클러스터 전반적인 지표들을 제공
// @Accept  json
// @Produce json
// @Success 200 {object} object
// @Failure 400 {object} nil
// @Failure 404 {object} nil
// @Failure 500 {object} nil
// @Router /api/v1/pods/resource-quota [get]
func (h *PodHandler) getClusterInfo(c *fiber.Ctx) error {
	info, err := h.ps.GetClusterInfo()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(err)
	}
	return c.Status(fiber.StatusOK).JSON(info)
}

// @Summary pod의 리소스 정보 및 사용량 관련 정보 제공
// @Description pod의 리소스 quota 정보와 사용량 및 사용량 기반의 최적 사용량을 제공한다.
// namespace, name을 지정하지 않으면 모든 pod들에 대해 제공한다. 단, 둘 다 명시하거나 둘 다 명시하지 않아야함.
// @Accept  json
// @Produce json
// @Param name      path  string  false  "the name of pod"
// @Param namespace path  string  false  "the namespace of pod"
// @Param start     query string  false "start time"
// @Param end       query string  false "end time"
// @Success 200 {object} Pod or Pod list
// @Failure 400 {object} nil
// @Failure 404 {object} nil
// @Failure 500 {object} nil
// @Router /api/v1/pods [get]
func (h *PodHandler) getRightsizing(c *fiber.Ctx) error {
	query, err := query.ParseAndValidate(c)
	if err != nil {
		h.logger.Debug("query parser error", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(err)
	}
	if query.Namespace == "" && query.Name == "" {
		pods, err := h.ps.GetAllPod(query)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(err)
		}
		return c.Status(fiber.StatusOK).JSON(pods)
	} else {
		pod, err := h.ps.GetPod(query)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(err)
		}

		if pod == nil {
			return c.Status(fiber.StatusNotFound).JSON(nil)
		}
		return c.Status(fiber.StatusOK).JSON(pod)
	}
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

// @Summary 특정 pod의 forecast 완료 여부를 알려줌
// @Accept  json
// @Produce json
// @Param name      path string true "the name of pod"
// @Param namespace path string true "the namespace of pod"
// @Success 200 {object} object
// @Failure 400 {object} nil
// @Failure 404 {object} nil
// @Failure 500 {object} nil
// @Router /api/v1/pods/forecast/status [get]
func (h *PodHandler) getForecastStatus(c *fiber.Ctx) error {
	var (
		namespace = c.Params("namespace")
		name      = c.Params("name")
	)

	status, err := h.ps.GetForecastStatus(namespace, name)
	if err != nil {
		h.logger.Debug("failed to get status", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(err)
	}

	return c.Status(fiber.StatusOK).JSON(map[string]interface{}{
		status: status,
	})
}

// @Summary 특정 pod의 forecast 결과 제공
// @Description forecast 작업이 끝나지 않은 경우 nil 값 제공
// @Accept  json
// @Produce json
// @Param name      path string true "the name of pod"
// @Param namespace path string true "the namespace of pod"
// @Success 200 {object} object
// @Failure 400 {object} nil
// @Failure 404 {object} nil
// @Failure 500 {object} nil
// @Router /api/v1/pods/forecast [get]
func (h *PodHandler) getForecastResult(c *fiber.Ctx) error {
	var (
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
		"result": forecastUsage,
	})
}

// @Summary 사용자의 요청에 따라 발급한 forecast id를 통해서 forecast 완료 여부 제공
// @Accept  json
// @Produce json
// @Param uuid path string true "the uuid of forecast task"
// @Success 200 {object} object
// @Failure 400 {object} nil
// @Failure 404 {object} nil
// @Failure 500 {object} nil
// @Router /api/v1/pods/forecast/{uuid}/status [get]
func (h *PodHandler) getForecastStatusByID(c *fiber.Ctx) error {
	var (
		uuid = c.Params("uuid")
	)

	status, err := h.ps.GetForecastStatusByID(uuid)
	if err != nil {
		h.logger.Debug("failed to get status", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(err)
	}

	return c.Status(fiber.StatusOK).JSON(map[string]interface{}{
		"status": status,
	})
}

// @Summary 사용자의 요청에 따라 발급한 forecast id를 통해서 forecast 결과를 제공
// @Description id를 통해서 forecast 결과를 제공함. 만약 작업이 끝나지 않은 경우 nil 값 제공.
// @Accept  json
// @Produce json
// @Param uuid path string true "the uuid of forecast task"
// @Success 200 {object} object
// @Failure 400 {object} nil
// @Failure 404 {object} nil
// @Failure 500 {object} nil
// @Router /api/v1/pods/forecast/{uuid}/result [get]
func (h *PodHandler) getForecastResultByID(c *fiber.Ctx) error {
	var (
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
		"result": forecastUsage,
	})
}
