package uploads

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	common "github.com/Yulian302/lfusys-services-commons"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
)

type UploadsHandler struct {
	s3Client *s3.Client
	config   *common.Config
}

func NewUploadsHanlder(s3Client *s3.Client, config *common.Config) *UploadsHandler {
	return &UploadsHandler{
		s3Client: s3Client,
		config:   config,
	}
}

type HTTPError struct {
	Error string `json:"error" example:"error message"`
}

type UploadResponse struct {
	UploadId string `json:"upload_id" example:"abc123"`
	ChunkId  int    `json:"chunk_id" example:"1"`
	S3Key    string `json:"s3_key" example:"uploads/abc123/chunk_1"`
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
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid fields",
		})
		return
	}

	chunkId, err := strconv.Atoi(chunkIdStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid chunk ID",
		})
		return
	}

	chunkData, err := io.ReadAll(ctx.Request.Body)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "failed to read chunk"})
		return
	}
	if len(chunkData) == 0 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "no binary data"})
		return
	}

	// hash integrity check
	hash := sha256.New()
	hash.Write(chunkData)
	calculatedHash := hex.EncodeToString(hash.Sum(nil))
	if expectedHash != calculatedHash {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "integrity error"})
		return
	}

	key := fmt.Sprintf("uploads/%s/chunk_%d", uploadId, chunkId)
	_, err = h.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(h.config.AWS_BUCKET_NAME),
		Key:    aws.String(key),
		Body:   bytes.NewReader(chunkData),
	})
	if err != nil {
		log.Printf("S3 error: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload chunk"})
		return
	}

	ctx.JSON(http.StatusOK, UploadResponse{
		UploadId: uploadId,
		ChunkId:  chunkId,
		S3Key:    key,
	})
}
