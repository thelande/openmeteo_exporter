package main

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

const namespace = "openmeteo"

var (
	infoDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "location", "info"),
		"Information about the location.",
		[]string{"location", "latitude", "longitude", "timezone"},
		nil,
	)

	weatherGenerationTimeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "weather", "generation_time_ms"),
		"The time it took to generate the response, in milliseconds.",
		[]string{"location"},
		nil,
	)

	airqualityGenerationTimeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "airquality", "generation_time_ms"),
		"The time it took to generate the response, in milliseconds.",
		[]string{"location"},
		nil,
	)
)

type OpenMeteoCollector struct {
	Client    *OpenMeteoClient
	Locations []LocationConfig
}

func (c OpenMeteoCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- infoDesc
	ch <- weatherGenerationTimeDesc
}

func (c OpenMeteoCollector) Collect(ch chan<- prometheus.Metric) {
	for _, loc := range c.Locations {
		ch <- prometheus.MustNewConstMetric(
			infoDesc,
			prometheus.GaugeValue,
			1,
			loc.Name,
			fmt.Sprintf("%f", loc.Latitude),
			fmt.Sprintf("%f", loc.Longitude),
			loc.Timezone,
		)

		if loc.Weather != nil {
			weatherCollector := WeatherCollector{Client: c.Client, Location: &loc}
			weatherCollector.Collect(ch)
		}

		if loc.AirQuality != nil {
			airqualityCollector := AirQualityCollector{Client: c.Client, Location: &loc}
			airqualityCollector.Collect(ch)
		}
	}
}
