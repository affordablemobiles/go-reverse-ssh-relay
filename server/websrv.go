package main

import (
	"net/http"
	"strconv"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func startWebSocket() {
	r := mux.NewRouter()

	// Just give a 403 to nosey people trying to hit the index.
	r.HandleFunc("/", IndexHandler)

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
