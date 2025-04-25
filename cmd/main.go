package main

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ruziba3vich/online_compiler_api_gateway/genprotos/genprotos/compiler_service"
	handler "github.com/ruziba3vich/online_compiler_api_gateway/internal/http"
	"github.com/ruziba3vich/online_compiler_api_gateway/internal/repos"
	"github.com/ruziba3vich/online_compiler_api_gateway/internal/service"
	"github.com/ruziba3vich/online_compiler_api_gateway/internal/storage"
	"github.com/ruziba3vich/online_compiler_api_gateway/pkg/config"
	"github.com/ruziba3vich/online_compiler_api_gateway/pkg/lgg"
	logger "github.com/ruziba3vich/prodonik_lgger"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	fx.New(
		fx.Provide(
			config.NewConfig,
			lgg.NewLogger,
			logger.NewLogger,
			storage.NewLanguageStorage,
			service.NewLangService,
			handler.NewLangHandler,
			newPythonGRPCClient,
			newJavaGRPCClient,
			newService,
			handler.NewHandler,
			newGinRouter,
			newHTTPServer,
		),
		fx.Invoke(registerRoutes),
		fx.Invoke(startServer),
	).Run()
}

func newPythonGRPCClient(cfg *config.Config, logger *lgg.Logger) (repos.Python, error) {
	conn, err := grpc.NewClient(cfg.PythonService, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("Failed to connect to Python Executor Service", map[string]any{"error": err})
		return nil, err
	}
	logger.Info("Connected to gRPC service", map[string]any{"address": cfg.PythonService})
	return compiler_service.NewCodeExecutorClient(conn), nil
}

func newJavaGRPCClient(cfg *config.Config, logger *lgg.Logger) (repos.Java, error) {
	conn, err := grpc.NewClient(cfg.JavaService, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("Failed to connect to Java Executor Service", map[string]any{"error": err})
		return nil, err
	}
	logger.Info("Connected to gRPC service", map[string]any{"address": cfg.JavaService})
	return compiler_service.NewCodeExecutorClient(conn), nil
}

func newService(logger *lgg.Logger, pythonClient repos.Python, javaClient repos.Java) *service.Service {
	return service.NewService(&sync.Mutex{}, *logger, pythonClient, javaClient)
}

func newGinRouter() *gin.Engine {
	return gin.Default()
}

func newHTTPServer(cfg *config.Config) *http.Server {
	return &http.Server{
		Addr: ":" + cfg.GatewayPort,
	}
}

func registerRoutes(router *gin.Engine, handler *handler.Handler, langHandler *handler.LangHandler) {
	router.GET("/execute", handler.HandleWebSocket)
	router.GET("/languages", langHandler.GetAllLanguages)
	router.POST("/create", langHandler.CreateLanguage)
}

func startServer(lc fx.Lifecycle, server *http.Server, router *gin.Engine, logger *lgg.Logger, cfg *config.Config) {
	server.Handler = router

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			logger.Info("Starting API Gateway", map[string]any{"address": cfg.GatewayPort})
			go func() {
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					logger.Error("Failed to run API Gateway", map[string]any{"error": err})
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			logger.Info("Shutting down API Gateway", map[string]any{"address": cfg.GatewayPort})
			ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
			return server.Shutdown(ctx)
		},
	})
}
