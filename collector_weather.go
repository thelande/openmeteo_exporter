package main

import (
	"fmt"

	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

type WeatherCollector struct {
	Client   *OpenMeteoClient
	Location *LocationConfig
}

func (c WeatherCollector) Collect(ch chan<- prometheus.Metric) {
	weatherResp, err := c.Client.GetWeather(c.Location)
	if err != nil {
		level.Warn(logger).Log(
			"msg", "Failed to collect weather information",
			"location", c.Location.Name,
			"err", err,
		)
		return
	}

	ch <- prometheus.MustNewConstMetric(
		weatherGenerationTimeDesc,
		prometheus.GaugeValue,
		float64(weatherResp.GenerationtimeMs),
		c.Location.Name,
	)

	for _, name := range c.Location.Weather.Variables {
		units := weatherResp.CurrentUnits.Variables[name]
		if units == "°F" {
			units = "fahrenheit"
		} else if units == "°C" {
			units = "celsius"
		} else if units == "%" {
			units = "percent"
		}

		description, _ := GetVariableDesc("weather", name)
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
			c.Location.Name,
		)
	}
}
