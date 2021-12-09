package main

import (
	"flag"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/gorilla/mux"
	"github.com/mmattice/go-openvpn-mgmt/openvpn"
	"github.com/rcrowley/go-metrics"
	"github.com/sclasen/go-metrics-cloudwatch/config"
	"github.com/sclasen/go-metrics-cloudwatch/reporter"
	"log"
	"net/http"
	"time"
)

var listenPort int
var miHost string
var miPort int
var miSock string
var debug bool
var publish bool
var metricsInterval time.Duration

func init() {
	const (
		defaultHost       = "localhost"
		usageHost         = "Management Interface IP"
		defaultListenPort = 1194
		usageListenPort   = "health check listen port"
		defaultPort       = 11940
		usagePort         = "Management Interface Port"
		defaultDebug      = false
		usageDebug        = "enable debugging"
		defaultSock       = ""
		usageSock         = "unix socket location"
		defaultInterval   = 30 * time.Second
		usageInterval     = "golang interval definition for metric publish"
		defaultPublish    = false
		usagePublish      = "publish metrics"
	)
	flag.StringVar(&miHost, "host", defaultHost, usageHost)
	flag.StringVar(&miHost, "h", defaultHost, usageHost+" (shorthand)")
	flag.IntVar(&listenPort, "lport", defaultListenPort, usageListenPort)
	flag.IntVar(&listenPort, "l", defaultListenPort, usageListenPort+" (shorthand)")
	flag.IntVar(&miPort, "port", defaultPort, usagePort)
	flag.IntVar(&miPort, "p", defaultPort, usagePort+" (shorthand)")
	flag.BoolVar(&debug, "d", defaultDebug, usageDebug)
	flag.StringVar(&miSock, "s", defaultSock, usageSock + " (shorthand)")
	flag.StringVar(&miSock, "socket", defaultSock, usageSock)
	flag.BoolVar(&publish, "publish", defaultPublish, usagePublish)
	flag.DurationVar(&metricsInterval, "i", defaultInterval, usageInterval)
}

func managementConnect(addr string, lsCh chan<- openvpn.LoadStat) {
	for {
		eventCh := make(chan openvpn.Event, 10)
		var mgmt, err = openvpn.Dial(addr, eventCh)
		if err != nil {
			if debug { log.Printf("Failed to connect to '%s' - %s\n", addr, err.Error())}
			lsCh <- openvpn.LoadStat{Clients: -1, BytesIn: -1, BytesOut: -1}
		} else {
			go gatherLoadStats(mgmt, lsCh)
			for range eventCh {
			}
		}
		time.Sleep(1 * time.Second)
	}
}

func gatherLoadStats(mgmt *openvpn.MgmtClient, lsCh chan<- openvpn.LoadStat) {
	for {
		time.Sleep(1 * time.Second)
		loadStat, err := mgmt.LoadStats()
		if debug {
			log.Printf("load-stats: %d %d %d\n", loadStat.Clients, loadStat.BytesIn, loadStat.BytesOut)
		}
		if err != nil {
			lsCh <- openvpn.LoadStat{Clients: -1, BytesIn: -1, BytesOut: -1}
			break
		} else {
			lsCh <- loadStat
		}
	}
}

var ls = openvpn.LoadStat{Clients: -1, BytesIn: -1, BytesOut: -1}

func handleLoadStatChannel(lsCh chan openvpn.LoadStat, registry metrics.Registry) {
	var clients = metrics.NewGauge()
	var bytesIn = metrics.NewGauge()
	var bytesOut = metrics.NewGauge()
	if registry != nil {
		err := registry.Register("vpn.clients", clients)
		if err != nil {
			return
		}
		err = registry.Register("vpn.bytesIn", bytesIn)
		if err != nil {
			return
		}
		err = registry.Register("vpn.bytesOut", bytesOut)
		if err != nil {
			return
		}
	}
	for loadStat := range lsCh {
		ls = loadStat
		if registry != nil {
			clients.Update(int64(ls.Clients))
			bytesIn.Update(int64(ls.BytesIn))
			bytesOut.Update(int64(ls.BytesOut))
		}
	}
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	if debug {log.Printf("status request from %s\n", r.RemoteAddr)}
	if ls.Clients < 0 {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("OpenVPN Server Unavailable - Cannot connect\n"))
	} else {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(fmt.Sprintf("OpenVPN Healthy %d %d %d\n", ls.Clients, ls.BytesIn, ls.BytesOut)))
	}
}

func main() {
	flag.Parse()
	var r = mux.NewRouter()
	lsCh := make(chan openvpn.LoadStat, 10)
	addr := fmt.Sprintf("%s:%d", miHost, miPort)
	if miSock != "" {
		addr = miSock
	}
	var registry metrics.Registry
	if publish {
		registry = metrics.NewRegistry()
		awsSession := session.Must(session.NewSession())
		metricsConf := &config.Config{
			Client:            cloudwatch.New(awsSession),
			Namespace:         "my-metrics-namespace",
			Filter:            &config.NoFilter{},
			ReportingInterval: metricsInterval,
			StaticDimensions:  map[string]string{"name":"value"},
		}
		go reporter.Cloudwatch(registry, metricsConf)
	}
	go managementConnect(addr, lsCh)
	go handleLoadStatChannel(lsCh, registry)
	r.HandleFunc("/status", statusHandler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", listenPort), r))
}
