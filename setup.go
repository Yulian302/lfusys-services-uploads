package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	common "github.com/Yulian302/lfusys-services-commons"
	"github.com/Yulian302/lfusys-services-commons/config"
	logger "github.com/Yulian302/lfusys-services-commons/logging"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel/sdk/trace"
)

type App struct {
	Server *http.Server

	DynamoDB *dynamodb.Client
	S3       *s3.Client
	Sqs      *sqs.Client

	Config    config.Config
	AwsConfig aws.Config

	Services       *Services
	TracerProvider *trace.TracerProvider
	Logger         logger.Logger
}

func SetupApp() (*App, error) {
	cfg := config.LoadConfig()

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	if strings.EqualFold(cfg.Env, "PROD") {
		gin.SetMode(gin.ReleaseMode)
	}

	awsCfg, err := initAWS(*cfg.AWSConfig)
	if err != nil {
		return nil, err
	}

	db := initDynamo(awsCfg)
	if db == nil {
		return nil, errors.New("could not init dynamodb")
	}

	s3 := initS3(awsCfg)
	if s3 == nil {
		return nil, errors.New("could not init s3")
	}

	sqs := initSqs(awsCfg)
	if sqs == nil {
		return nil, errors.New("could not init sqs")
	}

	appLogger := logger.NewSlogLogger(logger.CreateAppLogger(cfg.Env))

	app := &App{
		DynamoDB: db,
		S3:       s3,
		Sqs:      sqs,

		Config:    cfg,
		AwsConfig: awsCfg,
		Logger:    appLogger,
	}

	if cfg.Tracing {
		tp, err := common.InitTracer(context.Background(), "uploads", cfg.TracingAddr)
		if err != nil {
			app.Logger.Error("tracing start failed", "err", err.Error())
			os.Exit(1)
		}
		app.Logger.Info("tracing in progress...")

		app.TracerProvider = tp
	}

	app.Services = BuildServices(app)

	return app, nil
}

func (a *App) Run(r *gin.Engine) error {
	a.Server = &http.Server{
		Addr:    a.Config.UploadsAddr,
		Handler: r,
	}

	return a.Server.ListenAndServe()
}

func initAWS(cfg config.AWSConfig) (aws.Config, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(
		context.Background(),
		awsconfig.WithRegion(cfg.Region),
	)
	if err != nil {
		return aws.Config{}, fmt.Errorf("load aws config: %w", err)
	}
	return awsCfg, nil
}

func initDynamo(cfg aws.Config) *dynamodb.Client {
	return dynamodb.NewFromConfig(cfg)
}

func initS3(cfg aws.Config) *s3.Client {
	return s3.NewFromConfig(cfg)
}

func initSqs(cfg aws.Config) *sqs.Client {
	return sqs.NewFromConfig(cfg)
}

func (a *App) Shutdown(ctx context.Context) error {
	a.Logger.Info("starting graceful shutdown")

	if a.Server != nil {
		if err := a.Server.Shutdown(ctx); err != nil {
			a.Logger.Error("http server shutdown failed", "err", err.Error())
		}
	}

	if a.Services != nil {
		if err := a.Services.Shutdown(ctx); err != nil {
			a.Logger.Error("services shutdown failed", "err", err.Error())
		}
	}

	if a.TracerProvider != nil {
		if err := a.TracerProvider.Shutdown(ctx); err != nil {
			a.Logger.Error("tracer shutdown failed", "err", err.Error())
		}
	}

	a.Logger.Info("graceful shutdown complete")
	return nil
}
