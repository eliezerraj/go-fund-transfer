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
	logLevel = 	zerolog.InfoLevel // zerolog.InfoLevel zerolog.DebugLevel
	appServer	model.AppServer
	databaseConfig go_core_pg.DatabaseConfig
	databasePGServer go_core_pg.DatabasePGServer
	childLogger = log.With().Str("component","go-fund-transfer").Str("package", "main").Logger()
)

// About initialize the enviroment var
func init(){
	childLogger.Info().Str("func","init").Send()

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

// About main
func main (){
	childLogger.Info().Str("func","main").Interface("appServer :",appServer).Send()

	ctx, cancel := context.WithTimeout(	context.Background(), 
										time.Duration( appServer.Server.ReadTimeout ) * time.Second)
	defer cancel()

	// Open Database
	count := 1
	var err error
	for {
		databasePGServer, err = databasePGServer.NewDatabasePGServer(ctx, *appServer.DatabaseConfig)
		if err != nil {
			if count < 3 {
				childLogger.Error().Err(err).Msg("error open database... trying again !!")
			} else {
				childLogger.Error().Err(err).Msg("fatal error open Database aborting")
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
		childLogger.Error().Err(err).Send()
		panic(err)
	}
	
	// wire
	workerService := service.NewWorkerService(database, appServer.ApiService, workerEvent)
	httpRouters := api.NewHttpRouters(workerService)
	httpServer := server.NewHttpAppServer(appServer.Server)

	// start server
	httpServer.StartHttpAppServer(ctx, &httpRouters, &appServer)
}