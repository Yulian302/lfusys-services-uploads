package main

import (
	"context"
	"log"

	"github.com/Yulian302/lfusys-services-uploads/queues"
	"github.com/Yulian302/lfusys-services-uploads/services"
	"github.com/Yulian302/lfusys-services-uploads/store"
)

type Stores struct {
	chunks   store.ChunkStore
	sessions store.UploadsStore
}

type Services struct {
	Uploads       services.UploadService
	Sessions      services.SessionService
	UploadsNotify queues.UploadNotify

	Stores *Stores
}

type Shutdowner interface {
	Shutdown(context.Context) error
}

func BuildServices(app *App) *Services {

	upNotifyQueue := queues.NewSQSUploadNotify(app.Sqs, app.Config.ServiceConfig.UploadsNotificationsQueueName, app.Config.AWSConfig.AccountID)
	chunkStore := store.NewS3ChunkStore(app.S3, app.Config.AWSConfig.BucketName)
	sessionStore := store.NewDynamoDbUploadsStore(app.DynamoDB, app.Config.DynamoDBConfig.UploadsTableName)

	uploadService := services.NewUploadServiceImpl(chunkStore, app.Logger)
	sessionService := services.NewSessionServiceImpl(sessionStore, upNotifyQueue, app.Logger)

	return &Services{
		Uploads:       uploadService,
		Sessions:      sessionService,
		UploadsNotify: upNotifyQueue,

		Stores: &Stores{
			chunks:   chunkStore,
			sessions: sessionStore,
		},
	}
}

func (s *Services) Shutdown(ctx context.Context) error {
	log.Println("shutting down services")

	if s.Stores != nil {
		if err := s.Stores.Shutdown(ctx); err != nil {
			log.Printf("stores shutdown error: %v", err)
		}
	}

	log.Println("services shutdown complete")
	return nil
}

func (s *Stores) Shutdown(ctx context.Context) error {
	log.Println("shutting down stores")

	shutdownIfPossible := func(name string, v any) {
		if sh, ok := v.(Shutdowner); ok {
			if err := sh.Shutdown(ctx); err != nil {
				log.Printf("%s store shutdown error: %v", name, err)
			}
		}
	}

	shutdownIfPossible("chunks", s.chunks)
	shutdownIfPossible("sessions", s.sessions)

	log.Println("stores shutdown complete")
	return nil
}
