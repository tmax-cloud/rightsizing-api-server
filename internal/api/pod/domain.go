package pod

import (
	"context"
	grpcclient "rightsizing-api-server/internal/grpc"

	"rightsizing-api-server/internal/api/common/query"
	"rightsizing-api-server/internal/api/common/resource"
	"rightsizing-api-server/internal/models"
)

type PodRepository interface {
	GetAllPodQuota(query query.Query) ([]*Pod, error)
	GetAllPod(query query.Query) ([]*Pod, error)
	GetPod(query query.Query) (*Pod, error)
	Query(ctx context.Context, naemspace, name, startTime, endTime string) ([]*Container, error)
}

type PodService interface {
	GetClusterInfo() (interface{}, error)
	GetAllPod(query query.Query) ([]*Pod, error)
	GetPod(query query.Query) (*Pod, error)
	GetForecastStatusByID(uuid string) (string, error)
	GetForecastResultByID(uuid string) (map[string]*resource.ForecastUsage, error)
	GetForecastStatus(namespace, name string) (string, error)
	GetForecastResult(namespace, name string) (map[string]*resource.ForecastUsage, error)
	Forecast(query query.Query) (string, error)
}

type Pod struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	// Container information
	Containers []*Container `json:"containers,omitempty"`
	// total usage infromation
	Usages map[string]*resource.ResourceUsageInfo `json:"usage,omitempty"`
}

func (pod *Pod) Rightsizing(client *grpcclient.Client) error {
	for _, container := range pod.Containers {
		for name, usage := range container.Usage {
			if len(usage.Usage) > 100 {
				resp, err := client.Rightsizing(context.Background(), usage.Usage)
				if err != nil {
					return err
				}
				usage.OptimizedUsage = resp.Result
				pod.Usages[name].OptimizedUsage += resp.Result
			}
		}
	}
	return nil
}

type Container struct {
	Namespace string `json:"namespace"`
	Pod       string `json:"pod_name"`
	Name      string `json:"container_name"`
	// Resource usage list
	Usage map[string]*resource.ResourceUsageInfo `json:"usages,omitempty"`
}

func (c Container) UniquePod() string {
	return c.Namespace + "_" + c.Pod
}

func (c Container) UniqueName() string {
	return "container/" + c.UniquePod() + "_" + c.Name
}

func UniqueContainerName(c *models.Container) string {
	return c.Namespace + "_" + c.Pod + "_" + c.Name
}

func UniqueContainerNameByField(namespace, pod, name string) string {
	return namespace + "_" + pod + "_" + name
}
