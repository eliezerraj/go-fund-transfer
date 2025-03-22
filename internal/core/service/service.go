package service

import(
	"github.com/go-fund-transfer/internal/core/model"
	"github.com/go-fund-transfer/internal/adapter/database"
	"github.com/go-fund-transfer/internal/adapter/event"

	"github.com/rs/zerolog/log"
)

var childLogger = log.With().Str("component","go-fund-transfer").Str("package","internal.core.service").Logger()

type WorkerService struct {
	workerRepository *database.WorkerRepository
	apiService		[]model.ApiService
	workerEvent		*event.WorkerEvent
}

func NewWorkerService(	workerRepository *database.WorkerRepository, 
						apiService	[]model.ApiService,
						workerEvent	*event.WorkerEvent) *WorkerService{
	childLogger.Info().Str("func","NewWorkerService").Send()

	return &WorkerService{
		workerRepository: workerRepository,
		apiService: apiService,
		workerEvent: workerEvent,
	}
}