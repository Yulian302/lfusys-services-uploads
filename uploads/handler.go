package uploads

import (
	"crypto/sha256"
	"encoding/hex"
	error "errors"
	"io"
	"net/http"
	"strconv"

	logger "github.com/Yulian302/lfusys-services-commons/logging"
	"github.com/Yulian302/lfusys-services-commons/errors"
	"github.com/Yulian302/lfusys-services-uploads/services"
	"github.com/gin-gonic/gin"
)

type UploadsHandler struct {
	uploadService  services.UploadService
	sessionService services.SessionService

	logger logger.Logger
}

func NewUploadsHandler(uploadService services.UploadService, sesionService services.SessionService, l logger.Logger) *UploadsHandler {
	return &UploadsHandler{
		uploadService:  uploadService,
		sessionService: sesionService,
		logger:         l,
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
		h.logger.Warn("upload chunk failed",
			"upload_id", uploadId,
			"chunk_id", chunkIdStr,
			"reason", "missing_fields",
		)
		errors.BadRequestResponse(c, "invalid fields")
		return
	}

	chunkId, err := strconv.ParseUint(chunkIdStr, 10, 32)
	if err != nil {
		h.logger.Warn("upload chunk failed",
			"upload_id", uploadId,
			"chunk_id", chunkIdStr,
			"reason", "invalid_chunk_id",
		)
		errors.BadRequestResponse(c, "invalid chunk ID")
		return
	}

	chunkData, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Warn("upload chunk failed",
			"upload_id", uploadId,
			"chunk_id", chunkId,
			"reason", "failed_to_read_body",
		)
		errors.BadRequestResponse(c, "failed to read chunk data")
		return
	}

	if len(chunkData) == 0 {
		h.logger.Warn("upload chunk failed",
			"upload_id", uploadId,
			"chunk_id", chunkId,
			"reason", "empty_chunk_data",
		)
		errors.BadRequestResponse(c, "no chunk binary data")
		return
	}

	// hash integrity check
	hash := sha256.New()
	hash.Write(chunkData)
	calculatedHash := hex.EncodeToString(hash.Sum(nil))
	if expectedHash != calculatedHash {
		h.logger.Warn("upload chunk failed",
			"upload_id", uploadId,
			"chunk_id", chunkId,
			"expected_hash", expectedHash,
			"calculated_hash", calculatedHash,
			"reason", "integrity_error",
		)
		errors.BadRequestResponse(c, "integrity error")
		return
	}

	err = h.uploadService.Upload(c.Request.Context(), uploadId, uint32(chunkId), chunkData)
	if err != nil {
		h.logger.Error("upload chunk failed",
			"upload_id", uploadId,
			"chunk_id", chunkId,
			"chunk_size", len(chunkData),
			"error", err,
		)
		errors.InternalServerErrorResponse(c, err.Error())
		return
	}

	err = h.sessionService.MarkChunkComplete(c.Request.Context(), uploadId, uint32(chunkId))
	if err != nil {
		if error.Is(err, errors.ErrSessionNotFound) {
			h.logger.Warn("mark chunk complete failed",
				"upload_id", uploadId,
				"chunk_id", chunkId,
				"reason", "session_not_found",
			)
			errors.UnauthorizedResponse(c, "session not found")
		} else if error.Is(err, errors.ErrSessionUpdateDetails) {
			h.logger.Error("mark chunk complete failed",
				"upload_id", uploadId,
				"chunk_id", chunkId,
				"reason", "session_update_error",
			)
			errors.InternalServerErrorResponse(c, "could not update session details")
		} else {
			h.logger.Error("mark chunk complete failed",
				"upload_id", uploadId,
				"chunk_id", chunkId,
				"error", err,
			)
			errors.InternalServerErrorResponse(c, "internal server error")
		}
		return
	}

	h.logger.Info("chunk uploaded successfully",
		"upload_id", uploadId,
		"chunk_id", chunkId,
		"chunk_size", len(chunkData),
	)

	c.JSON(http.StatusOK, UploadResponse{
		UploadId: uploadId,
		ChunkId:  uint32(chunkId),
	})
}
