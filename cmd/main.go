package main

import(
	"time"
	"context"
	
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/go-fund-transfer/internal/infra/configuration"
	"github.com/go-fund-transfer/internal/core/model"
	"github.com/go-fund-transfer/internal/core/service"
	"github.com/go-fund-transfer/internal/infra/server"
	"github.com/go-fund-transfer/internal/adapter/api"
	"github.com/go-fund-transfer/internal/adapter/database"
	"github.com/go-fund-transfer/internal/adapter/event"
	go_core_pg "github.com/eliezerraj/go-core/database/pg"  
)

var(
	logLevel = 	zerolog.DebugLevel
	appServer	model.AppServer
	databaseConfig go_core_pg.DatabaseConfig
	databasePGServer go_core_pg.DatabasePGServer
)

func init(){
	log.Debug().Msg("init")
	zerolog.SetGlobalLevel(logLevel)

	infoPod, server := configuration.GetInfoPod()
	configOTEL 		:= configuration.GetOtelEnv()
	databaseConfig 	:= configuration.GetDatabaseEnv()
	apiService 	:= configuration.GetEndpointEnv() 
	kafkaConfigurations, topics := configuration.GetKafkaEnv() 

	appServer.InfoPod = &infoPod
	appServer.Server = &server
	appServer.ConfigOTEL = &configOTEL
	appServer.DatabaseConfig = &databaseConfig
	appServer.ApiService = apiService
	appServer.KafkaConfigurations = &kafkaConfigurations
	appServer.Topics = topics
}

func main (){
	log.Debug().Msg("----------------------------------------------------")
	log.Debug().Msg("main")
	log.Debug().Msg("----------------------------------------------------")
	log.Debug().Interface("appServer :",appServer).Msg("")
	log.Debug().Msg("----------------------------------------------------")

	ctx, cancel := context.WithTimeout(	context.Background(), 
										time.Duration( appServer.Server.ReadTimeout ) * time.Second)
	defer cancel()

	// Open Database
	count := 1
	var err error
	for {
		log.Debug().Interface("===== > databaseConfig :",appServer.DatabaseConfig).Msg("")
		databasePGServer, err = databasePGServer.NewDatabasePGServer(ctx, *appServer.DatabaseConfig)
		if err != nil {
			if count < 3 {
				log.Error().Err(err).Msg("error open database... trying again !!")
			} else {
				log.Error().Err(err).Msg("fatal error open Database aborting")
				panic(err)
			}
			time.Sleep(3 * time.Second) //backoff
			count = count + 1
			continue
		}
		break
	}

	// Database
	database := database.NewWorkerRepository(&databasePGServer)

	// Kafka
	workerEvent, err := event.NewWorkerEventTX(ctx, appServer.Topics, appServer.KafkaConfigurations)
	if err != nil {
		log.Error().Err(err).Msg("error open kafka")
		panic(err)
	}

	workerService := service.NewWorkerService(database, appServer.ApiService, workerEvent)
	httpRouters := api.NewHttpRouters(workerService)
	httpServer := server.NewHttpAppServer(appServer.Server)
	httpServer.StartHttpAppServer(ctx, &httpRouters, &appServer)
}