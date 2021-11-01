package main

import (
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/mmattice/go-openvpn/server/mi"
	"log"
	"net/http"
)

var listenPort int
var miHost string
var miPort int

func init() {
	const (
		defaultHost       = "localhost"
		usageHost         = "Management Interface IP"
		defaultListenPort = 1196
		usageListenPort   = "health check listen port"
		defaultPort       = 11960
		usagePort         = "Management Interface Port"
	)
	flag.StringVar(&miHost, "host", defaultHost, usageHost)
	flag.StringVar(&miHost, "h", defaultHost, usageHost+" (shorthand)")
	flag.IntVar(&listenPort, "lport", defaultListenPort, usageListenPort)
	flag.IntVar(&listenPort, "l", defaultListenPort, usageListenPort+" (shorthand)")
	flag.IntVar(&miPort, "port", defaultPort, usagePort)
	flag.IntVar(&miPort, "p", defaultPort, usagePort+" (shorthand)")
}

func main() {
	flag.Parse()
	var r = mux.NewRouter()
	r.HandleFunc("/status", statusHandler)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", listenPort), r))
}

func statusHandler(w http.ResponseWriter, _ *http.Request) {
	var client = mi.NewClient("tcp", fmt.Sprintf("%s:%d", miHost, miPort))
	var ls, err = client.GetLoadStats()
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		currentStatus := "Unavailable"
		_, err = w.Write([]byte(fmt.Sprintf("OpenVPN Server %s - Cannot connect\n", currentStatus)))
	} else {
		w.WriteHeader(http.StatusOK)
		_, err = w.Write([]byte(fmt.Sprintf("OpenVPN Healthy %d %d %d\n", ls.NClients, ls.BytesIn, ls.BytesOut)))
	}
}
