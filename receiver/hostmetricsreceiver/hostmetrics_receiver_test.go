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

package hostmetricsreceiver

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/component/componenttest"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdatautil"
	"github.com/open-telemetry/opentelemetry-collector/exporter/exportertest"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/component"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/scraper/cpuscraper"
)

func TestGatherMetrics_EndToEnd(t *testing.T) {
	sink := &exportertest.SinkMetricsExporter{}

	config := &Config{
		ScrapeInterval: 0,
		Scrapers: map[string]component.ScraperConfig{
			cpuscraper.TypeStr: &cpuscraper.Config{
				ReportPerCPU:     true,
				ReportPerProcess: true,
			},
		},
	}

	factories := map[string]component.ScraperFactory{
		cpuscraper.TypeStr: &cpuscraper.Factory{},
	}

	getSingleTickFn := func() <-chan time.Time {
		c := make(chan time.Time)
		go func() {
			// need to sleep briefly to ensure enough time has passed for windows
			// to generate two valid processor time measurements
			<-time.After(100 * time.Millisecond)

			c <- time.Now()
		}()
		return c
	}

	receiver, err := NewHostMetricsReceiver(zap.NewNop(), config, factories, sink, getSingleTickFn)

	if runtime.GOOS != "windows" {
		require.Error(t, err, "Expected error when creating a metrics receiver with cpuscraper collector on a non-windows environment")
		return
	}

	require.NoError(t, err, "Failed to create metrics receiver: %v", err)

	err = receiver.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err, "Failed to start metrics receiver: %v", err)
	defer func() { assert.NoError(t, receiver.Shutdown(context.Background())) }()

	// TODO consider adding mechanism to get notifed when metrics are processed so we
	// dont have to wait here
	<-time.After(110 * time.Millisecond)

	got := sink.AllMetrics()

	// expect 1 MetricData object
	assert.Equal(t, 1, len(got))
	md := pdatautil.MetricsToInternalMetrics(got[0])

	// expect 1 ResourceMetrics object
	rms := md.ResourceMetrics()
	assert.Equal(t, 1, rms.Len())
	rm := rms.At(0)

	// expect 1 InstrumentationLibraryMetrics object
	ilms := rm.InstrumentationLibraryMetrics()
	assert.Equal(t, 1, ilms.Len())
	ilm := ilms.At(0)

	// expect 3 metrics
	metrics := ilm.Metrics()
	assert.Equal(t, 3, metrics.Len())

	// for cpuscraper seconds metric, expect 5*#cores timeseries with appropriate labels
	hostCPUTimeMetric := metrics.At(0)
	expectedCPUSecondsDescriptor := cpuscraper.InitializeMetricCPUSecondsDescriptor(pdata.NewMetricDescriptor())
	assertDescriptorEqual(t, expectedCPUSecondsDescriptor, hostCPUTimeMetric.MetricDescriptor())
	assert.Equal(t, 5*runtime.NumCPU(), hostCPUTimeMetric.Int64DataPoints().Len())
	assertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 0, cpuscraper.StateLabel, cpuscraper.UserStateLabelValue)
	assertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 1, cpuscraper.StateLabel, cpuscraper.SystemStateLabelValue)
	assertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 2, cpuscraper.StateLabel, cpuscraper.IdleStateLabelValue)
	assertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 3, cpuscraper.StateLabel, cpuscraper.InterruptStateLabelValue)
	assertInt64MetricLabelHasValue(t, hostCPUTimeMetric, 4, cpuscraper.StateLabel, cpuscraper.IowaitStateLabelValue)

	// for cpuscraper utilization metric, expect #cores timeseries each with a value < 110
	// (values can go over 100% by a small margin)
	hostCPUUtilizationMetric := metrics.At(1)
	expectedCPUUtilizationDescriptor := cpuscraper.InitializeMetricCPUUtilizationDescriptor(pdata.NewMetricDescriptor())
	assertDescriptorEqual(t, expectedCPUUtilizationDescriptor, hostCPUUtilizationMetric.MetricDescriptor())
	ddp := hostCPUUtilizationMetric.DoubleDataPoints()
	assert.Equal(t, runtime.NumCPU(), ddp.Len())
	for i := 0; i < ddp.Len(); i++ {
		assert.LessOrEqual(t, ddp.At(i).Value(), float64(110))
	}

	// for cpuscraper utilization per process metric, expect >1 timeseries
	processCPUUtilizationMetric := metrics.At(2)
	expectedProcessCPUUtilizationDescriptor := cpuscraper.InitializeMetricProcessUtilizationDescriptor(pdata.NewMetricDescriptor())
	assertDescriptorEqual(t, expectedProcessCPUUtilizationDescriptor, processCPUUtilizationMetric.MetricDescriptor())
	assert.GreaterOrEqual(t, processCPUUtilizationMetric.DoubleDataPoints().Len(), 1)
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
