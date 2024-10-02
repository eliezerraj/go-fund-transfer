package sqs

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/rs/zerolog/log"
	"github.com/go-fund-transfer/internal/core"
	"github.com/go-fund-transfer/internal/lib"
	"github.com/go-fund-transfer/internal/config/config_aws"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

var childLogger = log.With().Str("event", "sqs").Logger()

type NotifierSQS struct{
	sqsClient 	*sqs.Client
	queueConfig	*core.QueueConfig
} 

func NewNotifierSQS(ctx context.Context, queueConfig *core.QueueConfig) (*NotifierSQS, error){
	childLogger.Debug().Msg("NewNotifierSQS")

	span := lib.Span(ctx, "event.NewNotifierSQS")	
    defer span.End()

	sdkConfig, err :=config_aws.GetAWSConfig(ctx, queueConfig.AwsRegion)
	if err != nil{
		return nil, err
	}

	sqsClient := sqs.NewFromConfig(*sdkConfig)
	notifierSQS := NotifierSQS{
		sqsClient: sqsClient,
		queueConfig: queueConfig,
	}

	return &notifierSQS, nil
} 

func (s *NotifierSQS) Producer(ctx context.Context, event core.Event) error {
	childLogger.Debug().Msg("ProducerEvent")

	span := lib.Span(ctx, "event.producer-sqs")	
    defer span.End()

	messageJson, err := json.Marshal(event)
	if err != nil {
		childLogger.Error().Err(err).Msg("error marshal json")
		return err
	}

	input := &sqs.SendMessageInput{
		QueueUrl:    aws.String(s.queueConfig.QueueUrl),
		MessageBody: aws.String(string(messageJson)),
		MessageGroupId:         aws.String(event.EventData.Transfer.AccountIDTo), 
        MessageDeduplicationId: aws.String(strconv.Itoa(event.EventData.Transfer.ID)),
	}

	_, err = s.sqsClient.SendMessage(ctx, input)
	if err != nil {
		childLogger.Error().Err(err).Msg("error sqsClient.SendMessage")
		return err
	}

	return nil
}