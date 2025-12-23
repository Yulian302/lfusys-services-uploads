package uploads

import (
	"crypto/sha256"
	"encoding/hex"
	error "errors"
	"io"
	"net/http"
	"strconv"

	"github.com/Yulian302/lfusys-services-commons/errors"
	"github.com/Yulian302/lfusys-services-uploads/services"
	"github.com/gin-gonic/gin"
)

type UploadsHandler struct {
	uploadService  services.UploadService
	sessionService services.SessionService
}

func NewUploadsHandler(uploadService services.UploadService, sesionService services.SessionService) *UploadsHandler {
	return &UploadsHandler{
		uploadService:  uploadService,
		sessionService: sesionService,
	}
}

type HTTPError struct {
	Error string `json:"error" example:"error message"`
}

// Upload godoc
//
//	@Summary		Upload file chunk
//	@Description	Upload a file chunk with integrity verification
//	@Tags			uploads
//	@Accept			octet-stream
//	@Produce		json
//	@Param			uploadId		path		string			true	"Upload session ID"
//	@Param			chunkId			path		int				true	"Chunk number"
//	@Param			X-Chunk-Hash	header		string			true	"SHA256 hash of chunk data"
//	@Success		200				{object}	UploadResponse	"Chunk uploaded successfully"
//	@Failure		400				{object}	HTTPError		"Invalid request or integrity error"
//	@Failure		500				{object}	HTTPError		"S3 upload failed"
//	@Router			/upload/{uploadId}/chunk/{chunkId} [put]
func (h *UploadsHandler) Upload(ctx *gin.Context) {
	uploadId := ctx.Param("uploadId")
	chunkIdStr := ctx.Param("chunkId")
	expectedHash := ctx.GetHeader("X-Chunk-Hash")

	if uploadId == "" || chunkIdStr == "" || expectedHash == "" {
		errors.BadRequestResponse(ctx, "invalid fields")
		return
	}

	chunkId, err := strconv.ParseUint(chunkIdStr, 10, 32)
	if err != nil {
		errors.BadRequestResponse(ctx, "invalid chunk ID")
		return
	}

	chunkData, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		errors.BadRequestResponse(ctx, "failed to read chunk data")
		return
	}

	if len(chunkData) == 0 {
		errors.BadRequestResponse(ctx, "no chunk binary data")
		return
	}

	// hash integrity check
	hash := sha256.New()
	hash.Write(chunkData)
	calculatedHash := hex.EncodeToString(hash.Sum(nil))
	if expectedHash != calculatedHash {
		errors.BadRequestResponse(ctx, "integrity error")
		return
	}

	err = h.uploadService.Upload(ctx, uploadId, uint32(chunkId), chunkData)
	if err != nil {
		errors.InternalServerErrorResponse(ctx, err.Error())
		return
	}

	err = h.sessionService.MarkChunkComplete(ctx, uploadId, uint32(chunkId))
	if err != nil {
		if error.Is(err, errors.ErrSessionNotFound) {
			errors.UnauthorizedResponse(ctx, "session not found")
		} else if error.Is(err, errors.ErrSessionUpdateDetails) {
			errors.InternalServerErrorResponse(ctx, "could not update session details")
		} else {
			errors.InternalServerErrorResponse(ctx, "internal server error")
		}
		return
	}

	ctx.JSON(http.StatusOK, UploadResponse{
		UploadId: uploadId,
		ChunkId:  uint32(chunkId),
	})
}
