package main

import (
	"log"
	"net/http"

	"github.com/brigadecore/brigade-dockerhub-gateway/internal/webhooks"
	libHTTP "github.com/brigadecore/brigade-foundations/http"
	"github.com/brigadecore/brigade-foundations/signals"
	"github.com/brigadecore/brigade-foundations/version"
	"github.com/brigadecore/brigade/sdk/v2/core"
	"github.com/gorilla/mux"
)

func main() {

	log.Printf(
		"Starting Brigade Docker Hub Gateway -- version %s -- commit %s",
		version.Version(),
		version.Commit(),
	)

	var webhooksService webhooks.Service
	{
		address, token, opts, err := apiClientConfig()
		if err != nil {
			log.Fatal(err)
		}
		webhooksService = webhooks.NewService(
			core.NewEventsClient(address, token, &opts),
		)
	}

	var tokenFilter libHTTP.Filter
	{
		config, err := tokenFilterConfig()
		if err != nil {
			log.Fatal(err)
		}
		tokenFilter = webhooks.NewTokenFilter(config)
	}

	var server libHTTP.Server
	{
		handler, err := webhooks.NewHandler(webhooksService)
		if err != nil {
			log.Fatal(err)
		}
		router := mux.NewRouter()
		router.StrictSlash(true)
		router.Handle(
			"/events",
			tokenFilter.Decorate(handler.ServeHTTP),
		).Methods(http.MethodPost)
		router.HandleFunc("/healthz", libHTTP.Healthz).Methods(http.MethodGet)
		serverConfig, err := serverConfig()
		if err != nil {
			log.Fatal(err)
		}
		server = libHTTP.NewServer(router, &serverConfig)
	}

	log.Println(
		server.ListenAndServe(signals.Context()),
	)
}
