package uploads

import (
	"bytes"
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

func (h *UploadsHandler) Upload(ctx *gin.Context) {
	uploadId := ctx.Param("uploadId")
	chunkIdStr := ctx.Param("chunkId")

	if uploadId == "" || chunkIdStr == "" {
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

	ctx.JSON(http.StatusOK, gin.H{
		"upload_id": uploadId,
		"chunk_id":  chunkId,
		"s3_key":    key,
	})
}
