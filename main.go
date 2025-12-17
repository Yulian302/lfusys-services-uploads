package main

import (
	"context"
	"log"
	"net/http"

	common "github.com/Yulian302/lfusys-services-commons"
	"github.com/Yulian302/lfusys-services-uploads/routers"
	"github.com/Yulian302/lfusys-services-uploads/uploads"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	cfg := common.LoadConfig()

	// verify aws credentials
	if cfg.AWS_ACCESS_KEY_ID == "" || cfg.AWS_SECRET_ACCESS_KEY == "" {
		log.Fatal("aws security credentials were not found")
	}

	awsCfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(cfg.AWS_REGION))
	if err != nil {
		log.Fatalf("failed to load aws config: %v", err)
	}

	s3Client := s3.NewFromConfig(awsCfg)

	r := gin.Default()

	r.GET("/test", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"message": "ok",
		})
	})

	uploadsHandler := uploads.NewUploadsHanlder(s3Client, &cfg)
	routers.RegisterUploadsRouter(uploadsHandler, r)

	r.Run(cfg.UPLOADS_ADDR)
}
