package vm

import (
	"context"
	"errors"
	"fmt"

	"github.com/RichardKnop/machinery/v1/tasks"
	"go.uber.org/zap"

	commonerrors "rightsizing-api-server/internal/api/common/errors"
	"rightsizing-api-server/internal/api/common/query"
	"rightsizing-api-server/internal/api/common/resource"
	"rightsizing-api-server/internal/api/common/rightsizing"
	grpcclient "rightsizing-api-server/internal/grpc"
	"rightsizing-api-server/internal/worker"
	pb "rightsizing-api-server/proto"
)

const (
	taskName = "vm_forecast"
)

type vmService struct {
	worker     *worker.Worker
	client     *grpcclient.Client
	repository VMRepository
	logger     *zap.Logger
}

var _ VMService = (*vmService)(nil)

func NewVMService(
	worker *worker.Worker,
	client *grpcclient.Client,
	repository VMRepository,
	logger *zap.Logger) VMService {

	s := &vmService{
		worker:     worker,
		client:     client,
		repository: repository,
		logger:     logger,
	}

	worker.RegisterTask(taskName, s.forecastTask)
	return s
}

func (s *vmService) GetVm(query query.Query) (*Vm, error) {
	s.logger.Debug("rightsizing vm",
		zap.String("id", query.ID),
		zap.String("vm", query.Name),
		zap.Time("start_time", query.StartTime),
		zap.Time("end_time", query.EndTime))

	vm, err := s.repository.GetVm(query)
	if err != nil {
		return nil, err
	}

	for _, usage := range vm.Usage {
		if err := rightsizing.Rightsizing(context.Background(), s.client, usage); err != nil {
			s.logger.Error("failed while rightsizing", zap.Error(err))
			return nil, err
		}
	}

	return vm, nil
}

func uniqueName(name string) string {
	return fmt.Sprintf("pod:%s-%s", name)
}

func (s *vmService) Forecast(query query.Query) (string, error) {
	var (
		name      = query.Name
		startTime = query.StartTime.Format("2006-01-02T15:04:05")
		endTime   = query.EndTime.Format("2006-01-02T15:04:05")
	)

	task := &tasks.Signature{
		Name: taskName,
		Args: []tasks.Arg{
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
	taskState, err := s.worker.SendTaskWithContext(context.Background(), task, uniqueName(name))
	if err != nil {
		return "", err
	}
	return taskState.TaskUUID, nil
}

func (s *vmService) forecastTask(name, startTime, endTime string) (string, error) {
	vms, err := s.repository.Query(context.Background(), name, startTime, endTime)
	if err != nil {
		return "", err
	}
	if len(vms) == 0 {
		return "", commonerrors.NotFoundErr("vm", name)
	} else if len(vms) > 1 {
		return "", commonerrors.NotUniqueErr("vm", name)
	}

	vm := vms[0]
	forecastUsage := &resource.ForecastUsage{
		Name:  vm.Name,
		Usage: make(map[string][]*pb.TimeSeriesDatapoint),
	}

	for _, usage := range vm.Usage {
		res, err := s.client.Forecast(context.Background(), usage.Usage)
		if err != nil {
			s.logger.Error("failed while forecast", zap.Error(err))
			return "", err
		}
		for _, result := range res.Result {
			forecastUsage.Usage[result.Name] = result.Data
		}
	}
	forecastUsages := []*resource.ForecastUsage{
		forecastUsage,
	}

	encodedUsage, err := resource.EncodeForecastUsage(forecastUsages)
	if err != nil {
		return "", err
	}

	return encodedUsage, nil
}

func (s *vmService) GetForecastStatus(name string) (string, error) {
	uuid, err := s.worker.GetUUID(uniqueName(name))
	if err != nil {
		return "", err
	}
	return s.getForecastStatus(uuid)
}

func (s *vmService) GetForecastStatusByID(uuid string) (string, error) {
	status, err := s.worker.GetTaskStatus(uuid)
	if err != nil {
		return "", err
	}
	return status, nil
}

func (s *vmService) getForecastStatus(uuid string) (string, error) {
	status, err := s.worker.GetTaskStatus(uuid)
	if err != nil {
		return "", err
	}
	return status, nil
}

func (s *vmService) GetForecastResult(name string) (map[string]*resource.ForecastUsage, error) {
	uuid, err := s.worker.GetUUID(uniqueName(name))
	if err != nil {
		return nil, err
	}
	return s.getForecastResult(uuid)
}

func (s *vmService) GetForecastResultByID(uuid string) (map[string]*resource.ForecastUsage, error) {
	return s.getForecastResult(uuid)
}

func (s *vmService) getForecastResult(uuid string) (map[string]*resource.ForecastUsage, error) {
	results, err := s.worker.GetTaskResult(uuid)
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
			s.logger.Debug("test", zap.String("name", usage.Name))
			usages[usage.Name] = usage
		}
	} else {
		return nil, commonerrors.NotFoundErr("result not found", uuid)
	}

	return usages, nil
}
