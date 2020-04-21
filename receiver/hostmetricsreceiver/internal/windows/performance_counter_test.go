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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
)

func TestNewPerfCounter_InvalidPath(t *testing.T) {
	descriptor := &PerfCounterDescriptor{
		Name: "test",
		Path: "Invalid Counter Path",
	}

	_, err := NewPerfCounter(descriptor)
	if assert.Error(t, err) {
		assert.Regexp(t, "^Unable to parse the counter path", err.Error())
	}
}

func TestNewPerfCounter_Valid(t *testing.T) {
	descriptor := &PerfCounterDescriptor{
		Name: "test",
		Path: `\Memory\Committed Bytes`,
	}

	pc, err := NewPerfCounter(descriptor)
	require.NoError(t, err, "Failed to create performance counter: %v", err)

	assert.Equal(t, descriptor, pc.descriptor)
	assert.NotNil(t, pc.query)
	assert.NotNil(t, pc.handle)

	var vals []CounterValue
	vals, err = pc.query.GetFormattedCounterArrayDouble(pc.handle)
	require.NoError(t, err)
	assert.Equal(t, []CounterValue{{InstanceName: "", Value: 0}}, vals)

	err = pc.query.Close()
	require.NoError(t, err, "Failed to close initialized performance counter query: %v", err)
}

func TestNewPerfCounter_ScrapeOnStartup(t *testing.T) {
	descriptor := &PerfCounterDescriptor{
		Name:            "test",
		Path:            `\Memory\Committed Bytes`,
		ScrapeOnStartup: true,
	}

	pc, err := NewPerfCounter(descriptor)
	require.NoError(t, err, "Failed to create performance counter: %v", err)

	assert.Equal(t, descriptor, pc.descriptor)
	assert.NotNil(t, pc.query)
	assert.NotNil(t, pc.handle)

	var vals []CounterValue
	vals, err = pc.query.GetFormattedCounterArrayDouble(pc.handle)
	require.NoError(t, err)
	assert.Greater(t, vals[0].Value, float64(0))

	err = pc.query.Close()
	require.NoError(t, err, "Failed to close initialized performance counter query: %v", err)
}

func TestPerfCounter_Close(t *testing.T) {
	pc, err := NewPerfCounter(&PerfCounterDescriptor{Name: "test", Path: `\Memory\Committed Bytes`})
	require.NoError(t, err)

	err = pc.Close()
	require.NoError(t, err, "Failed to close initialized performance counter query: %v", err)

	err = pc.Close()
	if assert.Error(t, err) {
		assert.Equal(t, "uninitialised query", err.Error())
	}
}

func TestPerfCounter_ScrapeData(t *testing.T) {
	pc, err := NewPerfCounter(&PerfCounterDescriptor{Name: "test", Path: `\Memory\Committed Bytes`})
	require.NoError(t, err)

	performanceCounters, err := pc.ScrapeData()
	require.NoError(t, err, "Failed to collect data: %v", err)

	assert.Equal(t, 1, len(performanceCounters))
	assert.NotNil(t, 1, performanceCounters[0])
}

func TestPerfCounter_InitializeMetricFromPerfCounterData_NoLabels(t *testing.T) {
	data := []CounterValue{{InstanceName: "", Value: 100}}

	metric := pdata.NewMetric()
	metric.InitEmpty()
	InitializeMetricFromPerfCounterData(metric, data, "", time.Now(), nil)

	ddp := metric.DoubleDataPoints()
	assert.Equal(t, 1, ddp.Len())
	assert.Equal(t, 0, ddp.At(0).LabelsMap().Cap())
	assert.Equal(t, float64(100), ddp.At(0).Value())
}

func TestPerfCounter_ScrapeDataAndConvertToMetric_Labels(t *testing.T) {
	data := []CounterValue{{InstanceName: "label_value_1", Value: 20}, {InstanceName: "label_value_2", Value: 50}}

	metric := pdata.NewMetric()
	metric.InitEmpty()
	InitializeMetricFromPerfCounterData(metric, data, "label", time.Now(), nil)

	ddp := metric.DoubleDataPoints()
	assert.Equal(t, 2, ddp.Len())
	assert.Equal(t, pdata.NewStringMap().InitFromMap(map[string]string{"label": "label_value_1"}), ddp.At(0).LabelsMap().Sort())
	assert.Equal(t, float64(20), ddp.At(0).Value())
	assert.Equal(t, pdata.NewStringMap().InitFromMap(map[string]string{"label": "label_value_2"}), ddp.At(1).LabelsMap().Sort())
	assert.Equal(t, float64(50), ddp.At(1).Value())
}
