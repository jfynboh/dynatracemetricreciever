package dynatracemetricreceiver

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

type Metric struct {
	Name       string
	Dimensions map[string]string
	Value      *float64
	Summary    map[string]float64
	Type       string
	Timestamp  *int64
}

func parseLine(line string) *Metric {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return nil
	}

	parts := strings.Fields(line)
	if len(parts) < 2 {
		fmt.Printf("Invalid line (too few parts): %s\n", line)
		return nil
	}

	metricPart := parts[0]
	metricName, dimensions := parseMetricAndDimensions(metricPart)

	var (
		value     *float64
		summary   map[string]float64
		mType     string
		timestamp *int64
	)

	typeOrValue := parts[1]
	tokens := strings.Split(typeOrValue, ",")

	if len(tokens) == 1 {
		if isSimpleNumber(tokens[0]) {
			v, _ := strconv.ParseFloat(tokens[0], 64)
			value = &v
		} else {
			mType = tokens[0]
			if len(parts) > 2 && isSimpleNumber(parts[2]) {
				v, _ := strconv.ParseFloat(parts[2], 64)
				value = &v
			}
		}
	} else {
		mType = tokens[0]
		if len(tokens) == 2 && isSimpleNumber(tokens[1]) {
			v, _ := strconv.ParseFloat(tokens[1], 64)
			value = &v
		} else if mType == "count" {
			for _, pair := range tokens[1:] {
				kv := strings.SplitN(pair, "=", 2)
				if len(kv) == 2 && kv[0] == "delta" {
					if f, err := strconv.ParseFloat(kv[1], 64); err == nil {
						value = &f
					}
				}
			}
		} else {
			summary = make(map[string]float64)
			for _, pair := range tokens[1:] {
				kv := strings.SplitN(pair, "=", 2)
				if len(kv) == 2 {
					if f, err := strconv.ParseFloat(kv[1], 64); err == nil {
						summary[kv[0]] = f
					}
				}
			}
		}
	}

	if len(parts) >= 3 {
		last := parts[len(parts)-1]
		if ts, err := strconv.ParseInt(last, 10, 64); err == nil {
			timestamp = &ts
		}
	}

	if timestamp == nil {
		now := time.Now().UnixNano() / int64(time.Nanosecond)
		timestamp = &now
	}

	return &Metric{
		Name:       metricName,
		Dimensions: dimensions,
		Value:      value,
		Summary:    summary,
		Type:       mType,
		Timestamp:  timestamp,
	}
}

func parseMetricAndDimensions(s string) (string, map[string]string) {
	dims := make(map[string]string)

	// Split the string into segments by comma
	segments := strings.Split(s, ",")

	// First segment is the metric name
	metricName := segments[0]

	// Remaining segments are dimension key-value pairs
	for _, segment := range segments[1:] {
		if kv := strings.SplitN(segment, "=", 2); len(kv) == 2 {
			key := kv[0]
			val := unescape(kv[1])
			dims[key] = val
		}
	}

	return metricName, dims
}

func unescape(s string) string {
	if strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\"") {
		s = s[1 : len(s)-1]
	}
	s = strings.ReplaceAll(s, `\\`, `\`)
	s = strings.ReplaceAll(s, `\"`, `"`)
	return s
}

func isSimpleNumber(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func applyAttributes(attrMap pcommon.Map, dims map[string]string) {
	for k, v := range dims {
		attrMap.PutStr(k, v)
	}
}

// ConvertToOtelMetrics converts your parsed Metric struct to OpenTelemetry Metrics pdata.Metrics
func ConvertToOtelMetrics(m Metric) pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	rm := metrics.ResourceMetrics().AppendEmpty()
	ilm := rm.ScopeMetrics().AppendEmpty()

	metric := ilm.Metrics().AppendEmpty()
	metric.SetName(m.Name)

	// Set timestamp or use current time
	var ts pcommon.Timestamp
	if m.Timestamp != nil {
		ts = pcommon.Timestamp(*m.Timestamp)
	} else {
		ts = pcommon.Timestamp(time.Now().UnixNano())
	}

	// attrs := pmetric.NewAttributeMap()
	// for k, v := range m.Dimensions {
	// 	attrs.InsertString(k, v)
	// }

	switch m.Type {
	case "gauge":
		gauge := metric.SetEmptyGauge()
		if len(m.Summary) > 0 {
			// Summary stats map to a single DataPoint with exemplar summary info in attributes
			dp := gauge.DataPoints().AppendEmpty()
			dp.SetTimestamp(ts)

			//dp.SetDoubleVal(0) // no single value

			dpAttrs := dp.Attributes()
			for k, v := range m.Summary {
				dpAttrs.PutStr(k, fmt.Sprintf("%g", v))
			}
			for k, v := range m.Dimensions {
				dpAttrs.PutStr(k, v)
			}
		} else if m.Value != nil {
			dp := gauge.DataPoints().AppendEmpty()
			dp.SetTimestamp(ts)
			dp.SetStartTimestamp(ts)
			dp.SetDoubleValue(*m.Value)
			applyAttributes(dp.Attributes(), m.Dimensions)
		}
	case "count":
		metric.SetEmptySum()
		sum := metric.Sum()
		sum.SetAggregationTemporality(pmetric.AggregationTemporalityDelta)
		sum.SetIsMonotonic(true)

		dp := sum.DataPoints().AppendEmpty()
		dp.SetTimestamp(ts)
		if delta, ok := m.Summary["delta"]; ok {
			dp.SetDoubleValue(delta)
		} else if m.Value != nil {
			dp.SetDoubleValue(*m.Value)
		} else {
			dp.SetDoubleValue(0)
		}
		applyAttributes(dp.Attributes(), m.Dimensions)

	default:
		// Default: treat as gauge with single value
		metric.SetEmptyGauge()
		gauge := metric.Gauge()
		if m.Value != nil {
			dp := gauge.DataPoints().AppendEmpty()
			dp.SetTimestamp(ts)
			dp.SetStartTimestamp(ts)
			dp.SetDoubleValue(*m.Value)
			applyAttributes(dp.Attributes(), m.Dimensions)
		}
	}

	return metrics
}

// -------------------------
// Example usage
// -------------------------

// func main() {
// 	testData := `workHours,team="devops\\bugfixing",project="\"product\"_improvement" 1000
// mymetric,team=teamA,businessapp=hr 1000
// mymetric,team=teamA,businessapp=hr 1000 1609459200000
// cpu.temperature,hostname=hostA,cpu=1 55
// cpu.temperature,hostname=hostA,cpu=2 45
// cpu.temperature,hostname=hostA,cpu=1 gauge,45
// cpu.temperature,hostname=hostA,cpu=1 gauge,min=17.1,max=17.3,sum=34.4,count=2
// cpu.temperature,dt.entity.host=HOST-4587AE40F95AD90D,cpu=1 gauge,min=17.1,max=17.3,sum=34.4,count=2
// new_user_count,region=EAST count,delta=50
// new_user_count,region=WEST count,delta=150
// cpu.temperature gauge,45
// #cpu.temperature gauge dt.meta.unit=count,dt.meta.description="The temperature of the CPU",dt.meta.displayname="CPU temperature"`

// 	scanner := bufio.NewScanner(strings.NewReader(testData))
// 	for scanner.Scan() {
// 		line := scanner.Text()
// 		if metric := parseLine(line); metric != nil {
// 			fmt.Printf("%+v\n", metric)
// 		}
// 	}
// }
