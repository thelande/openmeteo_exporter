/*
Copyright 2023 Thomas Helander

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
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
	metricsPath = kingpin.Flag(
		"web.telemetry-path",
		"Path under which to expose metrics.",
	).Default("/metrics").String()
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
	level.Info(logger).Log("msg", "Build context", "build_context", version.BuildContext())

	// User requested we list the available variables.
	if *listVariables != "" {
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

	landingConfig := web.LandingConfig{
		Name:        "Open-Meteo Exporter",
		Description: "Prometheus Open-Meteo Exporter",
		Version:     version.Info(),
		Links: []web.LandingLinks{
			{
				Address: *metricsPath,
				Text:    "Metrics",
			},
		},
	}
	landingPage, err := web.NewLandingPage(landingConfig)
	if err != nil {
		level.Error(logger).Log("err", err)
		os.Exit(1)
	}

	http.Handle(*metricsPath, promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	http.Handle("/", landingPage)

	srv := &http.Server{}
	if err := web.ListenAndServe(srv, webConfig, logger); err != nil {
		level.Error(logger).Log("msg", "HTTP listener stopped", "error", err)
		os.Exit(1)
	}
}
