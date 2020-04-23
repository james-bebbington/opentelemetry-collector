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
	"github.com/open-telemetry/opentelemetry-collector/exporter/exportertest"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal"
)

type validationFn func(*testing.T, []pdata.Metrics)

func TestScrapeMetrics_MinimalData(t *testing.T) {
	createScraperAndValidateScrapedMetrics(t, &Config{}, func(t *testing.T, got []pdata.Metrics) {
		metrics := internal.AssertSingleMetricDataAndGetMetricsSlice(t, got)

		// expect 1 metric
		assert.Equal(t, 1, metrics.Len())

		// for cpu seconds metric, expect 5 timeseries with appropriate labels
		hostCPUTimeMetric := metrics.At(0)
		expectedCPUSecondsDescriptor := InitializeMetricCPUSecondsDescriptor(pdata.NewMetricDescriptor())
		internal.AssertDescriptorEqual(t, expectedCPUSecondsDescriptor, hostCPUTimeMetric.MetricDescriptor())
		assert.Equal(t, 5, hostCPUTimeMetric.Int64DataPoints().Len())
		internal.AssertInt64MetricLabelDoesNotExist(t, hostCPUTimeMetric, 0, CPULabel)
		internal.AssertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 0, StateLabel, UserStateLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 1, StateLabel, SystemStateLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 2, StateLabel, IdleStateLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 3, StateLabel, InterruptStateLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 4, StateLabel, IowaitStateLabelValue)
	})
}

func TestScrapeMetrics_AllData(t *testing.T) {
	config := &Config{
		ReportPerCPU: true,
	}

	createScraperAndValidateScrapedMetrics(t, config, func(t *testing.T, got []pdata.Metrics) {
		metrics := internal.AssertSingleMetricDataAndGetMetricsSlice(t, got)

		// expect 1 metric
		assert.Equal(t, 1, metrics.Len())

		// for cpu seconds metric, expect 5*cores timeseries with appropriate labels
		hostCPUTimeMetric := metrics.At(0)
		expectedCPUSecondsDescriptor := InitializeMetricCPUSecondsDescriptor(pdata.NewMetricDescriptor())
		internal.AssertDescriptorEqual(t, expectedCPUSecondsDescriptor, hostCPUTimeMetric.MetricDescriptor())
		assert.Equal(t, 5*runtime.NumCPU(), hostCPUTimeMetric.Int64DataPoints().Len())
		internal.AssertInt64MetricLabelExists(t, hostCPUTimeMetric, 0, CPULabel)
		internal.AssertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 0, StateLabel, UserStateLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 1, StateLabel, SystemStateLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 2, StateLabel, IdleStateLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 3, StateLabel, InterruptStateLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 4, StateLabel, IowaitStateLabelValue)
	})
}

func createScraperAndValidateScrapedMetrics(t *testing.T, config *Config, assertFn validationFn) {
	config.SetCollectionInterval(100 * time.Millisecond)

	sink := &exportertest.SinkMetricsExporter{}

	scraper, err := NewCPUScraper(context.Background(), config, sink)
	require.NoError(t, err, "Failed to create cpu scraper: %v", err)

	err = scraper.Start(context.Background())
	require.NoError(t, err, "Failed to start cpu scraper: %v", err)
	defer func() { assert.NoError(t, scraper.Close(context.Background())) }()

	require.Eventually(t, func() bool {
		got := sink.AllMetrics()
		if len(got) == 0 {
			return false
		}

		assertFn(t, got)
		return true
	}, time.Second, 10*time.Millisecond, "No metrics were collected")
}
