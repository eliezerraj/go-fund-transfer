package controller

import (	
	"strconv"
	"net/http"
	"encoding/json"
	"github.com/rs/zerolog/log"
	"github.com/gorilla/mux"

	"github.com/go-fund-transfer/internal/core"
	"github.com/go-fund-transfer/internal/erro"
	"github.com/go-fund-transfer/internal/service"	
)

var childLogger = log.With().Str("handler", "controller").Logger()

//-------------------------------------------------------
type HttpWorkerAdapter struct {
	workerService 	*service.WorkerService
}

func NewHttpWorkerAdapter(workerService *service.WorkerService) HttpWorkerAdapter {
	childLogger.Debug().Msg("NewHttpWorkerAdapter")
	
	return HttpWorkerAdapter{
		workerService: workerService,
	}
}

type APIError struct {
	StatusCode	int  `json:"statusCode"`
	Msg			string `json:"msg"`
}

func (e APIError) Error() string {
	return e.Msg
}

func NewAPIError(statusCode int, err error) APIError {
	return APIError{
		StatusCode: statusCode,
		Msg:		err.Error(),
	}
}

func WriteJSON(rw http.ResponseWriter, code int, v any) error{
	rw.WriteHeader(code)
	return json.NewEncoder(rw).Encode(v)
}

func (h *HttpWorkerAdapter) Health(rw http.ResponseWriter, req *http.Request) {
	childLogger.Debug().Msg("Health")

	health := true
	json.NewEncoder(rw).Encode(health)
}

func (h *HttpWorkerAdapter) Live(rw http.ResponseWriter, req *http.Request) {
	childLogger.Debug().Msg("Live")

	live := true
	json.NewEncoder(rw).Encode(live)
}

func (h *HttpWorkerAdapter) Header(rw http.ResponseWriter, req *http.Request) {
	childLogger.Debug().Msg("Header")
	
	json.NewEncoder(rw).Encode(req.Header)
}

func (h *HttpWorkerAdapter) Transfer( rw http.ResponseWriter, req *http.Request) error {
	childLogger.Debug().Msg("Transfer")

	transfer := core.Transfer{}
	err := json.NewDecoder(req.Body).Decode(&transfer)
    if err != nil {
		apiError := NewAPIError(http.StatusBadRequest, erro.ErrUnmarshal)
		return apiError
    }
	defer req.Body.Close()

	res, err := h.workerService.Transfer(req.Context(), transfer)
	if err != nil {
		var apiError APIError
		switch err {
			case erro.ErrNotFound:
				apiError = NewAPIError(http.StatusNotFound, err)
			default:
				apiError = NewAPIError(http.StatusInternalServerError, err)
		}
		return apiError
	}

	return WriteJSON(rw, http.StatusOK, res)
}

func (h *HttpWorkerAdapter) CreditFundSchedule( rw http.ResponseWriter, req *http.Request) error {
	childLogger.Debug().Msg("CreditFundSchedule")

	transfer := core.Transfer{}
	err := json.NewDecoder(req.Body).Decode(&transfer)
    if err != nil {
		apiError := NewAPIError(http.StatusBadRequest, erro.ErrUnmarshal)
		return apiError
    }
	defer req.Body.Close()

	res, err := h.workerService.CreditFundSchedule(req.Context(), &transfer)
	if err != nil {
		var apiError APIError
		switch err {
			case erro.ErrNotFound:
				apiError = NewAPIError(http.StatusNotFound, err)
			default:
				apiError = NewAPIError(http.StatusInternalServerError, err)
		}
		return apiError
	}

	return WriteJSON(rw, http.StatusOK, res)
}

func (h *HttpWorkerAdapter) DebitFundSchedule( rw http.ResponseWriter, req *http.Request) error {
	childLogger.Debug().Msg("DebitFundSchedule")

	transfer := core.Transfer{}
	err := json.NewDecoder(req.Body).Decode(&transfer)
    if err != nil {
		apiError := NewAPIError(http.StatusBadRequest, erro.ErrUnmarshal)
		return apiError
    }

	res, err := h.workerService.DebitFundSchedule(req.Context(), &transfer)
	if err != nil {
		var apiError APIError
		switch err {
			case erro.ErrNotFound:
				apiError = NewAPIError(http.StatusNotFound, err)
			default:
				apiError = NewAPIError(http.StatusInternalServerError, err)
		}
		return apiError
	}

	return WriteJSON(rw, http.StatusOK, res)
}

func (h *HttpWorkerAdapter) Get(rw http.ResponseWriter, req *http.Request) error {
	childLogger.Debug().Msg("Get")

	vars := mux.Vars(req)
	transfer := core.Transfer{}

	varID, err := strconv.Atoi(vars["id"]) 
    if err != nil { 
		apiError := NewAPIError(http.StatusBadRequest, erro.ErrInvalidId)
		return apiError
    } 
  
	transfer.ID = varID
	res, err := h.workerService.Get(req.Context(), &transfer)
	if err != nil {
		var apiError APIError
		switch err {
			case erro.ErrNotFound:
				apiError = NewAPIError(http.StatusNotFound, err)
			default:
				apiError = NewAPIError(http.StatusInternalServerError, err)
		}
		return apiError
	}

	return WriteJSON(rw, http.StatusOK, res)
}

func (h *HttpWorkerAdapter) TransferViaEvent( rw http.ResponseWriter, req *http.Request) error {
	childLogger.Debug().Msg("TransferViaEvent")

	transfer := core.Transfer{}
	err := json.NewDecoder(req.Body).Decode(&transfer)
    if err != nil {
		apiError := NewAPIError(http.StatusBadRequest, erro.ErrUnmarshal)
		return apiError
    }
	defer req.Body.Close()

	res, err := h.workerService.TransferViaEvent(req.Context(), &transfer)
	if err != nil {
		var apiError APIError
		switch err {
			case erro.ErrNotFound:
				apiError = NewAPIError(http.StatusNotFound, err)
			default:
				apiError = NewAPIError(http.StatusInternalServerError, err)
		}
		return apiError
	}

	return WriteJSON(rw, http.StatusOK, res)
}