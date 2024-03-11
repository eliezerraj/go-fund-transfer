package main

import(
	"time"
	"os"
	"strconv"
	"net"
	"io/ioutil"
	"context"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
    "github.com/aws/aws-sdk-go-v2/config"

	"github.com/go-fund-transfer/internal/adapter/event"
	"github.com/go-fund-transfer/internal/handler"
	"github.com/go-fund-transfer/internal/core"
	"github.com/go-fund-transfer/internal/service"
	"github.com/go-fund-transfer/internal/repository/postgre"
	"github.com/go-fund-transfer/internal/adapter/restapi"
	
)

var(
	logLevel 	= 	zerolog.DebugLevel
	noAZ		=	true // set only if you get to split the xray trace per AZ
	serverUrlDomain, xApigwId 		string
	infoPod					core.InfoPod
	envDB	 				core.DatabaseRDS
	envKafka				core.KafkaConfig
	httpAppServerConfig 	core.HttpAppServer
	server					core.Server
	topic					core.Topic
	dataBaseHelper 			postgre.DatabaseHelper
	repoDB					postgre.WorkerRepository
)

func init(){
	log.Debug().Msg("init")
	zerolog.SetGlobalLevel(logLevel)
	
	err := godotenv.Load(".env")
	if err != nil {
		log.Info().Err(err).Msg("No .ENV File !!!!")
	}

	getEnv()

	server.ReadTimeout = 60
	server.WriteTimeout = 60
	server.IdleTimeout = 60
	server.CtxTimeout = 60

	infoPod.Topic = &topic

	// Get Database Secrets
	file_user, err := ioutil.ReadFile("/var/pod/secret/username")
	if err != nil {
		log.Error().Err(err).Msg("ERRO FATAL recuperacao secret-user")
		os.Exit(3)
	}
	file_pass, err := ioutil.ReadFile("/var/pod/secret/password")
	if err != nil {
		log.Error().Err(err).Msg("ERRO FATAL recuperacao secret-pass")
		os.Exit(3)
	}
	envDB.User = string(file_user)
	envDB.Password = string(file_pass)
	envDB.Db_timeout = 90

	// Load info pod
	// Get IP
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Error().Err(err).Msg("Error to get the POD IP address !!!")
		os.Exit(3)
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				infoPod.IPAddress = ipnet.IP.String()
			}
		}
	}
	infoPod.OSPID = strconv.Itoa(os.Getpid())

	// Get AZ only if localtest is true
	if (noAZ != true) {
		cfg, err := config.LoadDefaultConfig(context.TODO())
		if err != nil {
			log.Error().Err(err).Msg("ERRO FATAL get Context !!!")
			os.Exit(3)
		}
		client := imds.NewFromConfig(cfg)
		response, err := client.GetInstanceIdentityDocument(context.TODO(), &imds.GetInstanceIdentityDocumentInput{})
		if err != nil {
			log.Error().Err(err).Msg("Unable to retrieve the region from the EC2 instance !!!")
			os.Exit(3)
		}
		infoPod.AvailabilityZone = response.AvailabilityZone	
	} else {
		infoPod.AvailabilityZone = "-"
	}
	// Load info pod
	infoPod.Database = &envDB
	infoPod.KafkaConfig =&envKafka
}

func getEnv() {
	log.Debug().Msg("getEnv")

	if os.Getenv("API_VERSION") !=  "" {
		infoPod.ApiVersion = os.Getenv("API_VERSION")
	}
	if os.Getenv("POD_NAME") !=  "" {
		infoPod.PodName = os.Getenv("POD_NAME")
	}
	if os.Getenv("PORT") !=  "" {
		intVar, _ := strconv.Atoi(os.Getenv("PORT"))
		server.Port = intVar
	}

	if os.Getenv("DB_HOST") !=  "" {
		envDB.Host = os.Getenv("DB_HOST")
	}
	if os.Getenv("DB_PORT") !=  "" {
		envDB.Port = os.Getenv("DB_PORT")
	}
	if os.Getenv("DB_NAME") !=  "" {	
		envDB.DatabaseName = os.Getenv("DB_NAME")
	}
	if os.Getenv("DB_SCHEMA") !=  "" {	
		envDB.Schema = os.Getenv("DB_SCHEMA")
	}
	if os.Getenv("DB_DRIVER") !=  "" {	
		envDB.Postgres_Driver = os.Getenv("DB_DRIVER")
	}
	
	if os.Getenv("SERVER_URL_DOMAIN") !=  "" {	
		serverUrlDomain = os.Getenv("SERVER_URL_DOMAIN")
	}
	if os.Getenv("X_APIGW_API_ID") !=  "" {	
		xApigwId = os.Getenv("X_APIGW_API_ID")
	}

	if os.Getenv("KAFKA_USER") !=  "" {
		envKafka.KafkaConfigurations.Username = os.Getenv("KAFKA_USER")
	}
	if os.Getenv("KAFKA_PASSWORD") !=  "" {
		envKafka.KafkaConfigurations.Password = os.Getenv("KAFKA_PASSWORD")
	}
	if os.Getenv("KAFKA_PROTOCOL") !=  "" {
		envKafka.KafkaConfigurations.Protocol = os.Getenv("KAFKA_PROTOCOL")
	}
	if os.Getenv("KAFKA_MECHANISM") !=  "" {
		envKafka.KafkaConfigurations.Mechanisms = os.Getenv("KAFKA_MECHANISM")
	}
	if os.Getenv("KAFKA_CLIENT_ID") !=  "" {
		envKafka.KafkaConfigurations.Clientid = os.Getenv("KAFKA_CLIENT_ID")
	}
	if os.Getenv("KAFKA_BROKER_1") !=  "" {
		envKafka.KafkaConfigurations.Brokers1 = os.Getenv("KAFKA_BROKER_1")
	}
	if os.Getenv("KAFKA_BROKER_2") !=  "" {
		envKafka.KafkaConfigurations.Brokers2 = os.Getenv("KAFKA_BROKER_2")
	}
	if os.Getenv("KAFKA_BROKER_3") !=  "" {
		envKafka.KafkaConfigurations.Brokers3 = os.Getenv("KAFKA_BROKER_3")
	}

	if os.Getenv("KAFKA_PARTITION") !=  "" {
		intVar, _ := strconv.Atoi(os.Getenv("KAFKA_PARTITION"))
		envKafka.KafkaConfigurations.Partition = intVar
	}
	if os.Getenv("KAFKA_REPLICATION") !=  "" {
		intVar, _ := strconv.Atoi(os.Getenv("KAFKA_REPLICATION"))
		envKafka.KafkaConfigurations.ReplicationFactor = intVar
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

	if os.Getenv("NO_AZ") == "false" {	
		noAZ = false
	} else {
		noAZ = true
	}
}

func main() {
	log.Debug().Msg("----------------------------------------------------")
	log.Debug().Msg("main")
	log.Debug().Interface("envDB : ",envDB).Msg("")
	log.Debug().Msg("----------------------------------------------------")
	log.Debug().Interface("server: ",server).Msg("")
	log.Debug().Msg("----------------------------------------------------")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration( server.ReadTimeout ) * time.Second)
	defer cancel()

	// Open Database
	count := 1
	var err error
	for {
		dataBaseHelper, err = postgre.NewDatabaseHelper(ctx, envDB)
		if err != nil {
			if count < 3 {
				log.Error().Err(err).Msg("Erro na abertura do Database")
			} else {
				log.Error().Err(err).Msg("ERRO FATAL na abertura do Database aborting")
				panic(err)
			}
			time.Sleep(3 * time.Second)
			count = count + 1
			continue
		}
		break
	}
	
	// Setup msk
	producerWorker, err := event.NewProducerWorker(&envKafka)
	if err != nil {
		log.Error().Err(err).Msg("Erro na abertura do Kafka")
	}
	
	restapi	:= restapi.NewRestApi(serverUrlDomain, xApigwId)

	httpAppServerConfig.Server = server
	repoDB = postgre.NewWorkerRepository(dataBaseHelper)

	workerService := service.NewWorkerService(&repoDB, restapi, producerWorker, &topic)
	httpWorkerAdapter := handler.NewHttpWorkerAdapter(workerService)

	httpAppServerConfig.InfoPod = &infoPod
	httpServer := handler.NewHttpAppServer(httpAppServerConfig)

	httpServer.StartHttpAppServer(ctx, httpWorkerAdapter)
}