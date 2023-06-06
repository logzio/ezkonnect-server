package main

import (
	"fmt"
	"github.com/gorilla/mux"
	annotateapi "github.com/logzio/ezkonnect-server/api/annotate"
	stateapi "github.com/logzio/ezkonnect-server/api/state"
	"log"
	"net/http"
)

// main starts the server. Endpoints:
// 1. /api/v1/state - returns a list of all custom resources of type InstrumentedApplication
// 2. /api/v1/annotate/traces - handles the POST request for annotating a deployment or a statefulset
// 3. /api/v1/annotate/logs - handles the POST request for annotating a deployment or a statefulset with log annotations
func main() {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/api/v1/state", stateapi.GetCustomResourcesHandler).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/annotate/traces", annotateapi.UpdateTracesResourceAnnotations).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/annotate/logs", annotateapi.UpdateLogsResourceAnnotations).Methods(http.MethodPost)
	fmt.Println("Starting server on :5050")
	log.Fatal(http.ListenAndServe(":5050", router))
}
