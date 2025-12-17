package routers

import (
	"github.com/Yulian302/lfusys-services-uploads/uploads"
	"github.com/gin-gonic/gin"
)

func RegisterUploadsRouter(h *uploads.UploadsHandler, r *gin.Engine) {
	uploads := r.Group("/upload")

	uploads.PUT("/:uploadId/chunk/:chunkId", h.Upload)
}
