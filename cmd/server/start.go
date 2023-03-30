package main

import (
	"context"
	"fmt"
	"go-example/docs"
	v1 "go-example/internal/api/v1"
	"go-example/internal/config"
	"go-example/internal/entities"
	"go-example/internal/errors"
	"go-example/internal/log"
	internalTrace "go-example/internal/trace"
	"os"
	"os/signal"
	"time"

	// "go-example/internal/log"
	"strings"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cobra"
	swaggerFiles "github.com/swaggo/files"
	swagger "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	configPath string
	startCmd   = &cobra.Command{
		Use:   "start",
		Short: "start server",
		Long:  `start server, default port is 5000`,
		Run:   startServer,
	}
	enablePprof bool
)

func init() {
	cobra.OnInitialize(initConfig, initLogger)
	rootCmd.AddCommand(startCmd)
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "config file (default is $PWD/config/default.yaml)")
	startCmd.PersistentFlags().Int("port", 5000, "Port to run Application server on")
	startCmd.PersistentFlags().BoolVarP(&enablePprof, "pprof", "p", false, "enable pprof mode (default: false)")
	config.Viper().BindPFlag("port", startCmd.PersistentFlags().Lookup("port"))
}

func initConfig() {
	defer log.Sync()
	if len(configPath) != 0 {
		config.Viper().SetConfigFile(configPath)
	} else {
		config.Viper().AddConfigPath("./config")
		config.Viper().SetConfigName("default")
	}
	config.Viper().SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	config.Viper().AutomaticEnv()
	if err := config.Viper().ReadInConfig(); err != nil {
		log.Fatal(
			fmt.Sprintf("Load config from file [%s]: %v", config.Viper().ConfigFileUsed(), err))
	}
	config.Parse()
}

func initLogger() {
	log.ResetDefault(log.New(os.Stderr, config.Default.Otel.Log))
}

func startServer(cmd *cobra.Command, agrs []string) {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	observabilityCloser := initObservability(ctx)
	defer observabilityCloser(ctx)

	tracer := otel.Tracer("test-tracer")

	// Attributes represent additional key-value descriptors that can be bound
	// to a metric observer or recorder.
	commonAttrs := []attribute.KeyValue{
		attribute.String("attrA", "chocolate"),
		attribute.String("attrB", "raspberry"),
		attribute.String("attrC", "vanilla"),
	}

	// work begins
	ctx, span := tracer.Start(
		ctx,
		"CollectorExporter-Example",
		trace.WithAttributes(commonAttrs...))
	defer span.End()
	for i := 0; i < 10; i++ {
		_, iSpan := tracer.Start(ctx, fmt.Sprintf("Sample-%d", i))
		log.Info(fmt.Sprintf("Doing really hard work (%d / 10)\n", i+1))

		<-time.After(time.Second)
		iSpan.End()
	}

	log.Info("Start http-server")
	db, err := gorm.Open(postgres.Open(config.Default.Database.URL))
	if err != nil {
		log.Fatal(fmt.Sprint("Failed to connect database: ", err))
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Can't connect database")
	}
	sqlDB.SetMaxOpenConns(int(config.Default.Database.Pool.Max))
	defer func() {
		sqlDB.Close()
		log.Info("Closed db connection")
	}()
	go entities.AutoMigrate(db)
	setupDoc()
	router := gin.New()
	router.Use(errors.GinError())
	router.Use(gin.Recovery())
	if enablePprof {
		pprof.Register(router, "monitor/pprof")
	}
	apiV1Router := router.Group("/api/v1")
	v1.RegisterRouterAPIV1(apiV1Router, db)
	// use swagger middleware to serve the API docs
	router.GET("/doc/*any", swagger.WrapHandler(swaggerFiles.Handler))

	router.Run(fmt.Sprintf("%s:%d", config.Default.Server.Host, config.Default.Server.Port))
}

func setupDoc() {
	// programmatically set swagger info
	docs.SwaggerInfo.Title = "Go Example API"
	docs.SwaggerInfo.Description = "This is example golang server."
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = fmt.Sprintf("%s:%d", config.Default.Server.Host, config.Default.Server.Port)
	docs.SwaggerInfo.BasePath = "/api/v1"
	docs.SwaggerInfo.Schemes = []string{"http", "https"}
}

func initObservability(ctx context.Context) (close func(context.Context)) {
	closedFns := []func(context.Context){}

	// tracer initialization
	log.Info("Start trace provider")
	shutdown, err := internalTrace.InitTraceProvider(config.Default.Metadata.ServiceName, Version, config.Default.Otel.Trace)
	if err != nil {
		if err != internalTrace.ErrUndefindedTraceProto {
			log.Fatal(err.Error())
		}
		log.Info(err.Error())
	}

	if shutdown != nil {
		closedFns = append(closedFns, func(ctx context.Context) {
			if err := shutdown(ctx); err != nil {
				log.Fatal("failed to shutdown TracerProvider: " + err.Error())
			}
		})
	}

	return func(ctx context.Context) {
		for i := len(closedFns) - 1; i > 0; i-- {
			closedFns[i](ctx)
		}
	}
}
