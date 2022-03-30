package pod

import (
	"context"
	"errors"
	"fmt"
	pb "rightsizing-api-server/proto"

	"github.com/RichardKnop/machinery/v1/tasks"
	"go.uber.org/zap"

	commonerrors "rightsizing-api-server/internal/api/common/errors"
	"rightsizing-api-server/internal/api/common/query"
	"rightsizing-api-server/internal/api/common/resource"
	"rightsizing-api-server/internal/api/common/rightsizing"
	grpcclient "rightsizing-api-server/internal/grpc"
	"rightsizing-api-server/internal/worker"
)

const (
	taskName = "pod_forecast"
)

type podService struct {
	worker     *worker.Worker
	client     *grpcclient.Client
	repository PodRepository
	logger     *zap.Logger
}

var _ PodService = (*podService)(nil)

func NewPodService(
	worker *worker.Worker,
	client *grpcclient.Client,
	r PodRepository,
	logger *zap.Logger) PodService {
	s := &podService{
		worker:     worker,
		client:     client,
		repository: r,
		logger:     logger,
	}

	worker.RegisterTask(taskName, s.forecastTask)

	return s
}

func (ps *podService) GetAllPodQuota(query query.Query) ([]*Pod, error) {
	ps.logger.Debug("Get all pod quota")

	pods, err := ps.repository.GetAllPodQuota(query)
	if err != nil {
		ps.logger.Error("failed to get pod from database", zap.Error(err))
		return nil, err
	}

	for _, pod := range pods {
		for _, container := range pod.Containers {
			container.OptimizedUsage = make(map[string]float64)
			for _, usage := range container.Usage {
				if len(usage.Usage) == 0 {
					continue
				}
				if err := rightsizing.Rightsizing(context.Background(), ps.client, usage); err != nil {
					ps.logger.Error("failed while rightsizing", zap.Error(err))
					return nil, err
				}
				container.OptimizedUsage[usage.ResourceName] = usage.OptimizedUsage
			}
			container.Usage = nil
		}
	}
	return pods, nil
}

func (ps *podService) GetPodQuota(query query.Query) (*Pod, error) {
	ps.logger.Debug("rightsizing pod",
		zap.String("id", query.ID),
		zap.String("pod", query.Name))

	pod, err := ps.repository.GetPodQuota(query)
	if err != nil {
		ps.logger.Error("failed to get pod from database", zap.Error(err))
		return nil, err
	}

	if pod == nil {
		return nil, commonerrors.NotFoundErr("pod", query.Name)
	}
	for _, container := range pod.Containers {
		for _, usage := range container.Usage {
			if err := rightsizing.Rightsizing(context.Background(), ps.client, usage); err != nil {
				ps.logger.Error("failed while rightsizing", zap.Error(err))
				return nil, err
			}
			container.Usage = nil
		}
	}
	return pod, nil
}

func (ps *podService) GetAllPod(query query.Query) ([]*Pod, error) {
	ps.logger.Debug("rightsizing pod",
		zap.String("id", query.ID),
		zap.Time("start_time", query.StartTime),
		zap.Time("end_time", query.EndTime))

	pods, err := ps.repository.GetAllPod(query)
	if err != nil {
		ps.logger.Error("failed to get pod from database", zap.Error(err))
		return nil, err
	}

	if pods == nil {
		return nil, commonerrors.NotFoundErr("pod", "all")
	}

	for _, pod := range pods {
		for _, container := range pod.Containers {
			for _, usage := range container.Usage {
				if err := rightsizing.Rightsizing(context.Background(), ps.client, usage); err != nil {
					ps.logger.Error("failed while rightsizing", zap.Error(err))
					return nil, err
				}
			}
		}
	}
	return pods, nil
}

func (ps *podService) GetPod(query query.Query) (*Pod, error) {
	ps.logger.Debug("rightsizing pod",
		zap.String("id", query.ID),
		zap.String("namespace", query.Namespace),
		zap.String("pod", query.Name),
		zap.Time("start_time", query.StartTime),
		zap.Time("end_time", query.EndTime))

	pod, err := ps.repository.GetPod(query)
	if err != nil {
		ps.logger.Error("failed to get pod from database", zap.Error(err))
		return nil, err
	}

	if pod == nil {
		return nil, commonerrors.NotFoundErr("pod", query.Name)
	}

	for _, container := range pod.Containers {
		for _, usage := range container.Usage {
			if err := rightsizing.Rightsizing(context.Background(), ps.client, usage); err != nil {
				ps.logger.Error("failed while rightsizing", zap.Error(err))
				return nil, err
			}
		}
	}
	return pod, nil
}

func uniqueName(namespace, name string) string {
	return fmt.Sprintf("pod:%s-%s", namespace, name)
}

func (ps *podService) Forecast(query query.Query) (string, error) {
	var (
		namespace = query.Namespace
		name      = query.Name
		startTime = query.StartTime.Format("2006-01-02T15:04:05")
		endTime   = query.EndTime.Format("2006-01-02T15:04:05")
	)

	task := &tasks.Signature{
		Name: taskName,
		Args: []tasks.Arg{
			{
				Type:  "string",
				Value: namespace,
			},
			{
				Type:  "string",
				Value: name,
			},
			{
				Type:  "string",
				Value: startTime,
			},
			{
				Type:  "string",
				Value: endTime,
			},
		},
		RetryCount: 1,
	}

	taskState, err := ps.worker.SendTaskWithContext(context.Background(), task, uniqueName(namespace, name))
	if err != nil {
		return "", err
	}
	return taskState.TaskUUID, nil
}

func (ps *podService) forecastTask(namespace, name, startTime, endTime string) (string, error) {
	containers, err := ps.repository.Query(context.Background(), namespace, name, startTime, endTime)
	if err != nil {
		return "", err
	}

	if len(containers) == 0 {
		return "", commonerrors.NotFoundErr("pod", name)
	}

	var forecastUsages []*resource.ForecastUsage
	for _, container := range containers {
		forecastUsage := &resource.ForecastUsage{
			Name:  container.Name,
			Usage: make(map[string][]*pb.TimeSeriesDatapoint),
		}
		for _, usage := range container.Usage {
			res, err := ps.client.Forecast(context.Background(), usage.Usage)
			if err != nil {
				ps.logger.Error("failed while forecast", zap.Error(err))
				return "", err
			}
			for _, result := range res.Result {
				forecastUsage.Usage[result.Name] = result.Data
			}
		}
		forecastUsages = append(forecastUsages, forecastUsage)
	}
	encodedUsage, err := resource.EncodeForecastUsage(forecastUsages)
	if err != nil {
		return "", err
	}

	return encodedUsage, nil
}

func (ps *podService) GetForecastStatus(namespace, name string) (string, error) {
	uuid, err := ps.worker.GetUUID(uniqueName(namespace, name))
	if err != nil {
		return "", err
	}
	return ps.getForecastStatus(uuid)
}

func (ps *podService) GetForecastStatusByID(uuid string) (string, error) {
	status, err := ps.worker.GetTaskStatus(uuid)
	if err != nil {
		return "", err
	}
	return status, nil
}

func (ps *podService) getForecastStatus(uuid string) (string, error) {
	status, err := ps.worker.GetTaskStatus(uuid)
	if err != nil {
		return "", err
	}
	return status, nil
}

func (ps *podService) GetForecastResult(namespace, name string) (map[string]*resource.ForecastUsage, error) {
	uuid, err := ps.worker.GetUUID(uniqueName(namespace, name))
	if err != nil {
		return nil, err
	}
	return ps.getForecastResult(uuid)
}

func (ps *podService) GetForecastResultByID(uuid string) (map[string]*resource.ForecastUsage, error) {
	return ps.getForecastResult(uuid)
}

func (ps *podService) getForecastResult(uuid string) (map[string]*resource.ForecastUsage, error) {
	results, err := ps.worker.GetTaskResult(uuid)
	if errors.Is(err, tasks.ErrTaskReturnsNoValue) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	usages := make(map[string]*resource.ForecastUsage)
	if len(results) > 0 {
		value := results[0].Interface().(string)
		decodedUsage, err := resource.DecodeForecastUsage(value)
		if err != nil {
			return nil, err
		}
		for _, usage := range decodedUsage {
			ps.logger.Debug("test", zap.String("name", usage.Name))
			usages[usage.Name] = usage
		}
	} else {
		return nil, commonerrors.NotFoundErr("result not found", uuid)
	}

	return usages, nil
}
