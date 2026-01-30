package store

import (
	"bytes"
	"context"
	"fmt"

	"github.com/Yulian302/lfusys-services-commons/health"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type ChunkStore interface {
	PutChunk(ctx context.Context, key string, chunkData []byte) error

	health.ReadinessCheck
}

type S3ChunkStore struct {
	client     *s3.Client
	bucketName string
}

func NewS3ChunkStore(client *s3.Client, bucketName string) *S3ChunkStore {
	return &S3ChunkStore{
		client:     client,
		bucketName: bucketName,
	}
}

func (s *S3ChunkStore) IsReady(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucketName),
	})
	return err
}

func (s *S3ChunkStore) Name() string {
	return "S3[uploadChunks]"
}

func (store *S3ChunkStore) PutChunk(ctx context.Context, key string, chunkData []byte) error {
	_, err := store.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(store.bucketName),
		Key:    aws.String(key),
		Body:   bytes.NewReader(chunkData),
	})
	if err != nil {
		return fmt.Errorf("failed to upload chunk: %w", err)
	}
	return nil
}
