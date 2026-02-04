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
func (h *UploadsHandler) Upload(c *gin.Context) {
	uploadId := c.Param("uploadId")
	chunkIdStr := c.Param("chunkId")
	expectedHash := c.GetHeader("X-Chunk-Hash")

	if uploadId == "" || chunkIdStr == "" || expectedHash == "" {
		errors.BadRequestResponse(c, "invalid fields")
		return
	}

	chunkId, err := strconv.ParseUint(chunkIdStr, 10, 32)
	if err != nil {
		errors.BadRequestResponse(c, "invalid chunk ID")
		return
	}

	chunkData, err := io.ReadAll(c.Request.Body)
	if err != nil {
		errors.BadRequestResponse(c, "failed to read chunk data")
		return
	}

	if len(chunkData) == 0 {
		errors.BadRequestResponse(c, "no chunk binary data")
		return
	}

	// hash integrity check
	hash := sha256.New()
	hash.Write(chunkData)
	calculatedHash := hex.EncodeToString(hash.Sum(nil))
	if expectedHash != calculatedHash {
		errors.BadRequestResponse(c, "integrity error")
		return
	}

	err = h.uploadService.Upload(c.Request.Context(), uploadId, uint32(chunkId), chunkData)
	if err != nil {
		errors.InternalServerErrorResponse(c, err.Error())
		return
	}

	err = h.sessionService.MarkChunkComplete(c.Request.Context(), uploadId, uint32(chunkId))
	if err != nil {
		if error.Is(err, errors.ErrSessionNotFound) {
			errors.UnauthorizedResponse(c, "session not found")
		} else if error.Is(err, errors.ErrSessionUpdateDetails) {
			errors.InternalServerErrorResponse(c, "could not update session details")
		} else {
			errors.InternalServerErrorResponse(c, "internal server error")
		}
		return
	}

	c.JSON(http.StatusOK, UploadResponse{
		UploadId: uploadId,
		ChunkId:  uint32(chunkId),
	})
}
