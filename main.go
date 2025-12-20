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
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/Yulian302/lfusys-services-uploads/docs"
	_ "github.com/joho/godotenv/autoload"
)

//	@title			LFU Sys UW
//	@version		1.0
//	@description	LFU Sys upload workers
//	@swagger		2.0

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

//	@host		localhost:8080
//	@BasePath	/

// @externalDocs.description	OpenAPI
// @externalDocs.url			https://swagger.io/resources/open-api/
func main() {
	cfg := common.LoadConfig()

	if err := cfg.AWSConfig.ValidateSecrets(); err != nil {
		log.Fatal("aws security credentials were not found")
	}

	awsCfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(cfg.AWSConfig.Region))
	if err != nil {
		log.Fatalf("failed to load aws config: %v", err)
	}

	s3Client := s3.NewFromConfig(awsCfg)

	r := gin.Default()

	r.Use(cors.New(
		cors.Config{
			AllowOrigins:     []string{"http://localhost:3000", "http://127.0.0.1:3000", "http://frontend:3000"},
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Chunk-Hash"},
			AllowCredentials: true,
		},
	))

	r.GET("/test", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"message": "ok",
		})
	})

	if cfg.Env != "PROD" {
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	}

	uploadsHandler := uploads.NewUploadsHanlder(s3Client, &cfg)
	routers.RegisterUploadsRouter(uploadsHandler, r)

	r.Run(cfg.ServiceConfig.UploadsAddr)
}
