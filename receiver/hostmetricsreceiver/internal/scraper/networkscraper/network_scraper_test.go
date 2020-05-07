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

package networkscraper

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
	"github.com/open-telemetry/opentelemetry-collector/exporter/exportertest"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal"
)

type validationFn func(*testing.T, []pdata.Metrics)

func TestScrapeMetrics(t *testing.T) {
	createScraperAndValidateScrapedMetrics(t, &Config{}, func(t *testing.T, got []pdata.Metrics) {
		metrics := internal.AssertSingleMetricDataAndGetMetricsSlice(t, got)

		// expect 5 metrics
		assert.Equal(t, 5, metrics.Len())

		// for network packets metric, expect a transmit & receive datapoint
		hostNetworkPacketsMetric := metrics.At(0)
		internal.AssertDescriptorEqual(t, metricNetworkPacketsDescriptor, hostNetworkPacketsMetric.MetricDescriptor())
		assert.Equal(t, 2, hostNetworkPacketsMetric.Int64DataPoints().Len())
		internal.AssertInt64MetricLabelHasValue(t, hostNetworkPacketsMetric, 0, directionLabelName, transmitDirectionLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, hostNetworkPacketsMetric, 1, directionLabelName, receiveDirectionLabelValue)

		// for network dropped packets metric, expect a transmit & receive datapoint
		hostNetworkDroppedPacketsMetric := metrics.At(1)
		internal.AssertDescriptorEqual(t, metricNetworkDroppedPacketsDescriptor, hostNetworkDroppedPacketsMetric.MetricDescriptor())
		assert.Equal(t, 2, hostNetworkDroppedPacketsMetric.Int64DataPoints().Len())
		internal.AssertInt64MetricLabelHasValue(t, hostNetworkDroppedPacketsMetric, 0, directionLabelName, transmitDirectionLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, hostNetworkDroppedPacketsMetric, 1, directionLabelName, receiveDirectionLabelValue)

		// for network errors metric, expect a transmit & receive datapoint
		hostNetworkErrorsMetric := metrics.At(2)
		internal.AssertDescriptorEqual(t, metricNetworkErrorsDescriptor, hostNetworkErrorsMetric.MetricDescriptor())
		assert.Equal(t, 2, hostNetworkErrorsMetric.Int64DataPoints().Len())
		internal.AssertInt64MetricLabelHasValue(t, hostNetworkErrorsMetric, 0, directionLabelName, transmitDirectionLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, hostNetworkErrorsMetric, 1, directionLabelName, receiveDirectionLabelValue)

		// for network bytes metric, expect a transmit & receive datapoint
		hostNetworkBytesMetric := metrics.At(3)
		internal.AssertDescriptorEqual(t, metricNetworkBytesDescriptor, hostNetworkBytesMetric.MetricDescriptor())
		assert.Equal(t, 2, hostNetworkBytesMetric.Int64DataPoints().Len())
		internal.AssertInt64MetricLabelHasValue(t, hostNetworkBytesMetric, 0, directionLabelName, transmitDirectionLabelValue)
		internal.AssertInt64MetricLabelHasValue(t, hostNetworkBytesMetric, 1, directionLabelName, receiveDirectionLabelValue)

		// for tcp connections metric, expect at least one datapoint with a state label
		hostNetworkTCPConnectionsMetric := metrics.At(4)
		internal.AssertDescriptorEqual(t, metricNetworkTCPConnectionDescriptor, hostNetworkTCPConnectionsMetric.MetricDescriptor())
		internal.AssertInt64MetricLabelExists(t, hostNetworkTCPConnectionsMetric, 0, stateLabelName)
		assert.GreaterOrEqual(t, hostNetworkTCPConnectionsMetric.Int64DataPoints().Len(), 1)
	})
}

func createScraperAndValidateScrapedMetrics(t *testing.T, config *Config, assertFn validationFn) {
	config.SetCollectionInterval(5 * time.Millisecond)

	sink := &exportertest.SinkMetricsExporter{}

	scraper, err := NewNetworkScraper(context.Background(), config, sink)
	require.NoError(t, err, "Failed to create network scraper: %v", err)

	err = scraper.Start(context.Background())
	require.NoError(t, err, "Failed to start network scraper: %v", err)
	defer func() { assert.NoError(t, scraper.Shutdown(context.Background())) }()

	require.Eventually(t, func() bool {
		got := sink.AllMetrics()
		if len(got) == 0 {
			return false
		}

		assertFn(t, got)
		return true
	}, time.Second, 2*time.Millisecond, "No metrics were collected")
}
