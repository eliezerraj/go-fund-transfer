package webserver

import(
	"time"
	"context"
	"encoding/json"

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
	"github.com/go-fund-transfer/internal/handler/listener"
	"github.com/go-fund-transfer/internal/config/secret_manager_aws"
)

var(
	logLevel = zerolog.DebugLevel
	appServer	core.AppServer
	producerWorker	event.EventNotifier
	restApiCallData core.RestApiCallData
)

func init(){
	log.Debug().Msg("init")
	zerolog.SetGlobalLevel(logLevel)

	infoPod , server, restEndpoint, awsServiceConfig := util.GetInfoPod()
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
	appServer.AwsServiceConfig = &awsServiceConfig
}

func Server() {
	log.Debug().Msg("----------------------------------------------------")
	log.Debug().Msg("main")
	log.Debug().Msg("----------------------------------------------------")
	log.Debug().Interface("appServer :",appServer).Msg("")
	log.Debug().Msg("----------------------------------------------------")

	ctx, cancel := context.WithTimeout(context.Background(), 
									time.Duration( appServer.Server.ReadTimeout) * time.Second)
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

	// Load DSA secrets
	clientSecretManager, err := secret_manager_aws.NewClientSecretManager(ctx, *appServer.AwsServiceConfig)
	if err != nil {
		log.Error().Err(err).Msg("erro NewClientSecretManager")
	}
	res_secret, err :=clientSecretManager.GetSecret(ctx, appServer.AwsServiceConfig.SecretJwtSACredential)
	if err != nil {
		log.Error().Err(err).Msg("erro GetSecret")
	}

	var secretData map[string]string
	if err := json.Unmarshal([]byte(*res_secret), &secretData); err != nil {
		log.Error().Err(err).Msg("erro Unmarshal Secret")
	}

	//Setup JWT SA
	restApiCallData.Method = "POST"
	restApiCallData.Url = appServer.AwsServiceConfig.ServiceUrlJwtSA +"/oauth_credential"
	restApiCallData.UsernameAuth = secretData["username"]
	restApiCallData.PasswordAuth = secretData["password"]
	appServer.RestApiCallData = &restApiCallData

	// Setup Service (usecase)
	restApiService	:= restapi.NewRestApiService(&appServer)

	//Token Refresh
	token_refresh := listener.NewToken(	context.Background(), 
										listener.AuthFuncTest,
										restApiCallData, 
										restApiService.CallApiRest)

	workerService := service.NewWorkerService(	&repoDatabase, 
												&appServer,
												restApiService, 
												producerWorker, 
												appServer.KafkaConfig.Topic,
												token_refresh)

	httpWorkerAdapter 	:= controller.NewHttpWorkerAdapter(workerService)
	httpServer 			:= handler.NewHttpAppServer(appServer.Server)
	
	httpServer.StartHttpAppServer(ctx,&httpWorkerAdapter,&appServer)
}