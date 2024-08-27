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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/go-kit/log/level"
)

const (
	weatherApi    = "https://api.open-meteo.com/v1/forecast"
	airqualityApi = "https://air-quality-api.open-meteo.com/v1/air-quality"
)

// Mapping of variable name to description. Used to validate the list of
// requests variables as well as provide descriptions for the metrics.
var (
	ErrNon2XXResponse = errors.New("received non-2XX status")
	WeatherVariables  = map[string]string{
		"temperature_2m":             "Air temperature at 2 meters above ground",
		"relative_humidity_2m":       "Relative humidity at 2 meters above ground",
		"dew_point_2m":               "Dew point temperature at 2 meters above ground",
		"apparent_temperature":       "Apparent temperature is the perceived feels-like temperature combining wind chill factor, relative humidity and solar radiation",
		"pressure_msl":               "Atmospheric air pressure reduced to mean sea level (msl)",
		"surface_pressure":           "Atmospheric air pressure at surface",
		"cloud_cover":                "Total cloud cover as an area fraction",
		"cloud_cover_low":            "Low level clouds and fog up to 3 km altitude",
		"cloud_cover_mid":            "Mid level clouds from 3 to 8 km altitude",
		"cloud_cover_high":           "High level clouds from 8 km altitude",
		"wind_speed_10m":             "Wind speed at 10 meters above ground.",
		"wind_speed_80m":             "Wind speed at 80 meters above ground.",
		"wind_speed_120m":            "Wind speed at 120 meters above ground.",
		"wind_speed_180m":            "Wind speed at 180 meters above ground.",
		"wind_direction_10m":         "Wind direction at 10 meters above ground.",
		"wind_direction_80m":         "Wind direction at 80 meters above ground.",
		"wind_direction_120m":        "Wind direction at 120 meters above ground.",
		"wind_direction_180m":        "Wind direction at 180 meters above ground.",
		"wind_gusts_10m":             "Gusts at 10 meters above ground as a maximum of the preceding hour",
		"shortwave_radiation":        "Shortwave solar radiation as average of the preceding hour. This is equal to the total global horizontal irradiation",
		"direct_radiation":           "Direct solar radiation as average of the preceding hour on the horizontal plane",
		"direct_normal_irradiance":   "Direct solar radiation as average of the preceding hour on the normal plane (perpendicular to the sun)",
		"diffuse_radiation":          "Diffuse solar radiation as average of the preceding hour",
		"vapour_pressure_deficit":    "Vapour Pressure Deficit (VPD) in kilopascal (kPa). For high VPD (>1.6), water transpiration of plants increases. For low VPD (<0.4), transpiration decreases",
		"cape":                       "Convective available potential energy",
		"evapotranspiration":         "Evapotranspration from land surface and plants that weather models assumes for this location. Available soil water is considered. 1 mm evapotranspiration per hour equals 1 liter of water per spare meter.",
		"et0_fao_evapotranspiration": "ET₀ Reference Evapotranspiration of a well watered grass field. Based on FAO-56 Penman-Monteith equations ET₀ is calculated from temperature, wind speed, humidity and solar radiation. Unlimited soil water is assumed. ET₀ is commonly used to estimate the required irrigation for plants.",
		"precipitation":              "Total precipitation (rain, showers, snow) sum of the preceding hour",
		"snowfall":                   "Snowfall amount of the preceding hour in centimeters. For the water equivalent in millimeter, divide by 7. E.g. 7 cm snow = 10 mm precipitation water equivalent",
		"precipitation_probability":  "Probability of precipitation with more than 0.1 mm of the preceding hour. Probability is based on ensemble weather models with 0.25° (~27 km) resolution. 30 different simulations are computed to better represent future weather conditions.",
		"rain":                       "Rain from large scale weather systems of the preceding hour in millimeter (or inch)",
		"showers":                    "Showers from convective precipitation in millimeters from the preceding hour",
		"weather_code":               "Weather condition as a numeric code. Follow WMO weather interpretation codes.",
		"snow_depth":                 "Snow depth on the ground in meters",
		"freezing_level_height":      "Altitude above sea level of the 0°C level",
		"visibility":                 "Viewing distance in meters. Influenced by low clouds, humidity and aerosols. Maximum visibility is approximately 24 km.",
		"soil_temperature_0cm":       "Temperature in the soil at 0 cm depth. 0 cm is the surface temperature on land or water surface temperature on water.",
		"soil_temperature_6cm":       "Temperature in the soil at 6 cm depth. 0 cm is the surface temperature on land or water surface temperature on water.",
		"soil_temperature_18cm":      "Temperature in the soil at 18 cm depth. 0 cm is the surface temperature on land or water surface temperature on water.",
		"soil_temperature_54cm":      "Temperature in the soil at 54 cm depth. 0 cm is the surface temperature on land or water surface temperature on water.",
		"soil_moisture_0_to_1cm":     "Average soil water content as volumetric mixing ratio at 0-1 cm depths.",
		"soil_moisture_1_to_3cm":     "Average soil water content as volumetric mixing ratio at 1-3 cm depths.",
		"soil_moisture_3_to_9cm":     "Average soil water content as volumetric mixing ratio at 3-9 cm depths.",
		"soil_moisture_9_to_27cm":    "Average soil water content as volumetric mixing ratio at 9-27 cm depths.",
		"soil_moisture_27_to_81cm":   "Average soil water content as volumetric mixing ratio at 27-81 cm depths.",
		"is_day":                     "1 if the current time step has daylight, 0 at night.",
	}
	AirQualityVariables = map[string]string{
		"pm2_5":                         "Particulate matter with diameter smaller than 2.5 µm (PM2.5) close to surface (10 meter above ground)",
		"pm10":                          "Particulate matter with diameter smaller than 10 µm (PM10) close to surface (10 meter above ground)",
		"carbon_monoxide":               "Carbon monoxide close to surface (10 meter above ground)",
		"nitrogen_dioxide":              "Nitrogen dioxide close to surface (10 meter above ground)",
		"sulphur_dioxide":               "Sulphur dioxide close to surface (10 meter above ground)",
		"ozone":                         "Ozone close to surface (10 meter above ground)",
		"ammonia":                       "Ammonia concentration. Only available for Europe.",
		"aerosol_optical_depth":         "Aerosol optical depth at 550 nm of the entire atmosphere to indicate haze.",
		"dust":                          "Saharan dust particles close to surface level (10 meter above ground).",
		"uv_index":                      "UV index considering clouds. See ECMWF UV Index recommendation for more information",
		"uv_index_clear_sky":            "UV index considering clear sky. See ECMWF UV Index recommendation for more information",
		"alder_pollen":                  "Pollen for various plants. Only available in Europe as provided by CAMS European Air Quality forecast.",
		"birch_pollen":                  "Pollen for various plants. Only available in Europe as provided by CAMS European Air Quality forecast.",
		"grass_pollen":                  "Pollen for various plants. Only available in Europe as provided by CAMS European Air Quality forecast.",
		"mugwort_pollen":                "Pollen for various plants. Only available in Europe as provided by CAMS European Air Quality forecast.",
		"olive_pollen":                  "Pollen for various plants. Only available in Europe as provided by CAMS European Air Quality forecast.",
		"ragweed_pollen":                "Pollen for various plants. Only available in Europe as provided by CAMS European Air Quality forecast.",
		"european_aqi":                  "European Air Quality Index (AQI) calculated for different particulate matter and gases individually. The consolidated european_aqi returns the maximum of all individual indices. Ranges from 0-20 (good), 20-40 (fair), 40-60 (moderate), 60-80 (poor), 80-100 (very poor) and exceeds 100 for extremely poor conditions.",
		"european_aqi_pm2_5":            "European Air Quality Index (AQI) calculated for different particulate matter and gases individually. The consolidated european_aqi returns the maximum of all individual indices. Ranges from 0-20 (good), 20-40 (fair), 40-60 (moderate), 60-80 (poor), 80-100 (very poor) and exceeds 100 for extremely poor conditions.",
		"european_aqi_pm10":             "European Air Quality Index (AQI) calculated for different particulate matter and gases individually. The consolidated european_aqi returns the maximum of all individual indices. Ranges from 0-20 (good), 20-40 (fair), 40-60 (moderate), 60-80 (poor), 80-100 (very poor) and exceeds 100 for extremely poor conditions.",
		"european_aqi_nitrogen_dioxide": "European Air Quality Index (AQI) calculated for different particulate matter and gases individually. The consolidated european_aqi returns the maximum of all individual indices. Ranges from 0-20 (good), 20-40 (fair), 40-60 (moderate), 60-80 (poor), 80-100 (very poor) and exceeds 100 for extremely poor conditions.",
		"european_aqi_ozone":            "European Air Quality Index (AQI) calculated for different particulate matter and gases individually. The consolidated european_aqi returns the maximum of all individual indices. Ranges from 0-20 (good), 20-40 (fair), 40-60 (moderate), 60-80 (poor), 80-100 (very poor) and exceeds 100 for extremely poor conditions.",
		"european_aqi_sulphur_dioxide":  "European Air Quality Index (AQI) calculated for different particulate matter and gases individually. The consolidated european_aqi returns the maximum of all individual indices. Ranges from 0-20 (good), 20-40 (fair), 40-60 (moderate), 60-80 (poor), 80-100 (very poor) and exceeds 100 for extremely poor conditions.",
		"us_aqi":                        "United States Air Quality Index (AQI) calculated for different particulate matter and gases individually. The consolidated us_aqi returns the maximum of all individual indices. Ranges from 0-50 (good), 51-100 (moderate), 101-150 (unhealthy for sensitive groups), 151-200 (unhealthy), 201-300 (very unhealthy) and 301-500 (hazardous).",
		"us_aqi_pm2_5":                  "United States Air Quality Index (AQI) calculated for different particulate matter and gases individually. The consolidated us_aqi returns the maximum of all individual indices. Ranges from 0-50 (good), 51-100 (moderate), 101-150 (unhealthy for sensitive groups), 151-200 (unhealthy), 201-300 (very unhealthy) and 301-500 (hazardous).",
		"us_aqi_pm10":                   "United States Air Quality Index (AQI) calculated for different particulate matter and gases individually. The consolidated us_aqi returns the maximum of all individual indices. Ranges from 0-50 (good), 51-100 (moderate), 101-150 (unhealthy for sensitive groups), 151-200 (unhealthy), 201-300 (very unhealthy) and 301-500 (hazardous).",
		"us_aqi_nitrogen_dioxide":       "United States Air Quality Index (AQI) calculated for different particulate matter and gases individually. The consolidated us_aqi returns the maximum of all individual indices. Ranges from 0-50 (good), 51-100 (moderate), 101-150 (unhealthy for sensitive groups), 151-200 (unhealthy), 201-300 (very unhealthy) and 301-500 (hazardous).",
		"us_aqi_ozone":                  "United States Air Quality Index (AQI) calculated for different particulate matter and gases individually. The consolidated us_aqi returns the maximum of all individual indices. Ranges from 0-50 (good), 51-100 (moderate), 101-150 (unhealthy for sensitive groups), 151-200 (unhealthy), 201-300 (very unhealthy) and 301-500 (hazardous).",
		"us_aqi_sulphur_dioxide":        "United States Air Quality Index (AQI) calculated for different particulate matter and gases individually. The consolidated us_aqi returns the maximum of all individual indices. Ranges from 0-50 (good), 51-100 (moderate), 101-150 (unhealthy for sensitive groups), 151-200 (unhealthy), 201-300 (very unhealthy) and 301-500 (hazardous).",
		"us_aqi_carbon_monoxide":        "United States Air Quality Index (AQI) calculated for different particulate matter and gases individually. The consolidated us_aqi returns the maximum of all individual indices. Ranges from 0-50 (good), 51-100 (moderate), 101-150 (unhealthy for sensitive groups), 151-200 (unhealthy), 201-300 (very unhealthy) and 301-500 (hazardous).",
	}
	ValidTemperatureUnits   = []string{"fahrenheit", "celsius"}
	ValidWindSpeedUnits     = []string{"kmh", "mph", "ms", "kn"}
	ValidPrecipitationUnits = []string{"mm", "inch"}
)

type ResponseUnits struct {
	Time      string `json:"time"`
	Interval  string `json:"interval"`
	Variables map[string]interface{}
}

type ResponseValues struct {
	Time      string `json:"time"`
	Interval  int    `json:"interval"`
	Variables map[string]interface{}
}

type BaseResponse struct {
	Latitude             float64        `json:"latitude"`
	Longitude            float64        `json:"longitude"`
	GenerationtimeMs     float32        `json:"generationtime_ms"`
	UTCOffsetSeconds     int            `json:"utc_offset_seconds"`
	Timezone             string         `json:"timezone"`
	TimezoneAbbreviation string         `json:"timezone_abbreviation"`
	CurrentUnits         ResponseUnits  `json:"current_units"`
	Current              ResponseValues `json:"current"`
}

type WeatherResponse struct {
	BaseResponse
	Elevation float64 `json:"elevation"`
}

func GetVariableDesc(category, name string) (string, error) {
	var val string
	var ok bool
	if category == "weather" {
		val, ok = WeatherVariables[name]
	} else {
		val, ok = AirQualityVariables[name]
	}

	if !ok {
		return "", fmt.Errorf("invalid variable name: %s", name)
	}
	return val, nil
}

func IsValidVariable(category, name string) bool {
	if _, err := GetVariableDesc(category, name); err != nil {
		return false
	}
	return true
}

type OpenMeteoClient struct{}

func (c OpenMeteoClient) doRequest(fullUrl string, values *url.Values) ([]byte, error) {
	level.Debug(logger).Log("url", fullUrl)
	resp, err := http.Get(fullUrl)
	if err != nil {
		level.Error(logger).Log("msg", "Failed to query open-meteo API", "err", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		level.Error(logger).Log("msg", "Failed to read response body", "err", err)
		return nil, err
	}

	if resp.StatusCode >= 400 {
		level.Warn(logger).Log("status", resp.Status, "statusCode", resp.StatusCode, "body", string(body))
		return nil, ErrNon2XXResponse
	}

	return body, nil
}

func buildBaseValues(loc *LocationConfig, vars []string) *url.Values {
	values := &url.Values{}
	values.Add("latitude", fmt.Sprintf("%f", loc.Latitude))
	values.Add("longitude", fmt.Sprintf("%f", loc.Longitude))

	var current []string
	current = append(current, vars...)

	values.Add("current", strings.Join(current, ","))

	return values
}

func (c OpenMeteoClient) GetWeather(l *LocationConfig) (*WeatherResponse, error) {
	url, err := url.Parse(weatherApi)
	if err != nil {
		level.Error(logger).Log("msg", "Failed to form response URL", "err", err)
		return nil, err
	}

	values := buildBaseValues(l, l.Weather.Variables)
	values.Add("timezone", l.Timezone)
	values.Add("temperature_unit", l.Weather.TemperatureUnit)
	values.Add("wind_speed_unit", l.Weather.WindSpeedUnit)
	values.Add("precipitation_unit", l.Weather.PrecipitationUnit)
	url.RawQuery = values.Encode()

	body, err := c.doRequest(url.String(), values)
	if err != nil {
		return nil, err
	}

	level.Debug(logger).Log("body", string(body))

	var bareResp map[string]interface{}
	if err = json.Unmarshal(body, &bareResp); err != nil {
		return nil, err
	}

	resp := WeatherResponse{}
	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	resp.Current.Variables = make(map[string]interface{})
	resp.CurrentUnits.Variables = make(map[string]interface{})

	omitValues := []string{"time", "interval"}
	for name, value := range bareResp["current"].(map[string]interface{}) {
		if slices.Contains(omitValues, name) {
			continue
		}

		resp.Current.Variables[name] = value
		resp.CurrentUnits.Variables[name] = bareResp["current_units"].(map[string]interface{})[name]
	}

	return &resp, nil
}

func (c OpenMeteoClient) GetAirQuality(l *LocationConfig) (*BaseResponse, error) {
	url, err := url.Parse(airqualityApi)
	if err != nil {
		level.Error(logger).Log("msg", "Failed to form response URL", "err", err)
		return nil, err
	}
	values := buildBaseValues(l, l.AirQuality.Variables)
	url.RawQuery = values.Encode()

	body, err := c.doRequest(url.String(), values)
	if err != nil {
		return nil, err
	}

	level.Debug(logger).Log("body", string(body))

	var bareResp map[string]interface{}
	if err = json.Unmarshal(body, &bareResp); err != nil {
		return nil, err
	}

	resp := BaseResponse{}
	if err = json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}

	resp.Current.Variables = make(map[string]interface{})
	resp.CurrentUnits.Variables = make(map[string]interface{})

	omitValues := []string{"time", "interval"}
	for name, value := range bareResp["current"].(map[string]interface{}) {
		if slices.Contains(omitValues, name) {
			continue
		}

		resp.Current.Variables[name] = value
		resp.CurrentUnits.Variables[name] = bareResp["current_units"].(map[string]interface{})[name]
	}

	return &resp, nil
}
