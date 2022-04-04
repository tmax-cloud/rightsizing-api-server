package app

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	swagger "github.com/arsmn/fiber-swagger/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/etag"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/monitor"
	"github.com/gofiber/fiber/v2/middleware/pprof"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"gorm.io/gorm"

	"rightsizing-api-server/cmd/api-server/app/options"
	"rightsizing-api-server/internal/api/pod"
	"rightsizing-api-server/internal/api/vm"
	cache2 "rightsizing-api-server/internal/cache"
	db "rightsizing-api-server/internal/database"
	grpcclient "rightsizing-api-server/internal/grpc"
	"rightsizing-api-server/internal/worker"
)

type Server struct {
	app        *fiber.App
	client     *grpcclient.Client
	db         *gorm.DB
	grpcClient *grpc.ClientConn
	worker     *worker.Worker
	logger     *zap.Logger
}

func NewServer(opts *options.Options, logger *zap.Logger, errCh chan<- error) *Server {
	// connect TimescaleDB (postgres)
	db, err := db.Connect()
	if err != nil {
		logger.Fatal("Unable to connect to TimescaleDB", zap.Error(err))
	}
	// connect rightsizing grpc server
	grpcConn, err := grpc.Dial(fmt.Sprintf("%s:%s", *opts.GrpcHost, *opts.GrpcPort), grpc.WithInsecure())
	if err != nil {
		logger.Fatal("Unable to connect to grpc client", zap.Error(err))
	}
	client := grpcclient.NewClient(grpcConn)
	// worker
	cache, err := cache2.NewCache()
	if err != nil {
		logger.Fatal("Unable to init cache", zap.Error(err))
	}

	worker, err := worker.NewWorker(cache, logger, errCh)
	if err != nil {
		logger.Fatal("Unable to initialize worker", zap.Error(err))
	}

	app := fiber.New(fiber.Config{
		AppName: "Rightsizing API Server",
		Prefork: false,
		// JSONEncoder: ffjson.Marshal,
	})

	app.Use(cors.New())
	app.Use(compress.New())
	app.Use(compress.New())
	app.Use(etag.New())
	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(fiberlogger.New(fiberlogger.Config{
		Format:     "[${time}] [${ip}:${port}] ${status} - ${latency} ${method} ${path}\n",
		TimeFormat: "2006-01-02 15:04:05",
	}))

	if *opts.Mode == "debug" {
		app.Use(pprof.New())
	}

	// pod
	podLogger := logger.Named("pod")
	podRepository := pod.NewPodRepository(db)
	podService := pod.NewPodService(cache, worker, client, podRepository, podLogger)
	pod.PodRouter(app.Group("/api/v1/"), podService, podLogger)
	// vm
	vmLogger := logger.Named("vm")
	vmRepository := vm.NewVMRepository(db)
	vmService := vm.NewVMService(cache, worker, client, vmRepository, vmLogger)
	vm.VMRouter(app.Group("/api/v1/"), vmService, vmLogger)

	app.Get("/dashboard", monitor.New())

	app.Get("/swagger/*", swagger.Handler) // default

	app.All("*", func(c *fiber.Ctx) error {
		errorMessage := fmt.Sprintf("Route '%s' does not exist in this API!", c.OriginalURL())

		return c.Status(fiber.StatusNotFound).JSON(&fiber.Map{
			"status":  "fail",
			"message": errorMessage,
		})
	})

	return &Server{
		app:        app,
		client:     client,
		db:         db,
		grpcClient: grpcConn,
		worker:     worker,
		logger:     logger,
	}
}

func (app *Server) Listen(port int, certFile, keyFile *string) error {
	app.logger.Info("Starting Rightsizing api-server ...")

	address := fmt.Sprintf(":%d", port)
	if certFile != nil && keyFile != nil {
		if *certFile != "" && *keyFile != "" {
			return app.app.ListenTLS(address, *certFile, *keyFile)
		}
	}
	return app.app.Listen(address)
}

func (app *Server) Shutdown(parentCtx context.Context) error {
	g, ctx := errgroup.WithContext(parentCtx)
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	g.Go(func() error {
		if err := app.app.Shutdown(); err != nil {
			return err
		}
		return nil
	})
	g.Go(func() error {
		app.worker.Stop(ctx)
		// grpc connection은 반드시 worker 다 끝나고 해야함.
		if err := app.grpcClient.Close(); err != nil {
			return err
		}
		return nil
	})
	if err := g.Wait(); err != nil {
		return err
	}
	return nil
}

func Run(opts *options.Options, logger *zap.Logger) error {
	// Start api-server
	apiServerError := make(chan error)

	server := NewServer(opts, logger, apiServerError)

	go func() {
		if err := server.Listen(*opts.Port, opts.CertFile, opts.KeyFile); err != nil && err != http.ErrServerClosed {
			logger.Fatal("RunTLS for api-server failed", zap.Error(err))
			apiServerError <- err
		}
	}()

	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		logger.Info("Shutdown server ...")

		ctx := context.Background()
		if err := server.Shutdown(ctx); err != nil {
			logger.Fatal("close api-server failed", zap.Error(err))
			return err
		}
	case err := <-apiServerError:
		return err
	}

	return nil
}
