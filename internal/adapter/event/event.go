package event

import (
	"context"

	go_core_observ "github.com/eliezerraj/go-core/observability"
	go_core_event "github.com/eliezerraj/go-core/event/kafka"

	"github.com/rs/zerolog/log"
)

var childLogger = log.With().Str("adapter", "event").Logger()

var tracerProvider go_core_observ.TracerProvider
var producerWorker go_core_event.ProducerWorker

type WorkerEvent struct {
	Topics	[]string
	WorkerKafka *go_core_event.ProducerWorker 
}

// About create a worker producer kafka
func NewWorkerEvent(ctx context.Context, topics []string, kafkaConfigurations *go_core_event.KafkaConfigurations) (*WorkerEvent, error) {
	childLogger.Info().Msg("NewWorkerEvent")

	//trace
	span := tracerProvider.Span(ctx, "adapter.event.NewWorkerEvent")
	defer span.End()

	workerKafka, err := producerWorker.NewProducerWorker(kafkaConfigurations)
	if err != nil {
		return nil, err
	}

	return &WorkerEvent{
		Topics: topics,
		WorkerKafka: workerKafka,
	},nil
}

// About create a worker producer kafka with transaction
func NewWorkerEventTX(ctx context.Context, topics []string, kafkaConfigurations *go_core_event.KafkaConfigurations) (*WorkerEvent, error) {
	childLogger.Info().Msg("NewWorkerEventTX")

	//trace
	span := tracerProvider.Span(ctx, "adapter.event.NewWorkerEventTX")
	defer span.End()

	workerKafka, err := producerWorker.NewProducerWorkerTX(kafkaConfigurations)
	if err != nil {
		return nil, err
	}

	// Start Kafka InitTransactions
	err = workerKafka.InitTransactions(ctx)
	if err != nil {
		childLogger.Error().Err(err).Msg("failed to kafka InitTransactions")
		return nil, err
	}	

	return &WorkerEvent{
		Topics: topics,
		WorkerKafka: workerKafka,
	},nil
}