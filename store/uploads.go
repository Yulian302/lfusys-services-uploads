package store

import (
	"context"
	cerr "errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Yulian302/lfusys-services-commons/errors"
	"github.com/Yulian302/lfusys-services-commons/health"
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

func (s *DynamoDbUploadsStore) IsReady() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err := s.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(s.tableName),
	})

	return err == nil
}

func (s *DynamoDbUploadsStore) Name() string {
	return "UploadsStore[sessions]"
}

func (s *DynamoDbUploadsStore) GetSession(ctx context.Context, uploadID string) (*UploadSession, error) {
	res, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"upload_id": &types.AttributeValueMemberS{Value: uploadID},
		},
	})
	if err != nil {
		fmt.Println(err.Error())
		return nil, fmt.Errorf("%w: %w", errors.ErrSessionNotFound, err)
	}

	var currentSession UploadSession
	if err := attributevalue.UnmarshalMap(res.Item, &currentSession); err != nil {
		return nil, fmt.Errorf("unmarshal failed: %w", err)
	}

	return &currentSession, nil
}

func (s *DynamoDbUploadsStore) PutChunk(ctx context.Context, uploadID string, chunkIdx uint32, totalChunks uint32) error {
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
			attribute_not_exists(uploaded_chunks)
			OR size(uploaded_chunks) <= :total
        `),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":chunk": &types.AttributeValueMemberNS{
				Value: []string{strconv.FormatUint(uint64(chunkIdx), 10)},
			},
			":total": &types.AttributeValueMemberN{
				Value: strconv.FormatUint(uint64(totalChunks), 10),
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
}

func (s *DynamoDbUploadsStore) TryFinalizeUpload(
	ctx context.Context,
	uploadID string,
	totalChunks uint32,
) (bool, error) {
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
			return false, nil // someone else finalized
		}
		return false, err
	}

	return true, nil
}
