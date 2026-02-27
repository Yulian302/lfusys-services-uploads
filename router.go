package main

import (
	"github.com/Yulian302/lfusys-services-commons/config"
	"github.com/Yulian302/lfusys-services-commons/health"
	logger "github.com/Yulian302/lfusys-services-commons/logging"
	"github.com/Yulian302/lfusys-services-commons/middleware"
	"github.com/Yulian302/lfusys-services-commons/responses"
	"github.com/Yulian302/lfusys-services-uploads/routers"
	"github.com/Yulian302/lfusys-services-uploads/uploads"
	"github.com/gin-gonic/gin"
)

func BuildRouter(app *App) *gin.Engine {
	r := gin.New()

	httpLogger := logger.CreateHttpLogger(app.Config.Env)
	middleware.ApplyLogging(r, httpLogger)

	middleware.ApplyCors(r, app.Config.Cors)

	if app.Config.Env != config.EnvProduction {
		middleware.ApplyTracing(r, "uploads-service")
		middleware.ApplySwagger(r)
	}

	registerRoutes(r, app)

	return r
}

func registerRoutes(r *gin.Engine, app *App) {
	r.GET("/test", func(ctx *gin.Context) {
		responses.JSONSuccess(ctx, "ok")
	})

	health.RegisterHealthRoutes(health.NewHealthHandler(
		app.Logger,
		app.Services.Stores.chunks,
		app.Services.UploadsNotify,
	),
		r,
	)

	v1 := routers.ApplyApiVersioning("1", r)

	routers.RegisterUploadsRouter(
		uploads.NewUploadsHandler(app.Services.Uploads, app.Services.UploadsNotify, app.Logger),
		v1,
	)
}
