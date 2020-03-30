package api

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func (s *server) routes() {
	api := s.router.PathPrefix("/v1/minion").Subrouter()
	api.HandleFunc("/ping", s.PingHandler).Methods(http.MethodGet)
	api.HandleFunc("/version", s.VersionHandler).Methods(http.MethodGet)
	api.Handle("/metrics", promhttp.Handler()).Methods(http.MethodGet)

	api.HandleFunc("/{account}/jobs", s.JobsListHandler).Methods(http.MethodGet)

	api.HandleFunc("/{account}/jobs/{group}", s.JobsListHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/jobs/{group}", s.JobsCreateHandler).Methods(http.MethodPost)

	api.HandleFunc("/{account}/jobs/{group}/{id}", s.JobsShowHandler).Methods(http.MethodGet)
	api.HandleFunc("/{account}/jobs/{group}/{id}", s.JobsUpdateHandler).Methods(http.MethodPut)

	api.HandleFunc("/{account}/jobs/{group}", s.JobsDeleteHandler).Methods(http.MethodDelete)
	api.HandleFunc("/{account}/jobs/{group}/{id}", s.JobsDeleteHandler).Methods(http.MethodDelete)

	api.HandleFunc("/{account}/jobs/{group}/{id}", s.JobsRunHandler).Methods(http.MethodPatch)
}
