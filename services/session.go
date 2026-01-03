package services

import (
	"context"

	"github.com/Yulian302/lfusys-services-uploads/queues"
	"github.com/Yulian302/lfusys-services-uploads/store"
)

type SessionService interface {
	MarkChunkComplete(ctx context.Context, uploadID string, chunkIdx uint32) error
}

type SessionServiceImpl struct {
	uploadsStore store.UploadsStore
	uploadNotify queues.UploadNotify
}

func NewSessionServiceImpl(sessionStore store.UploadsStore, uploadNotify queues.UploadNotify) *SessionServiceImpl {
	return &SessionServiceImpl{
		uploadsStore: sessionStore,
		uploadNotify: uploadNotify,
	}
}

func (s *SessionServiceImpl) MarkChunkComplete(ctx context.Context, uploadID string, chunkIdx uint32) error {
	session, err := s.uploadsStore.GetSession(ctx, uploadID)
	if err != nil {
		return err
	}

	if err := s.uploadsStore.PutChunk(ctx, uploadID, chunkIdx, session.TotalChunks); err != nil {
		return err
	}

	completed, err := s.uploadsStore.TryFinalizeUpload(
		ctx,
		uploadID,
		session.TotalChunks,
	)
	if err != nil {
		return err
	}

	if completed {
		return s.uploadNotify.NotifyUploadComplete(ctx, uploadID)
	}

	return nil
}
