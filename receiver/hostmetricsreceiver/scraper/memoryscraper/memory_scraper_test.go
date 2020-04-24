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

package memoryscraper

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-collector/component/componenttest"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
	"github.com/open-telemetry/opentelemetry-collector/exporter/exportertest"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal"
)

type validationFn func(*testing.T, []pdata.Metrics)

func TestScrapeMetrics_MinimalData(t *testing.T) {
	createScraperAndValidateScrapedMetrics(t, &Config{}, func(t *testing.T, got []pdata.Metrics) {
		metrics := internal.AssertSingleMetricDataAndGetMetricsSlice(t, got)

		// expect 3 metrics
		assert.Equal(t, 3, metrics.Len())

		// for memory used metric, expect 1 timeseries
		hostMemoryUsedMetric := metrics.At(0)
		expectedMemoryUsedDescriptor := InitializeMetricMemoryUsedDescriptor(pdata.NewMetricDescriptor())
		internal.AssertDescriptorEqual(t, expectedMemoryUsedDescriptor, hostMemoryUsedMetric.MetricDescriptor())
		assert.Equal(t, 1, hostMemoryUsedMetric.Int64DataPoints().Len())
		hostMemoryUsedVal := hostMemoryUsedMetric.Int64DataPoints().At(0).Value()

		// for memory total metric, expect 1 timeseries with a value > used value
		hostMemoryTotalMetric := metrics.At(1)
		expectedMemoryTotalDescriptor := InitializeMetricMemoryTotalDescriptor(pdata.NewMetricDescriptor())
		internal.AssertDescriptorEqual(t, expectedMemoryTotalDescriptor, hostMemoryTotalMetric.MetricDescriptor())
		assert.Equal(t, 1, hostMemoryTotalMetric.Int64DataPoints().Len())
		hostMemoryTotalVal := hostMemoryTotalMetric.Int64DataPoints().At(0).Value()
		assert.Greater(t, hostMemoryTotalVal, hostMemoryUsedVal)

		// for memory utilization metric, expect 1 timeseries with a value = used / total
		hostMemoryUtilizationMetric := metrics.At(2)
		expectedMemoryUtilizationDescriptor := InitializeMetricMemoryUtilizationDescriptor(pdata.NewMetricDescriptor())
		internal.AssertDescriptorEqual(t, expectedMemoryUtilizationDescriptor, hostMemoryUtilizationMetric.MetricDescriptor())
		assert.Equal(t, 1, hostMemoryUtilizationMetric.DoubleDataPoints().Len())
		hostMemoryUtilizationVal := hostMemoryUtilizationMetric.DoubleDataPoints().At(0).Value()
		assert.Equal(t, hostMemoryUtilizationVal, float64(hostMemoryUsedVal)/float64(hostMemoryTotalVal))
	})
}

func TestScrapeMetrics_AllData(t *testing.T) {
	// TODO
}

func createScraperAndValidateScrapedMetrics(t *testing.T, config *Config, assertFn validationFn) {
	config.SetCollectionInterval(10 * time.Millisecond)

	sink := &exportertest.SinkMetricsExporter{}

	scraper, err := NewMemoryScraper(context.Background(), config, sink)
	require.NoError(t, err, "Failed to create memory scraper: %v", err)

	err = scraper.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err, "Failed to start memory scraper: %v", err)

	require.Eventually(t, func() bool {
		got := sink.AllMetrics()
		if len(got) == 0 {
			return false
		}

		defer func() { assert.NoError(t, scraper.Close(context.Background())) }()

		assertFn(t, got)
		return true
	}, 100*time.Millisecond, 5*time.Millisecond, "No metrics were collected")
}
