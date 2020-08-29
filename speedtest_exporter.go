// Copyright (C) 2016, 2017 Nicolas Lamirault <nicolas.lamirault@gmail.com>

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	"github.com/dchest/uniuri"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	prom_version "github.com/prometheus/common/version"

	"github.com/nlamirault/speedtest_exporter/speedtest"
	"github.com/nlamirault/speedtest_exporter/version"
)

const (
	namespace = "speedtest"
)

func init() {
	prometheus.MustRegister(prom_version.NewCollector("speedtest_exporter"))
}

func main() {
	var (
		showVersion   = flag.Bool("version", false, "Print version information.")
		listenAddress = flag.String("web.listen-address", ":9112", "Address to listen on for web interface and telemetry.")
		metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
		configURL     = flag.String("speedtest.config-url", "http://c.speedtest.net/speedtest-config.php?x="+uniuri.New(), "Speedtest configuration URL")
		serverURL     = flag.String("speedtest.server-url", "http://c.speedtest.net/speedtest-servers-static.php?x="+uniuri.New(), "Speedtest server URL")
		//interval      = flag.Int("interval", 60*time.Second, "Interval for metrics.")

		ping = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "ping",
			Help:      "Latency (ms)",
		})
		download = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "download",
			Help:      "Download bandwidth (Mbps).",
		})
		upload = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "upload",
			Help:      "Upload bandwidth (Mbps).",
		})
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("Speedtest Prometheus exporter. v%s\n", version.Version)
		os.Exit(0)
	}

	log.Infoln("Starting speedtest exporter", prom_version.Info())
	log.Infoln("Build context", prom_version.BuildContext())

	interval := 60 * time.Second

	client, err := speedtest.NewClient(*configURL, *serverURL)
	if err != nil {
		log.Errorf("Can't create exporter : %s", err)
		os.Exit(1)
	}
	if client == nil {
		log.Errorf("Speedtest client not configured.")
		os.Exit(1)
	}

	log.Infoln("Register exporter")
	prometheus.MustRegister(ping, upload, download)

	go func() {
		for {
			log.Infof("Speedtest exporter starting")

			metrics, err := client.NetworkMetrics()
			if err != nil {
				log.Errorf("Failed to gather metrics")
				continue
			}

			ping.Set(metrics["ping"])
			download.Set(metrics["download"])
			upload.Set(metrics["upload"])
			log.Infof("Speedtest exporter finished")

			time.Sleep(interval)
		}
	}()

	http.Handle(*metricsPath, prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
             <head><title>Speedtest Exporter</title></head>
             <body>
             <h1>Speedtest Exporter</h1>
             <p><a href='` + *metricsPath + `'>Metrics</a></p>
             </body>
             </html>`))
	})

	log.Infoln("Listening on", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
