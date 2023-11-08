package main

import (
	"fmt"

	"github.com/go-kit/log/level"
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

	generationTimeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "weather", "generation_time_ms"),
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
	ch <- generationTimeDesc
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

		weatherResp, err := c.Client.GetWeather(&loc)
		if err != nil {
			level.Warn(logger).Log("msg", "Failed to collect weather information", "location", loc.Name, "err", err)
			continue
		}

		ch <- prometheus.MustNewConstMetric(
			generationTimeDesc,
			prometheus.GaugeValue,
			float64(weatherResp.GenerationtimeMs),
			loc.Name,
		)

		for _, name := range loc.Weather.Variables {
			units := weatherResp.CurrentUnits.Variables[name]
			if units == "°F" {
				units = "fahrenheit"
			} else if units == "°C" {
				units = "celsius"
			} else if units == "%" {
				units = "percent"
			}

			description, _ := GetVariableDesc(name)
			desc := prometheus.NewDesc(
				prometheus.BuildFQName(namespace, "weather", fmt.Sprintf("%s_%s", name, units)),
				description,
				[]string{"location"},
				nil,
			)

			ch <- prometheus.MustNewConstMetric(
				desc,
				prometheus.GaugeValue,
				float64(weatherResp.Current.Variables[name].(float64)),
				loc.Name,
			)
		}
	}
}
