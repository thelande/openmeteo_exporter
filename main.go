package main

import (
	"fmt"
	"net/http"
	"os"

	kingpin "github.com/alecthomas/kingpin/v2"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/olekukonko/tablewriter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"github.com/prometheus/exporter-toolkit/web"
	webflag "github.com/prometheus/exporter-toolkit/web/kingpinflag"
)

var (
	configFile = kingpin.Flag(
		"config.file",
		"Path to configuration file.",
	).Default("config.yaml").String()
	listVariables = kingpin.Flag(
		"variables.list",
		"List the variables available for querying and then exit.",
	).Enum("weather", "airquality")
	webConfig = webflag.AddFlags(kingpin.CommandLine, ":9812")
	logger    log.Logger
)

func main() {
	promlogConfig := &promlog.Config{}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	kingpin.CommandLine.UsageWriter(os.Stdout)
	kingpin.HelpFlag.Short('h')
	kingpin.Version(version.Print("openmeteo_exporter"))
	kingpin.Parse()

	logger = promlog.New(promlogConfig)
	level.Info(logger).Log("msg", "Starting openmeteo_exporter", "version", version.Info())

	// User requested we list the available variables.
	if listVariables != nil {
		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Name", "Description"})
		table.SetRowLine(true)
		table.SetColWidth(80)

		var vars map[string]string
		var title string
		if *listVariables == "weather" {
			title = "Weather Variables"
			vars = WeatherVariables
		} else {
			title = "Air Quality Variables"
			vars = AirQualityVariables
		}

		fmt.Println(title)
		for name, desc := range vars {
			table.Append([]string{name, desc})
		}
		table.Render()

		os.Exit(0)
	}

	var config Config
	if err := config.ReloadConfig(*configFile); err != nil {
		level.Error(logger).Log("msg", "Failed to load configuration", "err", err)
		os.Exit(1)
	}

	collector := OpenMeteoCollector{Client: &OpenMeteoClient{}, Locations: config.Locations}

	// Use a custom handler to avoid generating the go_collector metrics.
	registry := prometheus.NewRegistry()
	registry.MustRegister(collector)
	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>
            <head>
            <title>Open-Meteo Exporter</title>
            <style>
            label{
            display:inline-block;
            width:75px;
            }
            form label {
            margin: 10px;
            }
            form input {
            margin: 10px;
            }
            </style>
            </head>
            <body>
            <h1>Open-Meteo Exporter</h1>
			<p><a href="/metrics">Metrics</a></p>
            </body>
            </html>`))
	})

	srv := &http.Server{}
	if err := web.ListenAndServe(srv, webConfig, logger); err != nil {
		level.Error(logger).Log("msg", "HTTP listener stopped", "error", err)
		os.Exit(1)
	}
}
