# Dynatrace Metric Receiver
Implementation of an Otel Metric Receiver for the Dynatrace Metric ingest protocol.



## Embedding the Dynatrace Metric Receiver into an OpenTelemetry Collector

The Dynatrace Metric Receiver is currently not yet included by default in any distribution of the OpenTelemetry Collector.

You need to follow the steps for [building a custom collector](https://opentelemetry.io/docs/collector/custom-collector/). The example below represents a valid `builder-config.yaml` that includes the Dynatrace Processor.


For x86:
```
curl --proto '=https' --tlsv1.2 -fL -o ocb \
https://github.com/open-telemetry/opentelemetry-collector-releases/releases/download/cmd%2Fbuilder%2Fv0.128.0/ocb_0.128.0_linux_amd64
chmod +x ocb
```
For arm64:
```
curl --proto '=https' --tlsv1.2 -fL -o ocb \
https://github.com/open-telemetry/opentelemetry-collector-releases/releases/download/cmd%2Fbuilder%2Fv0.128ls.0/ocb_0.128.0_linux_arm64
chmod +x ocb
```
To Build:
```
 ./ocb --config builder-config.yaml 
```

builder-config.yaml:
```yaml
dist:
  name: otelcol-dev
  description: Basic OTel Collector distribution that includes the Dynatrace Processor
  output_path: ./otelcol-dev
  otelcol_version: 0.128.0

exporters:
  - gomod: go.opentelemetry.io/collector/exporter/debugexporter v0.128.0
  - gomod: go.opentelemetry.io/collector/exporter/otlpexporter v0.128.0
  - gomod: go.opentelemetry.io/collector/exporter/otlphttpexporter v0.128.0

processors:
  - gomod: go.opentelemetry.io/collector/processor/batchprocessor v0.128.0
  - gomod: github.com/Reinhard-Pilz-Dynatrace/dynatraceprocessor v0.128.3

receivers:
  - gomod: go.opentelemetry.io/collector/receiver/otlpreceiver v0.128.0
  - gomod: github.com/open-telemetry/opentelemetry-collector-contrib/receiver/filelogreceiver v0.128.0

providers:
  - gomod: go.opentelemetry.io/collector/confmap/provider/envprovider v1.18.0
  - gomod: go.opentelemetry.io/collector/confmap/provider/fileprovider v1.18.0
  - gomod: go.opentelemetry.io/collector/confmap/provider/httpprovider v1.18.0
  - gomod: go.opentelemetry.io/collector/confmap/provider/httpsprovider v1.18.0
  - gomod: go.opentelemetry.io/collector/confmap/provider/yamlprovider v1.18.0
```

## Configuration

```yaml
receivers:

  dynatracemetric:
        endpoint: 127.0.0.1:14499
```

The example below of a valid `collector-config.yaml` shows how to configure an OpenTelemetry Collector to
* Recieve metrics on port 14499
* Print out the metric Signals on stdout
* Send off the collected metrics to Dynatrace
  - The configured `Api-Token` needs to contain the permissions `ingest.logs`

```yaml
receivers:
  dynatracemetric:
        endpoint: 127.0.0.1:14499
processors:
  batch:
  resourcedetection/dynatrace:
    override: false
    detectors: [dynatrace]

exporters:
  debug:
    verbosity: detailed
  otlphttp:
    endpoint: "https://#######.live.dynatrace.com/api/v2/otlp"
    headers:
      Authorization: "Api-Token dt0c01.#####.##########"

service:
  pipelines:
    metrics:
      receivers: [dynatracemetric]
      processors: [resourcedetection/dynatrace]
      exporters: [otlphttp,debug]
  telemetry:
    metrics:
      level: none

```

To run newly build collector:

```
otelcol-dev/otelcol-dev --config collector-config.yaml
```

