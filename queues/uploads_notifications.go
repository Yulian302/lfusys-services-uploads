package queues

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Yulian302/lfusys-services-commons/health"
	logger "github.com/Yulian302/lfusys-services-commons/logging"
	"github.com/Yulian302/lfusys-services-commons/retries"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type UploadNotify interface {
	NotifyChunkComplete(ctx context.Context, uploadId string, chunkIdx uint32) error

	health.ReadinessCheck
}

type SQSUploadNotify struct {
	client    *sqs.Client
	queueName string
	accountID string
	region    string

	logger logger.Logger
}

func NewSQSUploadNotify(client *sqs.Client, queueName string, accountId string, region string, l logger.Logger) *SQSUploadNotify {
	return &SQSUploadNotify{
		client:    client,
		queueName: queueName,
		accountID: accountId,
		region:    region,
		logger:    l,
	}
}

func (q *SQSUploadNotify) IsReady(ctx context.Context) error {
	_, err := q.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:     aws.String(fmt.Sprintf("https://sqs.%s.amazonaws.com/%s/%s.fifo", q.region, q.accountID, q.queueName)),
		MessageBody:  aws.String("healthcheck"),
		DelaySeconds: 0,
	})
	return err
}

func (q *SQSUploadNotify) Name() string {
	return "NoficationQueue[uploadsComplete]"
}

func (q *SQSUploadNotify) NotifyChunkComplete(ctx context.Context, uploadId string, chunkIdx uint32) error {
	messageBody := &UploadCompleteMessage{
		UploadId: uploadId,
		ChunkIdx: chunkIdx,
	}
	messageBodyStr, err := json.Marshal(messageBody)
	if err != nil {
		q.logger.Error("upload notification failed", "reason", "bad message body")
		return err
	}

	var msgId string
	queueUrl := fmt.Sprintf("https://sqs.%s.amazonaws.com/%s/%s.fifo", q.region, q.accountID, q.queueName)

	err = retries.Retry(
		ctx,
		retries.DefaultAttempts,
		retries.DefaultBaseDelay,
		func() error {
			out, err := q.client.SendMessage(ctx, &sqs.SendMessageInput{
				QueueUrl:    aws.String(queueUrl),
				MessageBody: aws.String(string(messageBodyStr)),

				MessageGroupId:         aws.String(uploadId),
				MessageDeduplicationId: aws.String(fmt.Sprintf("dudup-%s-%d", uploadId, chunkIdx)),
			})
			if err != nil {
				return err
			}

			msgId = *out.MessageId
			return nil
		},
		retries.IsRetriableSQSError,
	)
	if err != nil {
		q.logger.Error("upload notification sending failed", "err", err)
		return err
	}

	q.logger.Info(fmt.Sprintf("Chunk %d complete message sent successfully. Message ID: %s", chunkIdx, msgId))

	return nil
}
