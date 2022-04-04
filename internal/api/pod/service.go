package pod

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/RichardKnop/machinery/v1/tasks"
	"go.uber.org/zap"

	commonerrors "rightsizing-api-server/internal/api/common/errors"
	"rightsizing-api-server/internal/api/common/query"
	"rightsizing-api-server/internal/api/common/resource"
	"rightsizing-api-server/internal/cache"
	grpcclient "rightsizing-api-server/internal/grpc"
	"rightsizing-api-server/internal/worker"
	pb "rightsizing-api-server/proto"
)

const (
	taskName       = "pod_forecast"
	overallInfoKey = "overallInfo"
)

type podService struct {
	cache      *cache.Cache
	worker     *worker.Worker
	client     *grpcclient.Client
	repository PodRepository
	logger     *zap.Logger
}

var _ PodService = (*podService)(nil)

func NewPodService(
	cache *cache.Cache,
	worker *worker.Worker,
	client *grpcclient.Client,
	r PodRepository,
	logger *zap.Logger) PodService {
	s := &podService{
		cache:      cache,
		worker:     worker,
		client:     client,
		repository: r,
		logger:     logger,
	}

	worker.RegisterTask(taskName, s.forecastTask)

	return s
}

func GetStatus(a, b float64) string {
	const threshold = 0.2
	eps := math.Abs(a-b) / b
	if eps < threshold {
		return "optimized"
	} else if a < b {
		return "underallocated"
	}
	return "overallocated"
}

func (ps *podService) GetClusterInfo() (interface{}, error) {
	var pods []*Pod
	ps.logger.Debug("GetClusterInfo")

	// 리소스 사용량 history 제외하고 저장되어 있음.
	item, exist := ps.cache.Get(overallInfoKey)
	if !exist {
		var err error
		query := query.Query{
			StartTime: time.Now().AddDate(0, 0, -7),
			EndTime:   time.Now(),
		}

		pods, err = ps.GetAllPod(query)
		if err != nil {
			ps.logger.Error("failed to get pod from database", zap.Error(err))
			return nil, err
		}

		for _, pod := range pods {
			for _, container := range pod.Containers {
				for _, usage := range container.Usage {
					usage.Usage = nil
				}
			}
		}
		ps.cache.SetWithTTL(overallInfoKey, pods, time.Minute*5)
	} else {
		pods = item.([]*Pod)
	}

	if pods == nil {
		return nil, commonerrors.NotFoundErr("pod", "all")
	}

	averageUsages := map[string]float64{
		"cpu":    0,
		"memory": 0,
	}
	resourceStatus := map[string]map[string]int{
		"cpu": {
			"optimized":      0,
			"underallocated": 0,
			"overallocated":  0,
		},
		"memory": {
			"optimized":      0,
			"underallocated": 0,
			"overallocated":  0,
		},
	}

	for _, pod := range pods {
		for name, usage := range pod.Usages {
			standard := usage.GetStandardQuota()
			averageUsages[name] += usage.CurrentUsage
			if standard != -1 {
				status := GetStatus(usage.CurrentUsage, standard)
				resourceStatus[name][status] += 1
			}
		}
	}

	for name, usage := range averageUsages {
		averageUsages[name] = usage / float64(len(pods))
	}

	result := map[string]map[string]float64{
		"cpu":    make(map[string]float64),
		"memory": make(map[string]float64),
	}

	for resourceName, _ := range result {
		result[resourceName]["average"] = averageUsages[resourceName]
		for status, count := range resourceStatus[resourceName] {
			result[resourceName][status] = float64(count)
		}
	}

	return result, nil
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
		if err := pod.Rightsizing(ps.client); err != nil {
			return nil, err
		}
	}

	sort.Slice(pods, func(i, j int) bool {
		if pods[i].Namespace == pods[j].Namespace {
			return pods[i].Name < pods[j].Name
		}
		return pods[i].Namespace < pods[j].Namespace
	})
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

	if err := pod.Rightsizing(ps.client); err != nil {
		return nil, err
	}

	pod.Usages = map[string]*resource.ResourceUsageInfo{
		"cpu":    {ResourceName: "cpu"},
		"memory": {ResourceName: "memory"},
	}
	for _, container := range pod.Containers {
		for name, usage := range container.Usage {
			pod.Usages[name].Request += usage.Request
			pod.Usages[name].Limit += usage.Limit
			pod.Usages[name].CurrentUsage += usage.CurrentUsage
			pod.Usages[name].OptimizedUsage += usage.OptimizedUsage
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
