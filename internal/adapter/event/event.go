package event

import (

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

func NewWorkerEvent(topics []string, kafkaConfigurations *go_core_event.KafkaConfigurations) (*WorkerEvent, error) {
	childLogger.Debug().Msg("workerEvent")

	workerKafka, err := producerWorker.NewProducerWorker(kafkaConfigurations)
	if err != nil {
		return nil, err
	}

	return &WorkerEvent{
		Topics: topics,
		WorkerKafka: workerKafka,
	},nil
}