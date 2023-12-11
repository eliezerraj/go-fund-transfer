package event

import (
	"time"
	"encoding/json"
	"math/rand"
	"context"
	"hash/fnv"

	"github.com/rs/zerolog/log"

	"github.com/confluentinc/confluent-kafka-go/kafka"
	"github.com/go-fund-transfer/internal/core"

	"github.com/aws/aws-xray-sdk-go/xray"

)

var childLogger = log.With().Str("event", "msk").Logger()

type ProducerWorker struct{
	configurations  *core.KafkaConfig
	producer        *kafka.Producer
}

func NewProducerWorker(configurations *core.KafkaConfig) ( *ProducerWorker, error) {
	childLogger.Debug().Msg("NewProducerWorker")

	kafkaBrokerUrls := 	configurations.KafkaConfigurations.Brokers1 + "," + configurations.KafkaConfigurations.Brokers2 + "," + configurations.KafkaConfigurations.Brokers3

	config := &kafka.ConfigMap{	"bootstrap.servers":            kafkaBrokerUrls,
								"security.protocol":            configurations.KafkaConfigurations.Protocol, //"SASL_SSL",
								"sasl.mechanisms":              configurations.KafkaConfigurations.Mechanisms, //"SCRAM-SHA-256",
								"sasl.username":                configurations.KafkaConfigurations.Username,
								"sasl.password":                configurations.KafkaConfigurations.Password,
								"acks": 						"all", // acks=0  acks=1 acks=all
								"message.timeout.ms":			5000,
								"retries":						5,
								"retry.backoff.ms":				500,
								"enable.idempotence":			true,                     
								}

	producer, err := kafka.NewProducer(config)
	if err != nil {
		childLogger.Error().Err(err).Msg("Failed to create producer:")
		return nil, err
	}

	return &ProducerWorker{ configurations : configurations,
							producer : producer,
	}, nil
}

//Hash key
func getPartition(key int, part *int) int32 {
	return int32(key%*part)
}

func hash(s string) int {
	h := fnv.New32a()
	h.Write([]byte(s))
	return int(h.Sum32())
}

func (p *ProducerWorker) Producer(ctx context.Context, event core.Event) error{
	childLogger.Debug().Msg("Producer")

	_, root := xray.BeginSubsegment(ctx, "Event.Producer")
	defer func() {
		root.Close(nil)
	}()

	rand.Seed(time.Now().UnixNano())
	min := 1
	max := 4
	salt := rand.Intn(max-min+1) + min

	payload, err := json.Marshal(event)
	if err != nil {
		childLogger.Error().Err(err).Msg("Erro no Marshall")
		return err
	}
	key	:= event.EventData.Transfer.AccountIDFrom

	childLogger.Debug().Msg("++++++++++++++++++++++++++++++++")
	childLogger.Debug().Interface("Topic ==>",event.EventType).Msg("")
	childLogger.Debug().Interface("Key   ==>",key).Msg("")
	childLogger.Debug().Interface("Event ==>",event.EventData).Msg("")
	//log.Debug().Interface("PartHash :",getPartitionHash(key, &p.configurations.KafkaConfigurations.Partition))).Msg(""))
	//log.Debug().Interface("Partition",getPartition(salt, &p.configurations.KafkaConfigurations.Partition))).Msg(""))
	childLogger.Debug().Msg("++++++++++++++++++++++++++++++++")

	producer := p.producer
	deliveryChan := make(chan kafka.Event)
	err = producer.Produce(	&kafka.Message	{
												TopicPartition: kafka.TopicPartition{	Topic: &event.EventType, 
																						Partition: getPartition(	salt,
																													&p.configurations.KafkaConfigurations.Partition,
																												), //kafka.PartitionAny,
																					}, 
												Value: 	[]byte(payload), 
												Headers:  []kafka.Header{{	Key: "key", 
																			Value: []byte(key),
																		}},
											},
							deliveryChan)
	if err != nil {
		childLogger.Error().Err(err).Msg("Failed to producer message")
		return err
	}
	e := <-deliveryChan
	m := e.(*kafka.Message)

	if m.TopicPartition.Error != nil {
		childLogger.Error().Err(m.TopicPartition.Error).Msg("Delivery failed")
	} else {		
		childLogger.Debug().Msg("Delivered message to topic")
		childLogger.Debug().Interface("topic  : ",*m.TopicPartition.Topic).Msg("")
		childLogger.Debug().Interface("partition  : ", m.TopicPartition.Partition).Msg("")
		childLogger.Debug().Interface("offset : ",m.TopicPartition.Offset).Msg("")
	}
	close(deliveryChan)

	return nil
}
