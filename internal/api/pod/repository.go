package pod

import (
	"context"

	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"

	"rightsizing-api-server/internal/api/common/query"
	"rightsizing-api-server/internal/api/common/resource"
	"rightsizing-api-server/internal/models"
)

type podRepository struct {
	db *gorm.DB
}

var _ PodRepository = (*podRepository)(nil)

func NewPodRepository(db *gorm.DB) PodRepository {
	return &podRepository{
		db: db,
	}
}

func (r *podRepository) GetAllPodQuota(query query.Query) ([]*Pod, error) {
	var (
		startTime = query.StartTime.Format("2006-01-02T15:04:05")
		endTime   = query.EndTime.Format("2006-01-02T15:04:05")
	)

	containers, err := r.QueryResourceQuota(context.Background(), "", "", startTime, endTime)
	if err != nil {
		return nil, err
	}
	podMap := make(map[string]*Pod)
	for _, container := range containers {
		uniquePod := uniqueName(container.Namespace, container.Pod)
		if _, exist := podMap[uniquePod]; !exist {
			podMap[uniquePod] = &Pod{
				Namespace:  container.Namespace,
				Name:       container.Pod,
				Containers: make([]*Container, 0),
			}
		}
		podMap[uniquePod].Containers = append(podMap[uniquePod].Containers, container)
	}

	var pods []*Pod
	for _, pod := range podMap {
		pods = append(pods, pod)
	}
	return pods, nil
}

func (r *podRepository) GetPodQuota(query query.Query) (*Pod, error) {
	var (
		namespace = query.Namespace
		name      = query.Name
		startTime = query.StartTime.Format("2006-01-02T15:04:05")
		endTime   = query.EndTime.Format("2006-01-02T15:04:05")
	)

	containerMap, err := r.QueryResourceQuota(context.Background(), namespace, name, startTime, endTime)
	if err != nil {
		return nil, err
	}

	var containers []*Container
	for _, container := range containerMap {
		containers = append(containers, container)
	}

	return &Pod{
		Namespace:  query.Namespace,
		Name:       query.Name,
		Containers: containers,
	}, nil
}

func (r *podRepository) QueryResourceQuota(
	ctx context.Context, namespace, name, startTime, endTime string) (map[string]*Container, error) {
	var (
		containerRequest []models.ContainerQuota
		containerLimit   []models.ContainerQuota
		// query
		requestQuery = requestQuotaQuery + allQuotaQuery
		limitQuery   = limitQuotaQuery + allQuotaQuery
	)

	if namespace != "" && name != "" {
		requestQuery = requestQuotaQuery + targetQuotaQuery
		limitQuery = limitQuotaQuery + targetQuotaQuery
	}

	ctxDB := r.db.WithContext(ctx)
	g, _ := errgroup.WithContext(ctx)
	g.Go(func() error {
		db := ctxDB.Raw(requestQuery)
		if namespace != "" && name != "" {
			db = ctxDB.Raw(requestQuery, namespace, name)
		}
		err := db.Find(&containerRequest).Error
		if err != nil {
			return err
		}
		return nil
	})
	g.Go(func() error {
		db := ctxDB.Raw(limitQuery)
		if namespace != "" && name != "" {
			db = ctxDB.Raw(limitQuery, namespace, name)
		}
		err := db.Find(&containerLimit).Error
		if err != nil {
			return err
		}
		return nil
	})
	if err := g.Wait(); err != nil {
		return nil, err
	}

	containerUsages, err := r.Query(context.Background(), "", "", startTime, endTime)
	if err != nil {
		return nil, err
	}

	containers := make(map[string]*Container)
	for _, request := range containerRequest {
		name := UniqueContainerNameByField(request.Namespace, request.Pod, request.Name)
		if _, exist := containers[name]; !exist {
			containers[name] = &Container{
				Namespace: request.Namespace,
				Pod:       request.Pod,
				Name:      request.Name,
				Request:   make(map[string]float64),
				Limit:     make(map[string]float64),
			}
		}
		containers[name].Request[request.Resource] = request.Value
	}
	for _, limit := range containerLimit {
		name := UniqueContainerNameByField(limit.Namespace, limit.Pod, limit.Name)
		if _, exist := containers[name]; !exist {
			containers[name] = &Container{
				Namespace: limit.Namespace,
				Pod:       limit.Pod,
				Name:      limit.Name,
				Request:   make(map[string]float64),
				Limit:     make(map[string]float64),
			}
		}
		containers[name].Limit[limit.Resource] = limit.Value
	}

	for _, usage := range containerUsages {
		name := UniqueContainerNameByField(usage.Namespace, usage.Pod, usage.Name)
		if _, exist := containers[name]; !exist {
			containers[name] = &Container{
				Namespace:    usage.Namespace,
				Pod:          usage.Pod,
				Name:         usage.Name,
				Usage:        make(map[string]*resource.ResourceUsageInfo),
				CurrentUsage: make(map[string]float64),
				Request:      make(map[string]float64),
				Limit:        make(map[string]float64),
			}
		}
		containers[name].Usage = usage.Usage
		containers[name].CurrentUsage = usage.CurrentUsage
	}
	return containers, nil
}

func (r *podRepository) GetAllPod(query query.Query) ([]*Pod, error) {
	var (
		startTime = query.StartTime.Format("2006-01-02T15:04:05")
		endTime   = query.EndTime.Format("2006-01-02T15:04:05")
	)

	containers, err := r.Query(context.Background(), "", "", startTime, endTime)
	if err != nil {
		return nil, err
	}

	var podMap map[string]*Pod
	for _, container := range containers {
		name := uniqueName(container.Namespace, container.Pod)
		if _, exist := podMap[name]; !exist {
			podMap[name] = &Pod{
				Namespace:  container.Namespace,
				Name:       container.Pod,
				Containers: make([]*Container, 0),
			}
		}
		podMap[name].Containers = append(podMap[name].Containers, container)
	}

	var pods []*Pod
	for _, pod := range podMap {
		pods = append(pods, pod)
	}
	return pods, nil
}

func (r *podRepository) GetPod(query query.Query) (*Pod, error) {
	var (
		namespace = query.Namespace
		name      = query.Name
		startTime = query.StartTime.Format("2006-01-02T15:04:05")
		endTime   = query.EndTime.Format("2006-01-02T15:04:05")
	)

	containers, err := r.Query(context.Background(), namespace, name, startTime, endTime)
	if err != nil {
		return nil, err
	}

	return &Pod{
		Namespace:  query.Namespace,
		Name:       query.Name,
		Containers: containers,
	}, nil
}

func (r *podRepository) Query(ctx context.Context, namespace, name, startTime, endTime string) ([]*Container, error) {
	var (
		numMetric   = ContainerMetricTables.Len()
		metricNames = ContainerMetricTables.GetMetricNames()
		// goroutine and thread safe
		ctxDB = r.db.WithContext(ctx)
	)

	containerMetricUsages := make([][]models.Container, numMetric)
	g, _ := errgroup.WithContext(ctx)
	for i := 0; i < numMetric; i++ {
		idx := i
		g.Go(func() error {
			db := ctxDB.Scopes(ContainerMetricTables.GetIDTable(idx)).
				Preload("Usage", func(db *gorm.DB) *gorm.DB {
					return db.Table(ContainerMetricTables.GetMetricTableName(idx)).
						Where("value != 'NaN'").
						Where("bucket >= ? AND bucket <= ?", startTime, endTime).
						Order("bucket")
				})
			if namespace != "" && name != "" {
				db = db.Where("namespace=? AND pod=?", namespace, name)
			}
			err := db.Where("container!='POD' AND container != ''").
				Find(&containerMetricUsages[idx]).
				Error
			if err != nil {
				return err
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	containerMap := make(map[string]*Container)
	for metricIdx := 0; metricIdx < numMetric; metricIdx++ {
		metricName := metricNames[metricIdx]
		for _, containerUsage := range containerMetricUsages[metricIdx] {
			name := UniqueContainerName(&containerUsage)
			if _, exist := containerMap[name]; !exist {
				containerMap[name] = &Container{
					Namespace:    containerUsage.Namespace,
					Pod:          containerUsage.Pod,
					Name:         containerUsage.Name,
					Usage:        make(map[string]*resource.ResourceUsageInfo),
					CurrentUsage: make(map[string]float64),
				}
			}
			usage := resource.NewResourceUsage(metricName, containerUsage.Usage)
			containerMap[name].Usage[metricName] = usage
			containerMap[name].CurrentUsage[metricName] = usage.CurrentUsage
		}
	}

	var containers []*Container
	for _, container := range containerMap {
		containers = append(containers, container)
	}
	return containers, nil
}
