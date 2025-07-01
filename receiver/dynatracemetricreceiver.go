package dynatracemetricreceiver

import (
	//	"bufio"
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/receiver"
)

type dynatraceMetricReceiver struct {
	config   *Config
	consumer consumer.Metrics
	server   *http.Server
}

var errEmptyEndpoint = errors.New("empty endpoint")

func newDynatraceMetricReceiver(params receiver.Settings, cfg Config, consumer consumer.Metrics) (receiver.Metrics, error) {
	//return &dynatraceMetricReceiver{config: cfg, consumer: consumer}
	{
		if cfg.Endpoint == "" {
			return nil, errEmptyEndpoint
		}

		r := dynatraceMetricReceiver{
			config:   &cfg,
			consumer: consumer,
		}

		return &r, nil
	}
}

func (r *dynatraceMetricReceiver) Start(ctx context.Context, host component.Host) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics/ingest", r.handle)
	r.server = &http.Server{Addr: r.config.Endpoint, Handler: mux}

	go func() {
		if err := r.server.ListenAndServe(); err != http.ErrServerClosed {
			panic(fmt.Sprintf("listen error: %v", err))
		}
	}()
	return nil
}

func (r *dynatraceMetricReceiver) Shutdown(ctx context.Context) error {
	return r.server.Close()
}

type Result struct {
	Error        string `json:"error"`
	LinesOK      int    `json:"linesOk"`
	LinesInvalid int    `json:"linesInvalid"`
}

func (r *dynatraceMetricReceiver) handle(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "invalid method", http.StatusMethodNotAllowed)
		return
	}
	defer req.Body.Close()

	scanner := bufio.NewScanner(req.Body)

	finalMetrics := pmetric.NewMetrics()
	finalRm := finalMetrics.ResourceMetrics().AppendEmpty()
	finalSm := finalRm.ScopeMetrics().AppendEmpty()

	metricsCount := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		metric := parseLine(line)
		if metric != nil {
			m := ConvertToOtelMetrics(*metric)

			// Merge datapoints into the final collector object
			rms := m.ResourceMetrics()
			for i := 0; i < rms.Len(); i++ {
				ils := rms.At(i).ScopeMetrics()
				for j := 0; j < ils.Len(); j++ {
					metrics := ils.At(j).Metrics()
					for k := 0; k < metrics.Len(); k++ {
						metricCopy := metrics.At(k)
						newMetric := finalSm.Metrics().AppendEmpty()
						metricCopy.CopyTo(newMetric)
						metricsCount++
					}
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		http.Error(w, "read error", http.StatusInternalServerError)
		return
	}

	err := r.consumer.ConsumeMetrics(context.Background(), finalMetrics)
	if err != nil {
		http.Error(w, "failed to consume metrics", http.StatusInternalServerError)
		return
	}

	result := Result{
		Error:        "null",
		LinesOK:      metricsCount,
		LinesInvalid: 0,
	}

	resp, _ := json.Marshal(result)
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Server", "EEC")
	w.WriteHeader(http.StatusAccepted)
	w.Write(resp)
}

/*
func (r *dynatraceMetricReceiver) handle(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "invalid method", http.StatusMethodNotAllowed)
		return
	}
	defer req.Body.Close()

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "read error", http.StatusInternalServerError)
		return
	}

	//parsed := parseLines(string(body))
	// fmt.Printf("+++++++++++++++++++++  %+v\n", parsed)
	// metrics := convertToOtel(parsed)
	var metrics pmetric.Metrics

	scanner := bufio.NewScanner(strings.NewReader(string(body)))
	for scanner.Scan() {
		line := scanner.Text()
		metric := parseLine(line);
		if metric != nil {
			metrics := ConvertToOtelMetrics(metric)
			fmt.Printf("%+v\n", metric)

		}
	}

	err = r.consumer.ConsumeMetrics(context.Background(), metrics)
	if err != nil {
		http.Error(w, "failed to consume metrics", http.StatusInternalServerError)
		return
	}
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Server", "EEC")
	w.WriteHeader(http.StatusAccepted)

	result := Result{
		Error:        "null",
		LinesOK:      metrics.MetricCount(),
		LinesInvalid: 0,
	}
	jsondata, err := json.Marshal(result)
	if err != nil {
		fmt.Println("Error marshalling JSON with indent:", err)
	} else {
		w.Write(jsondata)
	}

	r.consumer.ConsumeMetrics(req.Context(), metrics)
}
*/

/*
func (r *receiver) handle(w http.ResponseWriter, req *http.Request) {
	scanner := bufio.NewScanner(req.Body)
	md := pmetric.NewMetrics()
	rms := md.ResourceMetrics().AppendEmpty()
	sm := rms.ScopeMetrics().AppendEmpty()
	metrics := sm.Metrics()

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		name := parts[0]
		val, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			continue
		}
		m := metrics.AppendEmpty()
		m.SetName(name)
		dp := m.SetEmptyGauge().DataPoints().AppendEmpty()
		dp.SetDoubleValue(val)
	}

	if err := r.consumer.ConsumeMetrics(context.Background(), md); err != nil {
		http.Error(w, "failed to consume metrics", http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusAccepted)
	}
}
*/
