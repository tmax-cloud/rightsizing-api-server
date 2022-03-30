package worker

import (
	"context"
	"os"
	"reflect"
	"time"

	"github.com/RichardKnop/machinery/v1"
	"github.com/RichardKnop/machinery/v1/config"
	"github.com/RichardKnop/machinery/v1/tasks"
	"github.com/caarlos0/env/v6"
	"github.com/dgraph-io/ristretto"
	"go.uber.org/zap"

	"rightsizing-api-server/internal/api/common/errors"
)

const (
	cnfPath     = "/var/redis-config.yaml"
	consumerTag = "forecast_worker"
)

type envConfig struct {
	Broker  string `env:"BROKER" envDefault:"redis://192.168.9.194:32628"`
	Bankend string `env:"RESULT_BACKEND" envDefault:"redis://192.168.9.194:32628"`
}

type Worker struct {
	cache  *ristretto.Cache
	server *machinery.Server
	worker *machinery.Worker
	logger *zap.Logger
}

func NewWorker(logger *zap.Logger, errCh chan<- error) (*Worker, error) {
	cnf, err := loadConfig()
	if err != nil {
		return nil, err
	}

	server, err := machinery.NewServer(cnf)
	if err != nil {
		return nil, err
	}

	cache, err := newCache()
	if err != nil {
		return nil, err
	}

	worker := server.NewWorker(consumerTag, 1)

	w := &Worker{
		cache:  cache,
		server: server,
		worker: worker,
		logger: logger,
	}

	worker.SetPreTaskHandler(w.preHandler)
	worker.SetErrorHandler(w.errorHandler)
	worker.SetPostTaskHandler(w.postHandler)

	go func() {
		if err := worker.Launch(); err != nil {
			errCh <- err
		}
	}()
	return w, nil
}

func loadConfig() (*config.Config, error) {
	var cnf *config.Config

	if _, err := os.Stat(cnfPath); err != nil {
		envConfig := &envConfig{}
		opts := env.Options{}
		if err := env.Parse(envConfig, opts); err != nil {
			return nil, err
		}

		cnf = &config.Config{
			DefaultQueue:    "machinery_tasks",
			ResultsExpireIn: 600, // 10 min
			Broker:          envConfig.Broker,
			ResultBackend:   envConfig.Bankend,
			Redis: &config.RedisConfig{
				MaxIdle:                3,
				IdleTimeout:            240,
				ReadTimeout:            15,
				WriteTimeout:           15,
				ConnectTimeout:         15,
				NormalTasksPollPeriod:  1000,
				DelayedTasksPollPeriod: 500,
			},
			NoUnixSignals: true,
		}
	} else {
		cnf, err = config.NewFromYaml("", true)
		if err != nil {
			return nil, err
		}
	}

	return cnf, nil
}

func newCache() (*ristretto.Cache, error) {
	cache, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // number of keys to track frequency of (10M).
		MaxCost:     1 << 20, // maximum cost of cache (1GB).
		BufferItems: 64,      // number of keys per Get buffer.
	})
	if err != nil {
		panic(err)
	}

	return cache, nil
}

func (w *Worker) preHandler(sig *tasks.Signature) {
	w.logger.Info("start task",
		zap.String("uuid", sig.UUID),
		zap.String("task", sig.Name),
		zap.Time("startAt", time.Now()),
		zap.Int("retry", sig.RetryCount))
}

func (w *Worker) errorHandler(err error) {
	w.logger.Error("error task", zap.Error(err))
}

func (w *Worker) postHandler(sig *tasks.Signature) {
	w.logger.Info("finish task",
		zap.String("uuid", sig.UUID),
		zap.String("task", sig.Name),
		zap.Time("finishAt", time.Now()))
}

func (w *Worker) SendTaskWithContext(ctx context.Context, task *tasks.Signature, name string) (*tasks.TaskState, error) {
	// 같은 서버 내에서만 caching
	// 서버 장애 발생 등 이유로 꺼져서 다시 켜진 경우 running 중이었던 작업 상태 보존이 매우 어려움
	uuid, exist := w.cache.Get(name)
	if exist {
		taskState, err := w.getTask(uuid.(string))
		if err != nil {
			return nil, err
		}
		return taskState, nil
	}

	result, err := w.server.SendTaskWithContext(ctx, task)
	if err != nil {
		return nil, err
	}
	taskState := result.GetState()

	w.cache.SetWithTTL(name, taskState.TaskUUID, 1, time.Minute*10)

	return taskState, nil
}

func (w *Worker) GetUUID(name string) (string, error) {
	uuid, exist := w.cache.Get(name)
	if !exist {
		return "", errors.NotFoundErr("uuid", name)
	}
	return uuid.(string), nil
}

func (w *Worker) getTask(uuid string) (*tasks.TaskState, error) {
	backend := w.server.GetBackend()
	return backend.GetState(uuid)
}

func (w *Worker) GetTaskStatus(uuid string) (string, error) {
	taskState, err := w.getTask(uuid)
	if err != nil {
		return "", err
	}
	return taskState.State, nil
}

func (w *Worker) GetTaskResult(uuid string) ([]reflect.Value, error) {
	backend := w.server.GetBackend()
	taskState, err := backend.GetState(uuid)
	if err != nil {
		return nil, err
	}
	if taskState.IsSuccess() {
		return tasks.ReflectTaskResults(taskState.Results)
	}
	return nil, tasks.ErrTaskReturnsNoValue
}

func (w *Worker) RegisterTask(name string, task interface{}) error {
	return w.server.RegisterTask(name, task)
}

func (w *Worker) RegisterTasks(tasks map[string]interface{}) error {
	return w.server.RegisterTasks(tasks)
}

func (w *Worker) Stop(ctx context.Context) {
	w.worker.Quit()
	w.cache.Close()
}
