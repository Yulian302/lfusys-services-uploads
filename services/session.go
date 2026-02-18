package services

import (
	"context"

	logger "github.com/Yulian302/lfusys-services-commons/logging"
	"github.com/Yulian302/lfusys-services-uploads/queues"
	"github.com/Yulian302/lfusys-services-uploads/store"
)

type SessionService interface {
	MarkChunkComplete(ctx context.Context, uploadID string, chunkIdx uint32) error
}

type SessionServiceImpl struct {
	uploadsStore store.UploadsStore
	uploadNotify queues.UploadNotify

	logger logger.Logger
}

func NewSessionServiceImpl(sessionStore store.UploadsStore, uploadNotify queues.UploadNotify, l logger.Logger) *SessionServiceImpl {
	return &SessionServiceImpl{
		uploadsStore: sessionStore,
		uploadNotify: uploadNotify,
		logger:       l,
	}
}

func (s *SessionServiceImpl) MarkChunkComplete(ctx context.Context, uploadID string, chunkIdx uint32) error {
	session, err := s.uploadsStore.GetSession(ctx, uploadID)
	if err != nil {
		s.logger.Error("failed to get upload session",
			"upload_id", uploadID,
			"chunk_idx", chunkIdx,
			"error", err,
		)
		return err
	}

	if err := s.uploadsStore.PutChunk(ctx, uploadID, chunkIdx, session.TotalChunks); err != nil {
		s.logger.Error("failed to mark chunk complete",
			"upload_id", uploadID,
			"chunk_idx", chunkIdx,
			"total_chunks", session.TotalChunks,
			"error", err,
		)
		return err
	}

	completed, err := s.uploadsStore.TryFinalizeUpload(
		ctx,
		uploadID,
		session.TotalChunks,
	)
	if err != nil {
		s.logger.Error("failed to finalize upload",
			"upload_id", uploadID,
			"error", err,
		)
		return err
	}

	if completed {
		s.logger.Info("upload finalized, notifying",
			"upload_id", uploadID,
		)
		return s.uploadNotify.NotifyUploadComplete(ctx, uploadID)
	}

	s.logger.Debug("chunk marked complete",
		"upload_id", uploadID,
		"chunk_idx", chunkIdx,
		"progress", float64(chunkIdx+1)/float64(session.TotalChunks),
	)
	return nil
}
