package util

import(
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/go-fund-transfer/internal/core"
)

func GetKafkaEnv() core.KafkaConfig {
	childLogger.Debug().Msg("GetKafkaEnv")

	err := godotenv.Load(".env")
	if err != nil {
		childLogger.Info().Err(err).Msg("env file not found !!!")
	}

	var kafkaConfig core.KafkaConfig
	var kafkaConfigurations core.KafkaConfigurations
	var topic core.Topic

	if os.Getenv("KAFKA_USER") !=  "" {
		kafkaConfigurations.Username = os.Getenv("KAFKA_USER")
	}
	if os.Getenv("KAFKA_PASSWORD") !=  "" {
		kafkaConfigurations.Password = os.Getenv("KAFKA_PASSWORD")
	}
	if os.Getenv("KAFKA_PROTOCOL") !=  "" {
		kafkaConfigurations.Protocol = os.Getenv("KAFKA_PROTOCOL")
	}
	if os.Getenv("KAFKA_MECHANISM") !=  "" {
		kafkaConfigurations.Mechanisms = os.Getenv("KAFKA_MECHANISM")
	}
	if os.Getenv("KAFKA_CLIENT_ID") !=  "" {
		kafkaConfigurations.Clientid = os.Getenv("KAFKA_CLIENT_ID")
	}
	if os.Getenv("KAFKA_BROKER_1") !=  "" {
		kafkaConfigurations.Brokers1 = os.Getenv("KAFKA_BROKER_1")
	}
	if os.Getenv("KAFKA_BROKER_2") !=  "" {
		kafkaConfigurations.Brokers2 = os.Getenv("KAFKA_BROKER_2")
	}
	if os.Getenv("KAFKA_BROKER_3") !=  "" {
		kafkaConfigurations.Brokers3 = os.Getenv("KAFKA_BROKER_3")
	}

	if os.Getenv("KAFKA_PARTITION") !=  "" {
		intVar, _ := strconv.Atoi(os.Getenv("KAFKA_PARTITION"))
		kafkaConfigurations.Partition = intVar
	}
	if os.Getenv("KAFKA_REPLICATION") !=  "" {
		intVar, _ := strconv.Atoi(os.Getenv("KAFKA_REPLICATION"))
		kafkaConfigurations.ReplicationFactor = intVar
	}

	if os.Getenv("TOPIC_CREDIT") !=  "" {
		topic.Credit = os.Getenv("TOPIC_CREDIT")
	}
	if os.Getenv("TOPIC_DEBIT") !=  "" {
		topic.Dedit = os.Getenv("TOPIC_DEBIT")
	}
	if os.Getenv("TOPIC_TRANSFER") !=  "" {
		topic.Transfer = os.Getenv("TOPIC_TRANSFER")
	}
	
	kafkaConfig.KafkaConfigurations = &kafkaConfigurations
	kafkaConfig.Topic = &topic

	return kafkaConfig
}