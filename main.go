package main

import (
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/mmattice/go-openvpn-mgmt/openvpn"
	"log"
	"net/http"
	"time"
)

var listenPort int
var miHost string
var miPort int

func init() {
	const (
		defaultHost       = "localhost"
		usageHost         = "Management Interface IP"
		defaultListenPort = 1194
		usageListenPort   = "health check listen port"
		defaultPort       = 11940
		usagePort         = "Management Interface Port"
	)
	flag.StringVar(&miHost, "host", defaultHost, usageHost)
	flag.StringVar(&miHost, "h", defaultHost, usageHost+" (shorthand)")
	flag.IntVar(&listenPort, "lport", defaultListenPort, usageListenPort)
	flag.IntVar(&listenPort, "l", defaultListenPort, usageListenPort+" (shorthand)")
	flag.IntVar(&miPort, "port", defaultPort, usagePort)
	flag.IntVar(&miPort, "p", defaultPort, usagePort+" (shorthand)")
}

func connect(addr string, lsCh chan<- openvpn.LoadStat) {
	for {
		eventCh := make(chan openvpn.Event, 10)
		var mgmt, err = openvpn.Dial(addr, eventCh)
		if err != nil {
			panic(err)
		}
		go func() {
			for {
				time.Sleep(1 * time.Second)
				loadStat, err := mgmt.LoadStats()
				if err != nil {
					lsCh <- openvpn.LoadStat{Clients: -1, BytesIn: -1, BytesOut: -1}
					break
				} else {
					lsCh <- loadStat
				}
			}
		}()
		for range eventCh {
		}
	}

}

var ls = openvpn.LoadStat{Clients: -1, BytesIn: -1, BytesOut: -1}

func main() {
	flag.Parse()
	var r = mux.NewRouter()
	lsCh := make(chan openvpn.LoadStat, 10)
    go connect(fmt.Sprintf("%s:%d", miHost, miPort), lsCh)
	go func() {
		for loadStat := range lsCh {
			ls = loadStat
		}
	}()
	r.HandleFunc("/status", statusHandler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", listenPort), r))
}



func statusHandler(w http.ResponseWriter, _ *http.Request) {
	if ls.Clients < 0 {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("OpenVPN Server Unavailable - Cannot connect\n"))
	} else {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(fmt.Sprintf("OpenVPN Healthy %d %d %d\n", ls.Clients, ls.BytesIn, ls.BytesOut)))
	}
}
