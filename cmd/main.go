package main

import(
	"time"
	"context"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/go-fund-transfer/internal/util"
	"github.com/go-fund-transfer/internal/adapter/event"
	"github.com/go-fund-transfer/internal/handler"
	"github.com/go-fund-transfer/internal/core"
	"github.com/go-fund-transfer/internal/service"
	"github.com/go-fund-transfer/internal/repository/postgre"
	"github.com/go-fund-transfer/internal/adapter/restapi"
	
)

var(
	logLevel 	= 	zerolog.DebugLevel
	appServer	core.AppServer
)

func init(){
	log.Debug().Msg("init")
	zerolog.SetGlobalLevel(logLevel)

	infoPod , server, restEndpoint := util.GetInfoPod()
	database := util.GetDatabaseEnv()
	configOTEL := util.GetOtelEnv()
	kafkaConfig := util.GetKafkaEnv()

	appServer.InfoPod = &infoPod
	appServer.Database = &database
	appServer.Server = &server
	appServer.RestEndpoint = &restEndpoint
	appServer.ConfigOTEL = &configOTEL
	appServer.KafkaConfig = &kafkaConfig
}

func main() {
	log.Debug().Msg("----------------------------------------------------")
	log.Debug().Msg("main")
	log.Debug().Msg("----------------------------------------------------")
	log.Debug().Interface("appServer :",appServer).Msg("")
	log.Debug().Msg("----------------------------------------------------")

	ctx, cancel := context.WithTimeout(context.Background(), 
									time.Duration( appServer.Server.ReadTimeout ) * time.Second)
	defer cancel()

	// Open Database
	count := 1
	var databaseHelper	postgre.DatabaseHelper
	var err error
	for {
		databaseHelper, err = postgre.NewDatabaseHelper(ctx, appServer.Database)
		if err != nil {
			if count < 3 {
				log.Error().Err(err).Msg("Erro open Database... trying again !!")
			} else {
				log.Error().Err(err).Msg("Fatal erro open Database aborting")
				panic(err)
			}
			time.Sleep(3 * time.Second)
			count = count + 1
			continue
		}
		break
	}
	
	repoDB := postgre.NewWorkerRepository(databaseHelper)
	restApiService	:= restapi.NewRestApiService()

	// Setup msk
	producerWorker, err := event.NewProducerWorker(appServer.KafkaConfig)
	if err != nil {
		log.Error().Err(err).Msg("Erro na abertura do Kafka")
	}

	workerService := service.NewWorkerService(	&repoDB, 
												appServer.RestEndpoint,
												restApiService, 
												producerWorker, 
												appServer.KafkaConfig.Topic)

	httpWorkerAdapter 	:= handler.NewHttpWorkerAdapter(workerService)
	httpServer 			:= handler.NewHttpAppServer(appServer.Server)

	httpServer.StartHttpAppServer(	ctx, 
									&httpWorkerAdapter,
									&appServer)
}