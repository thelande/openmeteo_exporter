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

		if value := weatherResp.Current.Variables[name]; value != nil {
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
