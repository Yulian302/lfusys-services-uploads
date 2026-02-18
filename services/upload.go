package services

import (
	"context"
	"fmt"

	logger "github.com/Yulian302/lfusys-services-commons/logging"
	"github.com/Yulian302/lfusys-services-uploads/store"
)

type UploadService interface {
	Upload(ctx context.Context, uploadID string, chunkID uint32, chunkData []byte) error
}

type UploadServiceImpl struct {
	chunkStore store.ChunkStore

	logger logger.Logger
}

func NewUploadServiceImpl(chunkStore store.ChunkStore, l logger.Logger) *UploadServiceImpl {
	return &UploadServiceImpl{
		chunkStore: chunkStore,
		logger:     l,
	}
}

func (s *UploadServiceImpl) Upload(ctx context.Context, uploadID string, chunkID uint32, chunkData []byte) error {
	key := fmt.Sprintf("uploads/%s/chunk_%d", uploadID, chunkID)
	if err := s.chunkStore.PutChunk(ctx, key, chunkData); err != nil {
		s.logger.Error("failed to upload chunk",
			"upload_id", uploadID,
			"chunk_id", chunkID,
			"chunk_size", len(chunkData),
			"error", err,
		)
		return err
	}

	s.logger.Debug("chunk uploaded successfully",
		"upload_id", uploadID,
		"chunk_id", chunkID,
		"chunk_size", len(chunkData),
	)
	return nil
}
