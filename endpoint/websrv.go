package main

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

func startwebsrv() {
	r := mux.NewRouter()

	// Health handler.
	r.NotFoundHandler = http.HandlerFunc(HealthHandler)

	// Bind to a port and pass our router in
	http.ListenAndServe(":"+strconv.Itoa(globalConfig.HealthcheckListenPort), r)
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, startTime.Format(time.RFC3339Nano), 200)
	return
}
