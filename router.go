package main

import (
	"strings"

	"github.com/Yulian302/lfusys-services-commons/health"
	"github.com/Yulian302/lfusys-services-commons/responses"
	"github.com/Yulian302/lfusys-services-uploads/routers"
	"github.com/Yulian302/lfusys-services-uploads/uploads"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func BuildRouter(app *App) *gin.Engine {
	r := gin.New()

	applyCors(r, app)
	applyTracing(r, app)
	applySwagger(r, app)

	registerRoutes(r, app)

	return r
}

func applyCors(r *gin.Engine, app *App) {
	origins := strings.Split(app.Config.CorsConfig.Origins, ",")
	r.Use(cors.New(
		cors.Config{
			AllowOrigins:     origins,
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Chunk-Hash"},
			AllowCredentials: true,
		},
	))
}

func applyTracing(r *gin.Engine, app *App) {
	if !app.Config.Tracing {
		return
	}

	r.Use(otelgin.Middleware("uploads-service"))
}

func applySwagger(r *gin.Engine, app *App) {
	if app.Config.Env == "PROD" {
		return
	}
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

func registerRoutes(r *gin.Engine, app *App) {
	r.GET("/test", func(ctx *gin.Context) {
		responses.JSONSuccess(ctx, "ok")
	})

	health.RegisterHealthRoutes(health.NewHealthHandler(
		app.Services.Stores.sessions,
		app.Services.Stores.chunks,
		app.Services.UploadsNotify,
	),
		r,
	)

	v1 := routers.ApplyApiVersioning("1", r)

	routers.RegisterUploadsRouter(
		uploads.NewUploadsHandler(app.Services.Uploads, app.Services.Sessions, app.Logger),
		v1,
	)
}
