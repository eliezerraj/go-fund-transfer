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
	Msg			any `json:"msg"`
}

func NewAPIError(statusCode int, err error) APIError {
	return APIError{
		StatusCode: statusCode,
		Msg:		err.Error(),
	}
}

// Middleware v02 - with decoratorDB
func (h *HttpWorkerAdapter) DecoratorDB(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		childLogger.Debug().Msg("-------------- Decorator - MiddleWareHandlerHeader (INICIO) --------------")
	
		/*if reqHeadersBytes, err := json.Marshal(r.Header); err != nil {
			childLogger.Error().Err(err).Msg("Could not Marshal http headers !!!")
		} else {
			childLogger.Debug().Str("Headers : ", string(reqHeadersBytes) ).Msg("")
		}

		childLogger.Debug().Str("Method : ", r.Method ).Msg("")
		childLogger.Debug().Str("URL : ", r.URL.Path ).Msg("")*/

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers","Content-Type,access-control-allow-origin, access-control-allow-headers")
	
		// If the user was informed then insert it in the session
		if string(r.Header.Get("client-id")) != "" {
			h.workerService.SetSessionVariable(r.Context(),string(r.Header.Get("client-id")))
		} else {
			h.workerService.SetSessionVariable(r.Context(),"NO_INFORMED")
		}

		childLogger.Debug().Msg("-------------- Decorator- MiddleWareHandlerHeader (FIM) ----------------")

		next.ServeHTTP(w, r)
	})
}

func (h *HttpWorkerAdapter) Health(rw http.ResponseWriter, req *http.Request) {
	childLogger.Debug().Msg("Health")

	health := true
	json.NewEncoder(rw).Encode(health)
	return
}

func (h *HttpWorkerAdapter) Live(rw http.ResponseWriter, req *http.Request) {
	childLogger.Debug().Msg("Live")

	live := true
	json.NewEncoder(rw).Encode(live)
	return
}

func (h *HttpWorkerAdapter) Header(rw http.ResponseWriter, req *http.Request) {
	childLogger.Debug().Msg("Header")
	
	json.NewEncoder(rw).Encode(req.Header)
	return
}

func (h *HttpWorkerAdapter) Transfer( rw http.ResponseWriter, req *http.Request) {
	childLogger.Debug().Msg("Transfer")

	transfer := core.Transfer{}
	err := json.NewDecoder(req.Body).Decode(&transfer)
    if err != nil {
		apiError := NewAPIError(400, erro.ErrUnmarshal)
		rw.WriteHeader(apiError.StatusCode)
		json.NewEncoder(rw).Encode(apiError)
		return
    }

	res, err := h.workerService.Transfer(req.Context(), transfer)
	if err != nil {
		var apiError APIError
		switch err {
			case erro.ErrNotFound:
				apiError = NewAPIError(404, err)
			default:
				apiError = NewAPIError(500, err)
		}
		rw.WriteHeader(apiError.StatusCode)
		json.NewEncoder(rw).Encode(apiError)
		return
	}

	json.NewEncoder(rw).Encode(res)
	return
}

func (h *HttpWorkerAdapter) CreditFundSchedule( rw http.ResponseWriter, req *http.Request) {
	childLogger.Debug().Msg("CreditFundSchedule")

	transfer := core.Transfer{}
	err := json.NewDecoder(req.Body).Decode(&transfer)
    if err != nil {
		rw.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(rw).Encode(erro.ErrUnmarshal.Error())
        return
    }

	res, err := h.workerService.CreditFundSchedule(req.Context(), &transfer)
	if err != nil {
		var apiError APIError
		switch err {
			case erro.ErrNotFound:
				apiError = NewAPIError(404, err)
			default:
				apiError = NewAPIError(500, err)
		}
		rw.WriteHeader(apiError.StatusCode)
		json.NewEncoder(rw).Encode(apiError)
		return
	}

	json.NewEncoder(rw).Encode(res)
	return
}

func (h *HttpWorkerAdapter) DebitFundSchedule( rw http.ResponseWriter, req *http.Request) {
	childLogger.Debug().Msg("DebitFundSchedule")

	transfer := core.Transfer{}
	err := json.NewDecoder(req.Body).Decode(&transfer)
    if err != nil {
		apiError := NewAPIError(400, erro.ErrUnmarshal)
		rw.WriteHeader(apiError.StatusCode)
		json.NewEncoder(rw).Encode(apiError)
		return
    }

	res, err := h.workerService.DebitFundSchedule(req.Context(), &transfer)
	if err != nil {
		var apiError APIError
		switch err {
			case erro.ErrNotFound:
				apiError = NewAPIError(404, err)
			default:
				apiError = NewAPIError(500, err)
		}
		rw.WriteHeader(apiError.StatusCode)
		json.NewEncoder(rw).Encode(apiError)
		return
	}

	json.NewEncoder(rw).Encode(res)
	return
}

func (h *HttpWorkerAdapter) Get(rw http.ResponseWriter, req *http.Request) {
	childLogger.Debug().Msg("Get")

	vars := mux.Vars(req)
	transfer := core.Transfer{}

	varID, err := strconv.Atoi(vars["id"]) 
    if err != nil { 
		apiError := NewAPIError(400, erro.ErrInvalidId)
		rw.WriteHeader(apiError.StatusCode)
		json.NewEncoder(rw).Encode(apiError)
		return
    } 
  
	transfer.ID = varID
	res, err := h.workerService.Get(req.Context(), &transfer)
	if err != nil {
		var apiError APIError
		switch err {
			default:
				apiError = NewAPIError(500, err)
		}
		rw.WriteHeader(apiError.StatusCode)
		json.NewEncoder(rw).Encode(apiError)
		return
	}

	json.NewEncoder(rw).Encode(res)
	return
}

func (h *HttpWorkerAdapter) TransferViaEvent( rw http.ResponseWriter, req *http.Request) {
	childLogger.Debug().Msg("TransferViaEvent")

	transfer := core.Transfer{}
	err := json.NewDecoder(req.Body).Decode(&transfer)
    if err != nil {
		apiError := NewAPIError(400, erro.ErrUnmarshal)
		rw.WriteHeader(apiError.StatusCode)
		json.NewEncoder(rw).Encode(apiError)
		return
    }

	res, err := h.workerService.TransferViaEvent(req.Context(), &transfer)
	if err != nil {
		var apiError APIError
		switch err {
			case erro.ErrNotFound:
				apiError = NewAPIError(404, err)
			default:
				apiError = NewAPIError(500, err)
		}
		rw.WriteHeader(apiError.StatusCode)
		json.NewEncoder(rw).Encode(apiError)
		return
	}

	json.NewEncoder(rw).Encode(res)
	return
}