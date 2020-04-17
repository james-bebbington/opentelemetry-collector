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

package cpuscraper

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
)

func TestScrapeMetrics_Uninitialized(t *testing.T) {
	scraper, err := NewCPUScraper(&Config{})
	require.NoError(t, err, "Failed to create cpu scraper: %v", err)

	err = scraper.ScrapeMetrics(context.Background(), pdata.NewMetricSlice())
	if assert.Error(t, err) {
		assert.Equal(t, "cpu scraper has not been initialized", err.Error())
	}
}

func TestScrapeMetrics_MinimalData(t *testing.T) {
	gotMetrics, ok := createScraperAndScrapeMetrics(t, &Config{})
	if !ok {
		return
	}

	// expect 2 metrics
	assert.Equal(t, 2, gotMetrics.Len())

	// for cpu seconds metric, expect 5 timeseries with appropriate labels
	hostCPUTimeMetric := gotMetrics.At(0)
	expectedCPUSecondsDescriptor := InitializeMetricCPUSecondsDescriptor(pdata.NewMetricDescriptor())
	assertDescriptorEqual(t, expectedCPUSecondsDescriptor, hostCPUTimeMetric.MetricDescriptor())
	assert.Equal(t, 5, hostCPUTimeMetric.Int64DataPoints().Len())
	assertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 0, StateLabel, UserStateLabelValue)
	assertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 1, StateLabel, SystemStateLabelValue)
	assertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 2, StateLabel, IdleStateLabelValue)
	assertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 3, StateLabel, InterruptStateLabelValue)
	assertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 4, StateLabel, IowaitStateLabelValue)

	// for cpu utilization metric, expect 1 timeseries with a value < 110
	// (value can go over 100% by a small margin)
	hostCPUUtilizationMetric := gotMetrics.At(1)
	expectedCPUUtilizationDescriptor := InitializeMetricCPUUtilizationDescriptor(pdata.NewMetricDescriptor())
	assertDescriptorEqual(t, expectedCPUUtilizationDescriptor, hostCPUUtilizationMetric.MetricDescriptor())
	assert.Equal(t, 1, hostCPUUtilizationMetric.DoubleDataPoints().Len())
	assert.LessOrEqual(t, hostCPUUtilizationMetric.DoubleDataPoints().At(0).Value(), float64(110))
}

func TestScrapeMetrics_AllData(t *testing.T) {
	config := &Config{
		ReportPerCPU:     true,
		ReportPerProcess: true,
	}

	gotMetrics, ok := createScraperAndScrapeMetrics(t, config)
	if !ok {
		return
	}

	// expect 2 metrics
	assert.Equal(t, 3, gotMetrics.Len())

	// for cpu seconds metric, expect 5*#cores timeseries with appropriate labels
	hostCPUTimeMetric := gotMetrics.At(0)
	expectedCPUSecondsDescriptor := InitializeMetricCPUSecondsDescriptor(pdata.NewMetricDescriptor())
	assertDescriptorEqual(t, expectedCPUSecondsDescriptor, hostCPUTimeMetric.MetricDescriptor())
	assert.Equal(t, 5*runtime.NumCPU(), hostCPUTimeMetric.Int64DataPoints().Len())
	assertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 0, StateLabel, UserStateLabelValue)
	assertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 1, StateLabel, SystemStateLabelValue)
	assertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 2, StateLabel, IdleStateLabelValue)
	assertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 3, StateLabel, InterruptStateLabelValue)
	assertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 4, StateLabel, IowaitStateLabelValue)

	// for cpu utilization metric, expect #cores timeseries each with a value < 110
	// (value can go over 100% by a small margin)
	hostCPUUtilizationMetric := gotMetrics.At(1)
	expectedCPUUtilizationDescriptor := InitializeMetricCPUUtilizationDescriptor(pdata.NewMetricDescriptor())
	assertDescriptorEqual(t, expectedCPUUtilizationDescriptor, hostCPUUtilizationMetric.MetricDescriptor())
	ddp := hostCPUUtilizationMetric.DoubleDataPoints()
	assert.Equal(t, runtime.NumCPU(), ddp.Len())
	for i := 0; i < ddp.Len(); i++ {
		assert.LessOrEqual(t, ddp.At(i).Value(), float64(110))
	}

	// for cpu utilization per process metric, expect >1 timeseries
	processCPUUtilizationMetric := gotMetrics.At(2)
	expectedProcessCPUUtilizationDescriptor := InitializeMetricProcessUtilizationDescriptor(pdata.NewMetricDescriptor())
	assertDescriptorEqual(t, expectedProcessCPUUtilizationDescriptor, processCPUUtilizationMetric.MetricDescriptor())
	assert.GreaterOrEqual(t, processCPUUtilizationMetric.DoubleDataPoints().Len(), 1)
}

func createScraperAndScrapeMetrics(t *testing.T, config *Config) (pdata.MetricSlice, bool) {
	var metrics = pdata.NewMetricSlice()

	scraper, err := NewCPUScraper(config)
	require.NoError(t, err, "Failed to create cpu scraper: %v", err)

	err = scraper.Initialize()

	if runtime.GOOS != "windows" {
		require.Error(t, err, "Expected error when creating a cpu scraper on a non-windows environment")
		return metrics, false
	}

	require.NoError(t, err, "Failed to initialize cpu scraper: %v", err)
	defer func() { assert.NoError(t, scraper.Close()) }()

	// need to sleep briefly to ensure enough time has passed for windows
	// to generate two valid processor time measurements
	<-time.After(100 * time.Millisecond)

	err = scraper.ScrapeMetrics(context.Background(), metrics)
	require.NoError(t, err, "Failed to collect cpu metrics: %v", err)

	return metrics, true
}

func assertDescriptorEqual(t *testing.T, expected pdata.MetricDescriptor, actual pdata.MetricDescriptor) {
	assert.Equal(t, expected.Name(), actual.Name())
	assert.Equal(t, expected.Description(), actual.Description())
	assert.Equal(t, expected.Unit(), actual.Unit())
	assert.Equal(t, expected.Type(), actual.Type())
	assert.EqualValues(t, expected.LabelsMap().Sort(), actual.LabelsMap().Sort())
}

func assertInt64MetricLabelHasValue(t *testing.T, metric pdata.Metric, index int, labelName string, expectedVal string) {
	ddp := metric.Int64DataPoints()
	val, ok := ddp.At(index).LabelsMap().Get(labelName)
	assert.Truef(t, ok, "Missing label %q in metric %q", labelName, metric.MetricDescriptor().Name())
	assert.Equal(t, expectedVal, val.Value())
}
