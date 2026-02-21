package store

import (
	"context"
	cerr "errors"
	"strconv"
	"time"

	apperror "github.com/Yulian302/lfusys-services-commons/errors"
	"github.com/Yulian302/lfusys-services-commons/health"
	"github.com/Yulian302/lfusys-services-commons/retries"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type UploadsStore interface {
	GetSession(ctx context.Context, uploadID string) (*UploadSession, error)
	PutChunk(ctx context.Context, uploadID string, chunkIdx uint32, totalChunks uint32) error
	TryFinalizeUpload(ctx context.Context, uploadID string, totalChunks uint32) (bool, error)

	health.ReadinessCheck
}

type DynamoDbUploadsStore struct {
	client    *dynamodb.Client
	tableName string
}

func NewDynamoDbUploadsStore(client *dynamodb.Client, tableName string) *DynamoDbUploadsStore {
	return &DynamoDbUploadsStore{
		client:    client,
		tableName: tableName,
	}
}

func (s *DynamoDbUploadsStore) IsReady(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	return retries.Retry(
		ctx,
		retries.HealthAttempts,
		retries.HealthBaseDelay,
		func() error {
			_, err := s.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
				TableName: aws.String(s.tableName),
			})

			return err
		},
		retries.IsRetriableDbError,
	)
}

func (s *DynamoDbUploadsStore) Name() string {
	return "UploadsStore[sessions]"
}

func (s *DynamoDbUploadsStore) GetSession(ctx context.Context, uploadID string) (*UploadSession, error) {
	var session UploadSession

	err := retries.Retry(
		ctx,
		retries.DefaultAttempts,
		retries.DefaultBaseDelay,
		func() error {
			out, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
				TableName: aws.String(s.tableName),
				Key: map[string]types.AttributeValue{
					"upload_id": &types.AttributeValueMemberS{Value: uploadID},
				},
			})
			if err != nil {
				return err
			}

			if out.Item == nil {
				return apperror.ErrSessionNotFound
			}

			return attributevalue.UnmarshalMap(out.Item, &session)
		},
		retries.IsRetriableDbError,
	)

	if err != nil {
		return nil, err
	}

	return &session, nil
}

func (s *DynamoDbUploadsStore) PutChunk(ctx context.Context, uploadID string, chunkIdx uint32, totalChunks uint32) error {
	return retries.Retry(
		ctx,
		retries.DefaultAttempts,
		retries.DefaultBaseDelay,
		func() error {
			_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
				TableName: aws.String(s.tableName),
				Key: map[string]types.AttributeValue{
					"upload_id": &types.AttributeValueMemberS{Value: uploadID},
				},
				UpdateExpression: aws.String(`
            ADD uploaded_chunks :chunk
            SET #status = :in_progress
        `),
				ConditionExpression: aws.String(`
			attribute_exists(upload_id)
        `),
				ExpressionAttributeValues: map[string]types.AttributeValue{
					":chunk": &types.AttributeValueMemberNS{
						Value: []string{strconv.FormatUint(uint64(chunkIdx), 10)},
					},
					":in_progress": &types.AttributeValueMemberS{Value: "in_progress"},
				},
				ExpressionAttributeNames: map[string]string{
					"#status": "status",
				},
				ReturnValues: types.ReturnValueAllNew,
			})

			var cfe *types.ConditionalCheckFailedException
			if err != nil && !cerr.As(err, &cfe) {
				return err
			}

			return nil
		},
		retries.IsRetriableDbError,
	)
}

func (s *DynamoDbUploadsStore) TryFinalizeUpload(
	ctx context.Context,
	uploadID string,
	totalChunks uint32,
) (bool, error) {
	finalized := false

	err := retries.Retry(
		ctx,
		retries.DefaultAttempts,
		retries.DefaultBaseDelay,
		func() error {
			_, err := s.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
				TableName: aws.String(s.tableName),
				Key: map[string]types.AttributeValue{
					"upload_id": &types.AttributeValueMemberS{Value: uploadID},
				},
				UpdateExpression: aws.String(`
			SET #status = :completed
		`),
				ConditionExpression: aws.String(`
			size(uploaded_chunks) = :total
			AND #status <> :completed
		`),
				ExpressionAttributeValues: map[string]types.AttributeValue{
					":total":     &types.AttributeValueMemberN{Value: strconv.FormatUint(uint64(totalChunks), 10)},
					":completed": &types.AttributeValueMemberS{Value: "completed"},
				},
				ExpressionAttributeNames: map[string]string{
					"#status": "status",
				},
			})

			if err != nil {
				var cfe *types.ConditionalCheckFailedException
				if cerr.As(err, &cfe) {
					finalized = false
					return nil // someone else finalized
				}
				return err
			}

			finalized = true
			return nil
		},
		retries.IsRetriableDbError,
	)

	return finalized, err
}
