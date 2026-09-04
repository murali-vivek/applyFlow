package queue

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/google/uuid"
)

type Message struct {
	OutreachID uuid.UUID `json:"outreachId"`
}

type SQSClient struct {
	client   *sqs.Client
	queueURL string
}

func NewSQSClient(ctx context.Context, region, endpoint, queueURL string) (*SQSClient, error) {
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(region),
	}
	if endpoint != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider("test", "test", ""),
		))
	}

	cfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := sqs.NewFromConfig(cfg, func(o *sqs.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
	})

	return &SQSClient{client: client, queueURL: queueURL}, nil
}

func (q *SQSClient) EnsureQueue(ctx context.Context) error {
	_, err := q.client.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: queueNameFromURL(q.queueURL),
	})
	return err
}

func queueNameFromURL(url string) *string {
	// http://localhost:4566/000000000000/applyflow-email-queue -> applyflow-email-queue
	for i := len(url) - 1; i >= 0; i-- {
		if url[i] == '/' {
			name := url[i+1:]
			return &name
		}
	}
	return &url
}

func (q *SQSClient) CreateQueueIfNeeded(ctx context.Context, queueName string) (string, error) {
	out, err := q.client.CreateQueue(ctx, &sqs.CreateQueueInput{
		QueueName: aws.String(queueName),
	})
	if err != nil {
		return "", err
	}
	return *out.QueueUrl, nil
}

func (q *SQSClient) Send(ctx context.Context, outreachID uuid.UUID) error {
	body, err := json.Marshal(Message{OutreachID: outreachID})
	if err != nil {
		return err
	}
	_, err = q.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(q.queueURL),
		MessageBody: aws.String(string(body)),
	})
	return err
}

type ReceivedMessage struct {
	ReceiptHandle string
	OutreachID    uuid.UUID
}

func (q *SQSClient) Receive(ctx context.Context) ([]ReceivedMessage, error) {
	out, err := q.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(q.queueURL),
		MaxNumberOfMessages: 10,
		WaitTimeSeconds:     20,
		VisibilityTimeout:   120,
	})
	if err != nil {
		return nil, err
	}

	var msgs []ReceivedMessage
	for _, m := range out.Messages {
		var payload Message
		if err := json.Unmarshal([]byte(*m.Body), &payload); err != nil {
			continue
		}
		msgs = append(msgs, ReceivedMessage{
			ReceiptHandle: *m.ReceiptHandle,
			OutreachID:    payload.OutreachID,
		})
	}
	return msgs, nil
}

func (q *SQSClient) Delete(ctx context.Context, receiptHandle string) error {
	_, err := q.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(q.queueURL),
		ReceiptHandle: aws.String(receiptHandle),
	})
	return err
}
