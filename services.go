package main

import (
	"context"
	"fmt"

	logger "github.com/Yulian302/lfusys-services-commons/logging"
	"github.com/Yulian302/lfusys-services-uploads/queues"
	"github.com/Yulian302/lfusys-services-uploads/services"
	"github.com/Yulian302/lfusys-services-uploads/store"
)

type Stores struct {
	chunks store.ChunkStore

	logger logger.Logger
}

type Services struct {
	Uploads       services.UploadService
	UploadsNotify queues.UploadNotify

	Stores *Stores
	logger logger.Logger
}

type Shutdowner interface {
	Shutdown(context.Context) error
}

func BuildServices(app *App) *Services {
	upNotifyQueue := queues.NewSQSUploadNotify(app.Sqs, app.Config.ServiceConfig.UploadsNotificationsQueueName, app.Config.AWSConfig.AccountID, app.Config.AWSConfig.Region, app.Logger)
	chunkStore := store.NewS3ChunkStore(app.S3, app.Config.AWSConfig.BucketName)

	uploadService := services.NewUploadServiceImpl(chunkStore, app.Logger)

	app.Logger.Info("uploads services initialized successfully")

	return &Services{
		Uploads:       uploadService,
		UploadsNotify: upNotifyQueue,

		Stores: &Stores{
			chunks: chunkStore,
		},
	}
}

func (s *Services) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down services")

	if s.Stores != nil {
		if err := s.Stores.Shutdown(ctx); err != nil {
			s.logger.Error("stores shutdown failed", "err", err.Error())
		}
	}

	s.logger.Info("services shutdown complete")
	return nil
}

func (s *Stores) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down stores")

	shutdownIfPossible := func(name string, v any) {
		if sh, ok := v.(Shutdowner); ok {
			if err := sh.Shutdown(ctx); err != nil {
				s.logger.Error(fmt.Sprintf("%s store shutdown failed", name), "err", err.Error())
			}
		}
	}

	shutdownIfPossible("chunks", s.chunks)

	s.logger.Info("stores shutdown complete")
	return nil
}
