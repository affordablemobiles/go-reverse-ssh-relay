package main

import (
	"net/http"
	"strconv"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func startWebStatus() {
	r := mux.NewRouter()

	// Just give a 403 to nosey people trying to hit the index.
	r.HandleFunc("/", IndexHandler)

	// Health handler.
	r.HandleFunc("/healthz", HealthHandler)

	api_base := mux.NewRouter()
	r.PathPrefix("/api").Handler(negroni.New(
		negroni.Wrap(handlers.ProxyHeaders(api_base)),
	))
	api_sub := api_base.PathPrefix("/api").Subrouter()

	// Pinpad API functions.
	api_sub.HandleFunc("/status_{port}.json", StatusJSONPortHandler)
	api_sub.HandleFunc("/status.json", StatusJSONHandler)
	// Pinpad API functions end.

	// Bind to a port and pass our router in
	http.ListenAndServe(":"+strconv.Itoa(globalConfig.ListenPortStatus), r)
}

func startWebSocket() {
	r := mux.NewRouter()

	// Just give a 403 to nosey people trying to hit the index.
	r.HandleFunc("/", IndexHandler)

	// Health handler.
	r.HandleFunc("/healthz", HealthHandler)

	api_base := mux.NewRouter()
	r.PathPrefix("/api").Handler(negroni.New(
		negroni.Wrap(handlers.ProxyHeaders(api_base)),
	))
	api_sub := api_base.PathPrefix("/api").Subrouter()

	// Pinpad API functions.
	api_sub.HandleFunc("/client_ws", ClientWSHandler)
	// Pinpad API functions end.

	// Bind to a port and pass our router in
	http.ListenAndServe(":"+strconv.Itoa(globalConfig.ListenPort), r)
}

func IndexHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(403)
	w.Write([]byte("Permission Denied"))
}

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "OK", 200)
	return
}
