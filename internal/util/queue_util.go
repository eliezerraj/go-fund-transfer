package util

import(
	"os"

	"github.com/joho/godotenv"
	"github.com/go-fund-transfer/internal/core"
)

func GetQueueEnv() core.QueueConfig {
	childLogger.Debug().Msg("GetQueueEnv")

	err := godotenv.Load(".env")
	if err != nil {
		childLogger.Info().Err(err).Msg("env file not found !!!")
	}

	var queueConfig	core.QueueConfig

	if os.Getenv("QUEUE_URL_CREDIT") !=  "" {
		queueConfig.QueueUrl = os.Getenv("QUEUE_URL_CREDIT")
	}
	if os.Getenv("AWS_REGION") !=  "" {
		queueConfig.AwsRegion = os.Getenv("AWS_REGION")
	}

	return queueConfig
}