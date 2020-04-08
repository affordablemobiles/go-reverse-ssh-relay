package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	co "github.com/a1comms/ssh-reverse-concentrator/shared"
	"github.com/gorilla/mux"
)

type StatusJSON struct {
	ListenPort     int                    `json:"listen_port"`
	Metadata       *co.JSONIdentityNotify `json:"metadata"`
	ConnectedSince time.Time              `json:"connected_since"`
}

func StatusJSONHandler(w http.ResponseWriter, r *http.Request) {
	clientMutex.RLock()
	defer clientMutex.RUnlock()

	retArray := []*StatusJSON{}

	for port, client := range clientMAP {
		retArray = append(retArray, &StatusJSON{
			ListenPort:     port,
			Metadata:       &client.metadata,
			ConnectedSince: client.connectedSince,
		})
	}

	sort.Slice(retArray[:], func(i, j int) bool {
		return retArray[i].ListenPort < retArray[j].ListenPort
	})

	data, err := json.MarshalIndent(retArray, "", "  ")
	if err != nil {
		log.Printf("Error Marshaling Status JSON: %s", err)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	w.WriteHeader(200)
	w.Write(data)
}

func StatusJSONPortHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	portIDString := vars["port"]
	if portIDString == "" {
		log.Printf("Invalid Port ID: %s", portIDString)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	port, err := strconv.Atoi(portIDString)
	if err != nil {
		log.Printf("Invalid Port ID: %s", portIDString)
		http.Error(w, "Internal Server Error", 500)
		return
	}

	clientMutex.RLock()
	defer clientMutex.RUnlock()

	if client, ok := clientMAP[port]; !ok {
		http.Error(w, "Endpoint Not Found", 404)
		return
	} else {
		retData := &StatusJSON{
			ListenPort:     port,
			Metadata:       &client.metadata,
			ConnectedSince: client.connectedSince,
		}

		data, err := json.MarshalIndent(retData, "", "  ")
		if err != nil {
			log.Printf("Error Marshaling Status JSON for port: %s", err)
			http.Error(w, "Internal Server Error", 500)
			return
		}

		w.WriteHeader(200)
		w.Write(data)
	}
}
