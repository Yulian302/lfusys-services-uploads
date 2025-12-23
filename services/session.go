package services

import (
	"context"
	"fmt"

	"github.com/Yulian302/lfusys-services-commons/errors"
	"github.com/Yulian302/lfusys-services-uploads/store"
)

type SessionService interface {
	MarkChunkComplete(ctx context.Context, uploadID string, chunkIdx uint32) error
}

type SessionServiceImpl struct {
	uploadsStore store.UploadsStore
}

func NewSessionServiceImpl(sessionStore store.UploadsStore) *SessionServiceImpl {
	return &SessionServiceImpl{
		uploadsStore: sessionStore,
	}
}

func (s *SessionServiceImpl) MarkChunkComplete(ctx context.Context, uploadID string, chunkIdx uint32) error {
	session, err := s.uploadsStore.GetSession(ctx, uploadID)
	if err != nil {
		return err
	}

	err = s.uploadsStore.PutChunk(ctx, uploadID, chunkIdx, session.TotalChunks)
	if err != nil {
		fmt.Println(err.Error())
		return fmt.Errorf("%w: %w", errors.ErrSessionUpdateDetails, err)
	}
	return nil
}
