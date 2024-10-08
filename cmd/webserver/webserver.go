package webserver

import(
	"time"
	"context"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/go-fund-transfer/internal/util"
	"github.com/go-fund-transfer/internal/adapter/event/kafka"
	"github.com/go-fund-transfer/internal/adapter/event/sqs"
	"github.com/go-fund-transfer/internal/adapter/event"
	"github.com/go-fund-transfer/internal/handler"
	"github.com/go-fund-transfer/internal/core"
	"github.com/go-fund-transfer/internal/service"
	"github.com/go-fund-transfer/internal/repository/pg"
	"github.com/go-fund-transfer/internal/handler/controller"
	"github.com/go-fund-transfer/internal/repository/storage"
	"github.com/go-fund-transfer/internal/adapter/restapi"
)

var(
	logLevel = zerolog.DebugLevel
	appServer	core.AppServer
	producerWorker	event.EventNotifier
)

func init(){
	log.Debug().Msg("init")
	zerolog.SetGlobalLevel(logLevel)

	infoPod , server, restEndpoint := util.GetInfoPod()
	database := util.GetDatabaseEnv()
	configOTEL := util.GetOtelEnv()
	kafkaConfig := util.GetKafkaEnv()
	queueConfig := util.GetQueueEnv()

	appServer.InfoPod = &infoPod
	appServer.Database = &database
	appServer.Server = &server
	appServer.RestEndpoint = &restEndpoint
	appServer.ConfigOTEL = &configOTEL
	appServer.KafkaConfig = &kafkaConfig
	appServer.QueueConfig = &queueConfig
}

func Server() {
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
	var databasePG	pg.DatabasePG
	var err error
	for {
		databasePG, err = pg.NewDatabasePGServer(ctx, appServer.Database)
		if err != nil {
			if count < 3 {
				log.Error().Err(err).Msg("error open Database... trying again !!")
			} else {
				log.Error().Err(err).Msg("fatal erro open Database aborting")
				panic(err)
			}
			time.Sleep(3 * time.Second)
			count = count + 1
			continue
		}
		break
	}
	
	repoDatabase := storage.NewWorkerRepository(databasePG)

	// Setup queue type
	if (appServer.InfoPod.QueueType == "kafka") {
		producerWorker, err = kafka.NewProducerWorker(appServer.KafkaConfig)
	} else {
		producerWorker, err = sqs.NewNotifierSQS(ctx, appServer.QueueConfig)
	}
	if err != nil {
		log.Error().Err(err).Msg("erro connect to queue")
	}

	restApiService	:= restapi.NewRestApiService(&appServer)
	workerService := service.NewWorkerService(	&repoDatabase, 
												&appServer,
												restApiService, 
												producerWorker, 
												appServer.KafkaConfig.Topic)

	httpWorkerAdapter 	:= controller.NewHttpWorkerAdapter(workerService)
	httpServer 			:= handler.NewHttpAppServer(appServer.Server)

	httpServer.StartHttpAppServer(ctx,&httpWorkerAdapter,&appServer)
}