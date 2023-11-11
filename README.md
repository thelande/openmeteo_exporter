# Open-Meteo Exporter

A [Prometheus](https://prometheus.io) exporter for exposing metrics from
[Open-Meteo.com](Open-Meteo.com).

## Configuration

The exporter uses a YAML configuration file to specify the locations and metrics
to expose.

```yaml
---
locations:
  - name: San Francisco
    latitude: 37.7577607
    longitude: -122.4787995
    timezone: America/Los_Angeles
    weather:
      variables:
        - temperature_2m
        - relative_humidity_2m
    air_quality:
      variables:
        - pm2_5
        - us_aqi
  - name: New York
    latitude: 40.6976312
    longitude: -74.1444877
    timezone: America/New_York
    weather:
      variables:
        - temperature_2m
        - relative_humidity_2m
    air_quality:
      variables:
        - pm2_5
        - us_aqi
```

Use the `--variables.list` option to list the available variables for either
`weather` or `air_quality`:

```console
$ ./openmeteo_exporter --variables.list=weather
ts=2023-11-11T23:23:02.995Z caller=main.go:62 level=info msg="Starting openmeteo_exporter" version="(version=0.2.2, branch=heads/tags/v0.2.2, revision=1bd3b33b67ec9ba80d67f04a2aea20f2e1dbf3e8)"
ts=2023-11-11T23:23:02.995Z caller=main.go:63 level=info msg="Build context" build_context="(go=go1.21.3, platform=windows/amd64, user=runneradmin@fv-az980-472, date=20231111-23:12:18, tags=netgo osusergo static_build)"
Weather Variables
+----------------------------+----------------------------------------------------------------------------------+
|            NAME            |                                   DESCRIPTION                                    |
+----------------------------+----------------------------------------------------------------------------------+
| cloud_cover                | Total cloud cover as an area fraction                                            |
+----------------------------+----------------------------------------------------------------------------------+
| cloud_cover_high           | High level clouds from 8 km altitude                                             |
+----------------------------+----------------------------------------------------------------------------------+
| wind_gusts_10m             | Gusts at 10 meters above ground as a maximum of the preceding hour               |
+----------------------------+----------------------------------------------------------------------------------+
...
```

## Running

Running the `openmeteo_exporter` command without arguments will cause it to
expose the metrics on port `9812` under `/metrics` and read from `config.yaml`.
Use the command line flags to change these settings as needed:

```console
$ ./openmeteo_exporter-0.2.2-amd64 --help
usage: openmeteo_exporter-0.2.2-amd64 [<flags>]


Flags:
  -h, --[no-]help           Show context-sensitive help (also try --help-long
                            and --help-man).
      --config.file="config.yaml"
                            Path to configuration file.
      --web.telemetry-path="/metrics"
                            Path under which to expose metrics.
      --variables.list=VARIABLES.LIST
                            List the variables available for querying and then
                            exit.
      --web.listen-address=:9812 ...
                            Addresses on which to expose metrics and web
                            interface. Repeatable for multiple addresses.
      --web.config.file=""  [EXPERIMENTAL] Path to configuration file
                            that can enable TLS or authentication. See:
                            https://github.com/prometheus/exporter-toolkit/blob/master/docs/web-configuration.md
      --log.level=info      Only log messages with the given severity or above.
                            One of: [debug, info, warn, error]
      --log.format=logfmt   Output format of log messages. One of: [logfmt,
                            json]
      --[no-]version        Show application version.
```

## Docker

A Docker image of the exporter is available
[here](https://hub.docker.com/r/thelande/openmeteo_exporter) and takes the same
arguments as the command.

```console
docker run -d -v ./config.yaml:/config.yaml:ro -p 9812:9812 \
  thelande/openmeteo_exporter --config.file=/config.yaml
```

## API Limits

The Open-Meteo API is free and does not require an API key for non-commercial
use. However, there are rate limits in place. You may read more about them
[here](https://open-meteo.com/en/terms).
