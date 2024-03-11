package handler

import (
	"time"
	"encoding/json"
	"net/http"
	"strconv"
	"os"
	"os/signal"
	"syscall"
	"context"
	"fmt"

	"github.com/gorilla/mux"

	"github.com/go-fund-transfer/internal/service"
	"github.com/go-fund-transfer/internal/core"
	"github.com/aws/aws-xray-sdk-go/xray"

)

type HttpWorkerAdapter struct {
	workerService 	*service.WorkerService
}

func NewHttpWorkerAdapter(workerService *service.WorkerService) *HttpWorkerAdapter {
	childLogger.Debug().Msg("NewHttpWorkerAdapter")
	return &HttpWorkerAdapter{
		workerService: workerService,
	}
}

type HttpServer struct {
	start 			time.Time
	httpAppServer 	core.HttpAppServer
}

func NewHttpAppServer(httpAppServer core.HttpAppServer) HttpServer {
	childLogger.Debug().Msg("NewHttpAppServer")

	return HttpServer{	start: time.Now(), 
						httpAppServer: httpAppServer,
					}
}

func (h HttpServer) StartHttpAppServer(ctx context.Context, httpWorkerAdapter *HttpWorkerAdapter) {
	childLogger.Info().Msg("StartHttpAppServer")
		
	myRouter := mux.NewRouter().StrictSlash(true)
	myRouter.Use(MiddleWareHandlerHeader)

	myRouter.HandleFunc("/", func(rw http.ResponseWriter, req *http.Request) {
		childLogger.Debug().Msg("/")
		json.NewEncoder(rw).Encode(h.httpAppServer)
	})

	myRouter.HandleFunc("/info", func(rw http.ResponseWriter, req *http.Request) {
		childLogger.Debug().Msg("/info")
		json.NewEncoder(rw).Encode(h.httpAppServer)
	})
	
	health := myRouter.Methods(http.MethodGet, http.MethodOptions).Subrouter()
    health.HandleFunc("/health", httpWorkerAdapter.Health)

	live := myRouter.Methods(http.MethodGet, http.MethodOptions).Subrouter()
    live.HandleFunc("/live", httpWorkerAdapter.Live)

	header := myRouter.Methods(http.MethodGet, http.MethodOptions).Subrouter()
    header.HandleFunc("/header", httpWorkerAdapter.Header)
	header.Use(MiddleWareHandlerHeader)

	transferFund := myRouter.Methods(http.MethodPost, http.MethodOptions).Subrouter()
	transferFund.Handle("/transfer", 
						xray.Handler(xray.NewFixedSegmentNamer(fmt.Sprintf("%s%s%s", "transfer:", h.httpAppServer.InfoPod.AvailabilityZone, ".add")), 
						http.HandlerFunc(httpWorkerAdapter.Transfer),
						),
	)
	transferFund.Use(httpWorkerAdapter.DecoratorDB)

	getTransfer := myRouter.Methods(http.MethodGet, http.MethodOptions).Subrouter()
	getTransfer.Handle("/get/{id}", 
						xray.Handler(xray.NewFixedSegmentNamer(fmt.Sprintf("%s%s%s", "transfer:", h.httpAppServer.InfoPod.AvailabilityZone, ".get")),
						http.HandlerFunc(httpWorkerAdapter.Get),
						),
	)
	getTransfer.Use(MiddleWareHandlerHeader)
	
	CreditFund := myRouter.Methods(http.MethodPost, http.MethodOptions).Subrouter()
	CreditFund.Handle("/creditFundSchedule", 
						xray.Handler(xray.NewFixedSegmentNamer(fmt.Sprintf("%s%s%s", "transfer:", h.httpAppServer.InfoPod.AvailabilityZone, ".creditFund")), 
						http.HandlerFunc(httpWorkerAdapter.CreditFundSchedule),
						),
	)
	CreditFund.Use(httpWorkerAdapter.DecoratorDB)
	
	DebitFund := myRouter.Methods(http.MethodPost, http.MethodOptions).Subrouter()
	DebitFund.Handle("/debitFundSchedule", 
						xray.Handler(xray.NewFixedSegmentNamer(fmt.Sprintf("%s%s%s", "transfer:", h.httpAppServer.InfoPod.AvailabilityZone, ".debitFund")), 
						http.HandlerFunc(httpWorkerAdapter.DebitFundSchedule),
						),
	)
	DebitFund.Use(httpWorkerAdapter.DecoratorDB)

	transferViaEvent := myRouter.Methods(http.MethodPost, http.MethodOptions).Subrouter()
	transferViaEvent.Handle("/transferViaEvent", 
						xray.Handler(xray.NewFixedSegmentNamer(fmt.Sprintf("%s%s%s", "transferViaEvent:", h.httpAppServer.InfoPod.AvailabilityZone, ".add")), 
						http.HandlerFunc(httpWorkerAdapter.TransferViaEvent),
						),
	)
	transferViaEvent.Use(httpWorkerAdapter.DecoratorDB)
	
	srv := http.Server{
		Addr:         ":" +  strconv.Itoa(h.httpAppServer.Server.Port),      	
		Handler:      myRouter,                	          
		ReadTimeout:  time.Duration(h.httpAppServer.Server.ReadTimeout) * time.Second,   
		WriteTimeout: time.Duration(h.httpAppServer.Server.WriteTimeout) * time.Second,  
		IdleTimeout:  time.Duration(h.httpAppServer.Server.IdleTimeout) * time.Second, 
	}

	childLogger.Info().Str("Service Port : ", strconv.Itoa(h.httpAppServer.Server.Port)).Msg("Service Port")

	go func() {
		err := srv.ListenAndServe()
		if err != nil {
			childLogger.Error().Err(err).Msg("Cancel http mux server !!!")
		}
	}()

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
	<-ch

	if err := srv.Shutdown(ctx); err != nil && err != http.ErrServerClosed {
		childLogger.Error().Err(err).Msg("WARNING Dirty Shutdown !!!")
		return
	}
	childLogger.Info().Msg("Stop Done !!!!")
}