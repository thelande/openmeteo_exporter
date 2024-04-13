/*
Copyright 2023-2024 Thomas Helander

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
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/go-kit/log/level"
	"gopkg.in/yaml.v3"
)

const (
	defaultTemperatureUnit   = "fahrenheit"
	defaultWindSpeedUnit     = "mph"
	defaultPrecipitationUnit = "inch"
)

type AirQualityConfig struct {
	Variables []string `yaml:"variables"`
}

type WeatherConfig struct {
	TemperatureUnit   string   `yaml:"temperature_unit"`
	WindSpeedUnit     string   `yaml:"wind_speed_unit"`
	PrecipitationUnit string   `yaml:"precipitation_unit"`
	Variables         []string `yaml:"variables"`
}

type LocationConfig struct {
	Name       string            `yaml:"name"`
	Latitude   float64           `yaml:"latitude"`
	Longitude  float64           `yaml:"longitude"`
	Timezone   string            `yaml:"timezone"`
	Weather    *WeatherConfig    `yaml:"weather"`
	AirQuality *AirQualityConfig `yaml:"air_quality"`
}

type Config struct {
	Locations []LocationConfig `yaml:"locations"`
}

func (c *Config) ReloadConfig(configFile string) error {
	var config []byte
	var err error

	if configFile != "" {
		config, err = os.ReadFile(configFile)
		if err != nil {
			level.Error(logger).Log("msg", "Failed to read config file", "path", configFile, "err", err)
			return err
		}
	} else {
		return errors.New("no configuration file specified")
	}

	if err = yaml.Unmarshal(config, c); err != nil {
		return err
	}

	if err = c.Validate(); err != nil {
		return err
	}

	level.Info(logger).Log("msg", "Loaded configuration file", "path", configFile, "locations", len(c.Locations))
	return nil
}

func (c *Config) Validate() error {
	if c.Locations == nil || len(c.Locations) == 0 {
		return errors.New("invalid config, no locations provided")
	}

	for _, loc := range c.Locations {
		if err := loc.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (l *LocationConfig) Validate() error {
	if len(l.Name) == 0 {
		return errors.New("invalid location, no name provided")
	}

	if l.Latitude == 0 {
		return fmt.Errorf("invalid location, no latitude provided: %s", l.Name)
	}

	if l.Longitude == 0 {
		return fmt.Errorf("invalid location, no longitude provided: %s", l.Name)
	}

	// Use auto if no timezone is set to enable automatic detection based on
	// the location: https://open-meteo.com/en/docs#api-documentation
	if len(l.Timezone) == 0 {
		l.Timezone = "auto"
	}

	if l.Weather != nil {
		if err := l.Weather.Validate(l); err != nil {
			return err
		}
	}
	if l.AirQuality != nil {
		if err := l.AirQuality.Validate(l); err != nil {
			return err
		}
	}
	if l.Weather == nil && l.AirQuality == nil {
		return fmt.Errorf("invalid location, no weather or air_quality sections defined: %s", l.Name)
	}

	return nil
}

func (w *WeatherConfig) Validate(l *LocationConfig) error {
	if len(w.Variables) == 0 {
		return fmt.Errorf("invalid weather config, no entries for variables: %s", l.Name)
	}

	for _, name := range w.Variables {
		if !IsValidVariable("weather", name) {
			return fmt.Errorf("invalid current weather variable, %s, for location: %s", name, l.Name)
		}
	}

	if len(w.TemperatureUnit) == 0 {
		w.TemperatureUnit = defaultTemperatureUnit
	}

	if !slices.Contains(ValidTemperatureUnits, w.TemperatureUnit) {
		return fmt.Errorf("invalid temperature_unit, %s, for location: %s", w.TemperatureUnit, l.Name)
	}

	if len(w.WindSpeedUnit) == 0 {
		w.WindSpeedUnit = defaultWindSpeedUnit
	}

	if !slices.Contains(ValidWindSpeedUnits, w.WindSpeedUnit) {
		return fmt.Errorf("invalid wind_speed_unit, %s, for location: %s", w.WindSpeedUnit, l.Name)
	}

	if len(w.PrecipitationUnit) == 0 {
		w.PrecipitationUnit = defaultPrecipitationUnit
	}

	if !slices.Contains(ValidPrecipitationUnits, w.PrecipitationUnit) {
		return fmt.Errorf("invalid precipitation_unit, %s, for location: %s", w.PrecipitationUnit, l.Name)
	}

	return nil
}

func (a *AirQualityConfig) Validate(l *LocationConfig) error {
	if len(a.Variables) == 0 {
		return fmt.Errorf("invalid air quality config, no entries for variables: %s", l.Name)
	}

	for _, name := range a.Variables {
		if !IsValidVariable("airquality", name) {
			return fmt.Errorf("invalid current air quality variable, %s, for location: %s", name, l.Name)
		}
	}

	return nil
}
