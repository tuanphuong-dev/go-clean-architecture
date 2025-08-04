package main

import (
	"context"
	"flag"
	"fmt"
	"go-clean-arch/common"
	"go-clean-arch/config"
	"go-clean-arch/database"
	"go-clean-arch/middleware"
	authClient "go-clean-arch/modules/auth/client"
	authAPI "go-clean-arch/modules/auth/delivery/api"
	authRepo "go-clean-arch/modules/auth/repository"
	authUC "go-clean-arch/modules/auth/usecase"
	userAPI "go-clean-arch/modules/user/delivery/api"
	userRPC "go-clean-arch/modules/user/delivery/rpc"
	userRepo "go-clean-arch/modules/user/repository"
	userUC "go-clean-arch/modules/user/usecase"
	"go-clean-arch/pkg/cache"
	"go-clean-arch/pkg/email"
	"go-clean-arch/pkg/log"
	"go-clean-arch/proto/pb"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"go-clean-arch/bootstrap"
	emailAPI "go-clean-arch/modules/email/delivery/api"
	emailRPC "go-clean-arch/modules/email/delivery/rpc"
	emailRepo "go-clean-arch/modules/email/repository"
	emailUC "go-clean-arch/modules/email/usecase"
)

func main() {
	// Parse command line flags
	envPath := flag.String("env-file", "", "ENV config file path")
	yamlPath := flag.String("config", "./config/config.yml", "ENV config file path")
	flag.Parse()

	configPaths := []string{*yamlPath}
	if *envPath == "" {
		fmt.Printf("App is starting with config path is '%s' and no load env file\n", *yamlPath)
	} else {
		fmt.Printf("App is starting with config path is '%s' and env path is '%s'...\n", *yamlPath, *envPath)
		configPaths = append(configPaths, *envPath)
	}

	cfg, err := config.Load(configPaths...)
	if err != nil {
		panic(fmt.Errorf("failed to load config: %w", err))
	}

	if err = config.Validate(cfg); err != nil {
		panic(fmt.Errorf("invalid config: %w", err))
	}

	// Initialize logger
	var logger log.Logger
	if cfg.App().IsProduction() {
		logger = log.MustNewProductionLogger(cfg.App().Name(), cfg.App().Version())
	} else {
		logger = log.MustNewDevelopmentLogger()
	}
	defer func() {
		if err := logger.Sync(); err != nil {
			fmt.Printf("Failed to sync logger: %v\n", err)
		}
	}()

	// Set logger for common package using adapter and as default logger
	loggerAdapter := common.NewLoggerAdapter(logger)
	common.SetLogger(loggerAdapter)
	log.SetDefaultLogger(logger)

	logger.Info("Application starting",
		log.String("name", cfg.App().Name()),
		log.String("version", cfg.App().Version()),
		log.String("environment", cfg.App().Environment()),
		log.String("config_path", *yamlPath),
	)

	db, err := database.Connect(cfg.Database(), logger)
	if err != nil {
		logger.Fatal("Failed to connect to database", log.Error(err))
	}

	if err = database.MigrateDB(db); err != nil {
		logger.Fatal("Failed to migrate database", log.Error(err))
	}

	logger.Info("Database connected and migrated successfully")

	// Initialize cache for rate limiting
	cacheConfig := &cache.Config{
		Host:       cfg.Redis().Host(),
		Port:       cfg.Redis().Port(),
		Password:   cfg.Redis().Password(),
		DB:         cfg.Redis().DB(),
		DefaultTTL: 5 * time.Minute,
	}

	cacheFactory := cache.NewCacheFactory(loggerAdapter)
	redisCache, err := cacheFactory.CreateCache(cache.Redis, cacheConfig)
	if err != nil {
		logger.Fatal("Failed to create Redis cache for rate limiting", log.Error(err))
	}
	defer redisCache.Close()

	logger.Info("Redis cache connected successfully for rate limiting")

	// Initialize repositories
	userRepo := userRepo.NewUserRepository(db)
	sessionRepo := authRepo.NewPgUserSessionRepo(db)
	emailTemplateRepo := emailRepo.NewEmailTemplateRepository(db)
	emailLogRepo := emailRepo.NewEmailLogRepository(db)

	// Initialize email templates
	emailTemplateConfig := bootstrap.EmailTemplateConfig{
		AppName:      cfg.App().Name(),
		AppURL:       cfg.Server().Domain(),
		SupportEmail: cfg.App().SystemAdminDefaultEmail(),
	}

	seeder := bootstrap.NewEmailTemplateSeeder(emailTemplateRepo, emailTemplateConfig, logger)
	if err := seeder.Seed(context.Background()); err != nil {
		logger.Error("Failed to initialize email templates", log.Error(err))
		// Don't fail the application, just log the error
	}

	bcryptHasher := common.NewBcryptHasher()
	userUsecase := userUC.NewUserUsecase(userRepo, bcryptHasher)

	// Initialize email usecase
	emailTmplRender := emailUC.NewTemplateRenderer(logger)
	emailUsecase := emailUC.NewEmailUsecase(
		emailLogRepo,
		emailTemplateRepo,
		email.NewMockClient(&email.Config{}, loggerAdapter),
		emailTmplRender,
		logger,
	)

	// Start gRPC server
	go func() {
		rpcAddr := fmt.Sprintf("%s:%d", cfg.RPC().Host(), cfg.RPC().Port())
		lis, err := net.Listen("tcp", rpcAddr)
		if err != nil {
			logger.Fatal("Failed to listen on RPC port",
				log.Int("port", cfg.RPC().Port()),
				log.Error(err),
			)
		}
		grpcServer := grpc.NewServer()
		pb.RegisterUserServiceServer(grpcServer, userRPC.NewUserRPC(userUsecase))
		pb.RegisterEmailServiceServer(grpcServer, emailRPC.NewEmailRPCServer(emailUsecase, logger))

		logger.Info("Starting gRPC server",
			log.String("address", rpcAddr),
			log.Int("port", cfg.RPC().Port()),
		)

		if err := grpcServer.Serve(lis); err != nil {
			logger.Fatal("Failed to serve gRPC server", log.Error(err))
		}
	}()

	grpcAddr := fmt.Sprintf("%s:%d", cfg.RPC().Host(), cfg.RPC().Port())
	grpcConn, err := grpc.NewClient(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatal("Failed to connect to gRPC server",
			log.String("address", grpcAddr),
			log.Error(err),
		)
	}

	userRpcClient := authClient.NewUserRPCClient(grpcConn)
	emailRpcClient := authClient.NewEmailRPCClient(grpcConn)
	jwtProvider := common.NewJWTProvider(cfg.App())
	authUsecase := authUC.NewAuthUsecase(sessionRepo, userRpcClient, emailRpcClient, jwtProvider, bcryptHasher)

	// Initialize dependencies for middlewares
	deps := middleware.Dependencies{
		Cache:       redisCache,
		Logger:      logger,
		JwtProvider: jwtProvider,
		SessionRepo: sessionRepo,
		UserRepo:    userRepo,
	}

	// Create middlewares instance
	middlewares := middleware.NewMiddlewares(deps)

	// Initialize handlers
	userHandler := userAPI.NewUserHandler(userUsecase, middlewares)
	authHandler := authAPI.NewAuthHandler(authUsecase, middlewares)
	emailHandler := emailAPI.NewEmailHandler(emailUsecase, emailTmplRender, logger, middlewares)

	// Disable Gin's default logger and recovery
	gin.DisableConsoleColor()
	gin.SetMode(gin.ReleaseMode)

	// Create Gin server without default middleware
	r := gin.New()

	// Add custom middleware in order
	r.Use(middlewares.CORSWithLogger())
	r.Use(middlewares.RequestIDMiddleware())

	// Add general rate limiting middleware
	r.Use(middlewares.RateLimitWithLogger(middleware.RateLimitConfig{
		WindowSize:  time.Minute,
		MaxRequests: 100,
		KeyPrefix:   "global:",
		SkipPaths:   []string{"/health", "/metrics"},
		// OnLimitReached is omitted - will use default handler
	}))

	r.Use(middlewares.LoggingMiddleware(middleware.LoggerConfig{
		SkipPaths:          []string{"/health", "/metrics"},
		EnableRequestBody:  !cfg.App().IsProduction(),
		EnableResponseBody: false,
		MaxBodySize:        1024,
	}))
	r.Use(gin.Recovery())

	// Register routes
	apiGroup := r.Group("/api/v1")
	userHandler.RegisterRoutes(apiGroup)
	authHandler.RegisterRoutes(apiGroup)
	emailHandler.RegisterRoutes(apiGroup)

	// Add health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "timestamp": time.Now().Unix()})
	})

	// Graceful shutdown setup
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server().Port()),
		Handler: r,
	}

	// Run server in goroutine
	go func() {
		logger.Info("Starting HTTP server",
			log.Int("port", cfg.Server().Port()),
			log.String("host", cfg.Server().Host()),
		)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("HTTP server error", log.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", log.Error(err))
	} else {
		logger.Info("Server exited gracefully")
	}
}
