// Copyright 2020, OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build windows

package windows

import (
	"time"

	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
)

type ValueTransformationFn func(val float64) float64

type PerfCounterDescriptor struct {
	Name            string
	Path            string
	ScrapeOnStartup bool
}

type PerfCounter struct {
	descriptor *PerfCounterDescriptor
	query      PerformanceQuery
	handle     PDH_HCOUNTER
}

// NewPerfCounter returns a new performance counter for the specified descriptor.
func NewPerfCounter(descriptor *PerfCounterDescriptor) (*PerfCounter, error) {
	query := &PerformanceQueryImpl{}
	err := query.Open()
	if err != nil {
		return nil, err
	}

	var handle PDH_HCOUNTER
	handle, err = query.AddEnglishCounterToQuery(descriptor.Path)
	if err != nil {
		return nil, err
	}

	// some perf counters (e.g. cpu) return the usage stats since the last measure
	// so we need to collect data on startup to avoid an invalid initial reading
	if descriptor.ScrapeOnStartup {
		err = query.CollectData()
		if err != nil {
			return nil, err
		}
	}

	counter := &PerfCounter{
		descriptor: descriptor,
		query:      query,
		handle:     handle,
	}

	return counter, nil
}

// Close all counters/handles related to the query and free all associated memory.
func (pc *PerfCounter) Close() error {
	return pc.query.Close()
}

// ScrapeData takes a measure of the performance counter and returns the value(s).
func (pc *PerfCounter) ScrapeData() ([]CounterValue, error) {
	var vals []CounterValue

	err := pc.query.CollectData()
	if err != nil {
		return nil, err
	}

	vals, err = pc.query.GetFormattedCounterArrayDouble(pc.handle)
	if err != nil {
		return nil, err
	}

	return vals, nil
}

// InitializeMetricFromPerfCounterData initializes the provided metric with
// the measurement data from a Performance Counter measurement.
//
// Options are provided to configure the name of the label the performance
// counter's "instanceName" value will be recorded against and to transform
// the returned values.
func InitializeMetricFromPerfCounterData(
	metric pdata.Metric,
	vals []CounterValue,
	instanceNameLabel string,
	startTime time.Time,
	transformFn ValueTransformationFn,
) pdata.Metric {
	vals = removeTotalUnlessOnlyOneValue(vals)

	ddps := metric.DoubleDataPoints()
	ddps.Resize(len(vals))

	for i, val := range vals {
		ddp := ddps.At(i)

		value := val.Value
		if transformFn != nil {
			value = transformFn(val.Value)
		}

		labels := map[string]string{}
		if len(vals) > 1 || (val.InstanceName != "" && val.InstanceName != "_Total") {
			labels[instanceNameLabel] = val.InstanceName
		}
		ddp.LabelsMap().InitFromMap(labels)
		ddp.SetStartTime(pdata.TimestampUnixNano(uint64(startTime.UnixNano())))
		ddp.SetTimestamp(pdata.TimestampUnixNano(uint64(time.Now().UnixNano())))
		ddp.SetValue(value)
	}

	return metric
}

func removeTotalUnlessOnlyOneValue(vals []CounterValue) []CounterValue {
	if len(vals) <= 1 {
		return vals
	}

	for i, val := range vals {
		if val.InstanceName == "_Total" {
			vals[i] = vals[len(vals)-1]
			vals[len(vals)-1] = CounterValue{}
			return vals[:len(vals)-1]
		}
	}

	return vals
}
