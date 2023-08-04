// This is a companion to prometheus pushgateway
// It is aimed to allow the saving of some arbitrary data specifying customer and instance names
// The aim is to be wrapped by a script which checks in the result on a regular basis.
// The client which is pusing data to this tool via curl is report_instance_data.sh
package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"

	"datapushgateway/functions"

	"github.com/perforce/p4prometheus/version"
	"github.com/sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
)

// We extract the bcrypted passwords from the config file used for prometheus pushgateway
// A very simple yaml structure.
var usersPasswords = map[string][]byte{}

// mainLogger is declared at the package level for the main function.
var mainLogger *logrus.Logger

func main() {
	var (
		authFile = kingpin.Flag(
			"auth.file",
			"Config file for pushgateway specifying user_basic_auth and list of user/bcrypted passwords.",
		).String()
		port = kingpin.Flag(
			"port",
			"Port to listen on.",
		).Default(":9092").String()
		debug = kingpin.Flag(
			"debug",
			"Enable debugging.",
		).Bool()
		dataDir = kingpin.Flag(
			"data",
			"directory where to store uploaded data.",
		).Short('d').Default("data").String()
	)

	kingpin.Version(version.Print("datapushgateway"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()

	// Create the logger after parsing the debug flag
	mainLogger = logrus.New()
	if *debug {
		mainLogger.Level = logrus.DebugLevel
	} else {
		mainLogger.Level = logrus.InfoLevel
	}
	// Create the logger after parsing the debug flag
	functions.SetDebugMode(*debug)

	err := functions.ReadAuthFile(*authFile)
	if err != nil {
		mainLogger.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/" {
			http.NotFound(w, req)
			return
		}
		w.WriteHeader(200)
		fmt.Fprintf(w, "Data PushGateway\n")
	})
	// Update the handleJSONData call
	mux.HandleFunc("/json/", func(w http.ResponseWriter, req *http.Request) {
		functions.HandleJSONData(w, req, mainLogger, *dataDir)
	})

	// Update the /data/ endpoint
	mux.HandleFunc("/data/", func(w http.ResponseWriter, req *http.Request) {
		user, pass, ok := req.BasicAuth()
		if ok && functions.VerifyUserPass(user, pass) {
			fmt.Fprintf(w, "Processed\n")
			query := req.URL.Query()
			mainLogger.Debugf("Request Params: %v", query)
			customer := query.Get("customer")
			instance := query.Get("instance")
			if customer == "" || instance == "" {
				http.Error(w, "Please specify customer and instance", http.StatusBadRequest)
				return
			}
			body, err := io.ReadAll(req.Body)
			if err != nil {
				functions.Debugf("Error reading body: %v", err)
				http.Error(w, "can't read body\n", http.StatusBadRequest)
				return
			}
			mainLogger.Debugf("Request Body: %s", string(body))
			functions.SaveData(*dataDir, customer, instance, string(body), mainLogger)
			w.Write([]byte("Data saved\n"))
		} else {
			w.Header().Set("WWW-Authenticate", `Basic realm="api"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
		}
	})

	srv := &http.Server{
		Addr:    *port,
		Handler: mux,
		TLSConfig: &tls.Config{
			MinVersion:               tls.VersionTLS13,
			PreferServerCipherSuites: true,
		},
	}

	log.Printf("Starting server on %s", *port)
	err = srv.ListenAndServe()
	// .ListenAndServeTLS(*certFile, *keyFile)
	log.Fatal(err)
}
