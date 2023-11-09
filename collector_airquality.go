package main

import (
	"fmt"
	"strings"

	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
)

type AirQualityCollector struct {
	Client   *OpenMeteoClient
	Location *LocationConfig
}

func (c AirQualityCollector) Collect(ch chan<- prometheus.Metric) {
	airQualityResp, err := c.Client.GetAirQuality(c.Location)
	if err != nil {
		level.Warn(logger).Log(
			"msg", "Failed to collect weather information",
			"location", c.Location.Name,
			"err", err,
		)
		return
	}

	ch <- prometheus.MustNewConstMetric(
		airqualityGenerationTimeDesc,
		prometheus.GaugeValue,
		float64(airQualityResp.GenerationtimeMs),
		c.Location.Name,
	)

	for _, name := range c.Location.AirQuality.Variables {
		units := airQualityResp.CurrentUnits.Variables[name].(string)
		if units == "μg/m³" {
			units = "ug_per_m3"
		} else if units == "Grains/m³" {
			units = "grains_per_m3"
		}
		units = strings.ToLower(units)

		description, _ := GetVariableDesc("airquality", name)
		desc := prometheus.NewDesc(
			prometheus.BuildFQName(namespace, "airquality", fmt.Sprintf("%s_%s", name, units)),
			description,
			[]string{"location"},
			nil,
		)

		if value := airQualityResp.Current.Variables[name]; value != nil {
			ch <- prometheus.MustNewConstMetric(
				desc,
				prometheus.GaugeValue,
				float64(value.(float64)),
				c.Location.Name,
			)
		} else {
			level.Warn(logger).Log("msg", "No value for metric returned", "name", name)
		}
	}
}
