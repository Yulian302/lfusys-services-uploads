package queues

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/Yulian302/lfusys-services-commons/health"
	logger "github.com/Yulian302/lfusys-services-commons/logging"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type UploadNotify interface {
	NotifyUploadComplete(ctx context.Context, uploadId string) error

	health.ReadinessCheck
}

type SQSUploadNotify struct {
	client    *sqs.Client
	queueName string
	accountID string

	logger logger.Logger
}

func NewSQSUploadNotify(client *sqs.Client, queueName string, accountId string) *SQSUploadNotify {
	return &SQSUploadNotify{
		client:    client,
		queueName: queueName,
		accountID: accountId,
	}
}

func (q *SQSUploadNotify) IsReady(ctx context.Context) error {
	_, err := q.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:     aws.String(fmt.Sprintf("https://sqs.%s.amazonaws.com/%s/%s.fifo", "eu-north-1", q.accountID, q.queueName)),
		MessageBody:  aws.String("healthcheck"),
		DelaySeconds: 0,
	})
	return err
}

func (q *SQSUploadNotify) Name() string {
	return "NoficationQueue[uploadsComplete]"
}

func (q *SQSUploadNotify) NotifyUploadComplete(ctx context.Context, uploadId string) error {
	messageBody := &UploadCompleteMessage{
		UploadId: uploadId,
	}
	messageBodyStr, err := json.Marshal(messageBody)
	if err != nil {
		return err
	}

	res, err := q.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(fmt.Sprintf("https://sqs.%s.amazonaws.com/%s/%s.fifo", "eu-north-1", q.accountID, q.queueName)),
		MessageBody: aws.String(string(messageBodyStr)),

		MessageGroupId:         aws.String(uploadId),
		MessageDeduplicationId: aws.String(fmt.Sprintf("dudup-%s", uploadId)),
	})
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	q.logger.Info("Message sent successfully. Message ID: %s", *res.MessageId)

	return nil
}
