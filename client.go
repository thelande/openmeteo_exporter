package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/go-kit/log/level"
)

const baseApi = "https://api.open-meteo.com/v1"

// Mapping of variable name to description. Used to validate the list of
// requests variables as well as provide descriptions for the metrics.
var (
	weatherVariables = map[string]string{
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
	ValidTemperatureUnits   = []string{"fahrenheit", "celsius"}
	ValidWindSpeedUnits     = []string{"kph", "mph", "ms", "kn"}
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

func GetVariableDesc(name string) (string, error) {
	val, ok := weatherVariables[name]
	if !ok {
		return "", fmt.Errorf("invalid variable name: %s", name)
	}
	return val, nil
}

func IsValidVariable(name string) bool {
	if _, err := GetVariableDesc(name); err != nil {
		return false
	}
	return true
}

type OpenMeteoClient struct{}

func (c OpenMeteoClient) doRequest(path string, values *url.Values) ([]byte, error) {
	url, err := url.Parse(fmt.Sprintf("%s/%s", baseApi, path))
	if err != nil {
		level.Error(logger).Log("msg", "Failed to form response URL", "err", err)
		return nil, err
	}
	url.RawQuery = values.Encode()

	fullUrl := url.String()
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

	return body, nil
}

func (c OpenMeteoClient) GetWeather(l *LocationConfig) (*WeatherResponse, error) {
	path := "forecast"
	values := &url.Values{}

	values.Add("latitude", fmt.Sprintf("%f", l.Latitude))
	values.Add("longitude", fmt.Sprintf("%f", l.Longitude))
	values.Add("timezone", l.Timezone)
	values.Add("temperature_unit", l.Weather.TemperatureUnit)
	values.Add("wind_speed_unit", l.Weather.WindSpeedUnit)
	values.Add("precipitation_unit", l.Weather.PrecipitationUnit)

	var current []string
	current = append(current, l.Weather.Variables...)

	values.Add("current", strings.Join(current, ","))

	body, err := c.doRequest(path, values)
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

func (c OpenMeteoClient) GetAirQuality() error {
	return nil
}
