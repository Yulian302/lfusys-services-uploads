package services

import (
	"context"
	"fmt"

	"github.com/Yulian302/lfusys-services-uploads/store"
)

type UploadService interface {
	Upload(ctx context.Context, uploadID string, chunkID uint32, chunkData []byte) error
}

type UploadServiceImpl struct {
	chunkStore store.ChunkStore
}

func NewUploadServiceImpl(chunkStore store.ChunkStore) *UploadServiceImpl {
	return &UploadServiceImpl{
		chunkStore: chunkStore,
	}
}

func (s *UploadServiceImpl) Upload(ctx context.Context, uploadID string, chunkID uint32, chunkData []byte) error {
	key := fmt.Sprintf("uploads/%s/chunk_%d", uploadID, chunkID)
	if err := s.chunkStore.PutChunk(ctx, key, chunkData); err != nil {
		return err
	}

	return nil
}
