package api

import (
	"encoding/json"
	"net/http"
	"github.com/rs/zerolog/log"
	"strconv"

	"github.com/go-fund-transfer/internal/core/service"
	"github.com/go-fund-transfer/internal/core/model"
	"github.com/go-fund-transfer/internal/core/erro"
	go_core_observ "github.com/eliezerraj/go-core/observability"
	"github.com/eliezerraj/go-core/coreJson"
	"github.com/gorilla/mux"
)

var childLogger = log.With().Str("adapter", "api.router").Logger()

var core_json coreJson.CoreJson
var core_apiError coreJson.APIError
var tracerProvider go_core_observ.TracerProvider

type HttpRouters struct {
	workerService 	*service.WorkerService
}

func NewHttpRouters(workerService *service.WorkerService) HttpRouters {
	return HttpRouters{
		workerService: workerService,
	}
}

// About return a health
func (h *HttpRouters) Health(rw http.ResponseWriter, req *http.Request) {
	childLogger.Debug().Msg("Health")

	health := true
	json.NewEncoder(rw).Encode(health)
}

// About return a live
func (h *HttpRouters) Live(rw http.ResponseWriter, req *http.Request) {
	childLogger.Debug().Msg("Live")

	live := true
	json.NewEncoder(rw).Encode(live)
}

// About show all header received
func (h *HttpRouters) Header(rw http.ResponseWriter, req *http.Request) {
	childLogger.Debug().Msg("Header")
	
	json.NewEncoder(rw).Encode(req.Header)
}

// About get the transfer transaction
func (h *HttpRouters) GetTransfer(rw http.ResponseWriter, req *http.Request) error {
	childLogger.Debug().Msg("GetTransfer")

	// trace
	span := tracerProvider.Span(req.Context(), "adapter.api.GetTransfer")
	defer span.End()

	//parameters
	vars := mux.Vars(req)
	varID, err := strconv.Atoi(vars["id"]) 
    if err != nil { 
		core_apiError = core_apiError.NewAPIError(err, http.StatusBadRequest)
		return  &core_apiError
    } 

	transfer := model.Transfer{}
	transfer.ID = varID

	// call service
	res, err := h.workerService.GetTransfer(req.Context(), &transfer)
	if err != nil {
		switch err {
		case erro.ErrNotFound:
			core_apiError = core_apiError.NewAPIError(err, http.StatusNotFound)
		default:
			core_apiError = core_apiError.NewAPIError(err, http.StatusInternalServerError)
		}
		return &core_apiError
	}
	
	return core_json.WriteJSON(rw, http.StatusOK, res)
}

// About add transfer transaction
func (h *HttpRouters) AddTransfer(rw http.ResponseWriter, req *http.Request) error {
	childLogger.Debug().Msg("AddTransfer")

	// trace
	span := tracerProvider.Span(req.Context(), "adapter.api.AddTransfer")
	defer span.End()

	//parameters
	transfer := model.Transfer{}
	err := json.NewDecoder(req.Body).Decode(&transfer)
    if err != nil {
		core_apiError = core_apiError.NewAPIError(err, http.StatusBadRequest)
		return &core_apiError
    }

	// call service
	res, err := h.workerService.AddTransfer(req.Context(), &transfer)
	if err != nil {
		switch err {
		case erro.ErrNotFound:
			core_apiError = core_apiError.NewAPIError(err, http.StatusNotFound)
		case erro.ErrTransInvalid:
			core_apiError = core_apiError.NewAPIError(err, http.StatusConflict)
		default:
			core_apiError = core_apiError.NewAPIError(err, http.StatusInternalServerError)
		}
		return &core_apiError
	}
	
	return core_json.WriteJSON(rw, http.StatusOK, res)
}

// About add transfer transaction via event
func (h *HttpRouters) AddTransferEvent(rw http.ResponseWriter, req *http.Request) error {
	childLogger.Debug().Msg("AddTransferEvent")

	// trace
	span := tracerProvider.Span(req.Context(), "adapter.api.AddTransferEvent")
	defer span.End()

	//parameters
	transfer := model.Transfer{}
	err := json.NewDecoder(req.Body).Decode(&transfer)
    if err != nil {
		core_apiError = core_apiError.NewAPIError(err, http.StatusBadRequest)
		return &core_apiError
    }

	// call service
	res, err := h.workerService.AddTransferEvent(req.Context(), &transfer)
	if err != nil {
		switch err {
		case erro.ErrNotFound:
			core_apiError = core_apiError.NewAPIError(err, http.StatusNotFound)
		case erro.ErrTransInvalid:
			core_apiError = core_apiError.NewAPIError(err, http.StatusConflict)
		default:
			core_apiError = core_apiError.NewAPIError(err, http.StatusInternalServerError)
		}
		return &core_apiError
	}
	
	return core_json.WriteJSON(rw, http.StatusOK, res)
}

// About add credit transaction via event
func (h *HttpRouters) CreditTransferEvent(rw http.ResponseWriter, req *http.Request) error {
	childLogger.Debug().Msg("CreditTransferEvent")

	// trace
	span := tracerProvider.Span(req.Context(), "adapter.api.CreditTransferEvent")
	defer span.End()

	//parameters
	transfer := model.Transfer{}
	err := json.NewDecoder(req.Body).Decode(&transfer)
    if err != nil {
		core_apiError = core_apiError.NewAPIError(err, http.StatusBadRequest)
		return &core_apiError
    }

	// call service
	res, err := h.workerService.CreditTransferEvent(req.Context(), &transfer)
	if err != nil {
		switch err {
		case erro.ErrNotFound:
			core_apiError = core_apiError.NewAPIError(err, http.StatusNotFound)
		case erro.ErrTransInvalid:
			core_apiError = core_apiError.NewAPIError(err, http.StatusConflict)
		case erro.ErrAmountInvalid:
			core_apiError = core_apiError.NewAPIError(err, http.StatusConflict)
		default:
			core_apiError = core_apiError.NewAPIError(err, http.StatusInternalServerError)
		}
		return &core_apiError
	}
	
	return core_json.WriteJSON(rw, http.StatusOK, res)
}

// About add debit transaction via event
func (h *HttpRouters) DebitTransferEvent(rw http.ResponseWriter, req *http.Request) error {
	childLogger.Debug().Msg("DebitTransferEvent")

	// trace
	span := tracerProvider.Span(req.Context(), "adapter.api.DebitTransferEvent")
	defer span.End()

	//parameters
	transfer := model.Transfer{}
	err := json.NewDecoder(req.Body).Decode(&transfer)
    if err != nil {
		core_apiError = core_apiError.NewAPIError(err, http.StatusBadRequest)
		return &core_apiError
    }

	// call service
	res, err := h.workerService.DebitTransferEvent(req.Context(), &transfer)
	if err != nil {
		switch err {
		case erro.ErrNotFound:
			core_apiError = core_apiError.NewAPIError(err, http.StatusNotFound)
		case erro.ErrTransInvalid:
			core_apiError = core_apiError.NewAPIError(err, http.StatusConflict)
		case erro.ErrAmountInvalid:
			core_apiError = core_apiError.NewAPIError(err, http.StatusConflict)
		default:
			core_apiError = core_apiError.NewAPIError(err, http.StatusInternalServerError)
		}
		return &core_apiError
	}
	
	return core_json.WriteJSON(rw, http.StatusOK, res)
}
