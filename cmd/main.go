package main

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	_ "github.com/ruziba3vich/online_compiler_api_gateway/docs"
	"github.com/ruziba3vich/online_compiler_api_gateway/genprotos/genprotos/compiler_service"
	"github.com/ruziba3vich/online_compiler_api_gateway/internal/db"
	handler "github.com/ruziba3vich/online_compiler_api_gateway/internal/http"
	"github.com/ruziba3vich/online_compiler_api_gateway/internal/middleware"
	"github.com/ruziba3vich/online_compiler_api_gateway/internal/repos"
	"github.com/ruziba3vich/online_compiler_api_gateway/internal/service"
	"github.com/ruziba3vich/online_compiler_api_gateway/internal/storage"
	"github.com/ruziba3vich/online_compiler_api_gateway/pkg/config"
	"github.com/ruziba3vich/online_compiler_api_gateway/pkg/lgg"
	logger "github.com/ruziba3vich/prodonik_lgger"
	limiter "github.com/ruziba3vich/prodonik_rl"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/gorm"
)

func main() {
	fx.New(
		fx.Provide(
			config.NewConfig,
			newRedisClient,
			newRateLimiter,
			lgg.NewLogger,
			newMiddleware,
			NewLogger,
			NewDB,
			storage.NewLangStorage,
			service.NewLangService,
			handler.NewLangHandler,
			newPythonGRPCClient,
			newJavaGRPCClient,
			newCppGRPCClient,
			newService,
			handler.NewHandler,
			newGinRouter,
			newHTTPServer,
		),
		fx.Invoke(registerRoutes),
		fx.Invoke(startServer),
	).Run()
}

func NewDB(cfg *config.Config) (*gorm.DB, error) {
	return db.NewDB(cfg.LangStorageFilePath)
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

func newCppGRPCClient(cfg *config.Config, logger *lgg.Logger) (repos.Cpp, error) {
	conn, err := grpc.NewClient(cfg.CppService, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("Failed to connect to Cpp Executor Service", map[string]any{"error": err})
		return nil, err
	}
	logger.Info("Connected to gRPC service", map[string]any{"address": cfg.CppService})
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

func registerRoutes(router *gin.Engine, handler *handler.Handler, langHandler *handler.LangHandler, middleware *middleware.MidWare) {
	router.Use(middleware.CORS())
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r := router.Group("/api/v1")
	r.Use(middleware.RateLimit())
	r.GET("/execute", handler.HandleWebSocket)
	r.GET("/languages", langHandler.GetAllLanguages)
	r.POST("/create", langHandler.CreateLanguage)
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

func NewLogger(cfg *config.Config) (*logger.Logger, error) {
	return logger.NewLogger(cfg.LogsFilePath)
}

func newRedisClient(cfg *config.Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.RedisCfg.Host + ":" + cfg.RedisCfg.Port,
		Password: cfg.RedisCfg.Password,
		DB:       cfg.RedisCfg.DB,
	})
}

func newRateLimiter(cfg *config.Config, clinent *redis.Client) *limiter.TokenBucketLimiter {
	return limiter.NewTokenBucketLimiter(clinent, cfg.RLCnfg.MaxTokens, cfg.RLCnfg.RefillRate, cfg.RLCnfg.Window)
}

func newMiddleware(limiter *limiter.TokenBucketLimiter, logger *logger.Logger) *middleware.MidWare {
	return middleware.NewMidWare(logger, limiter)
}
