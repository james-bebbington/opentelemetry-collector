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
	"fmt"
	"time"

	"github.com/shirou/gopsutil/mem"
	"go.opencensus.io/trace"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdatautil"
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
)

// Scraper for Memory Metrics
type Scraper struct {
	config    *Config
	consumer  consumer.MetricsConsumer
	startTime time.Time
	cancel    context.CancelFunc
}

// NewMemoryScraper creates a set of Memory related metrics
func NewMemoryScraper(ctx context.Context, cfg *Config, consumer consumer.MetricsConsumer) (*Scraper, error) {
	return &Scraper{config: cfg, consumer: consumer}, nil
}

// Start
func (c *Scraper) Start(ctx context.Context, host component.Host) error {
	ctx, c.cancel = context.WithCancel(ctx)

	go func() {
		c.startTime = time.Now()
		err := c.initialize()
		if err != nil {
			host.ReportFatalError(err)
			return
		}

		ticker := time.NewTicker(c.config.CollectionInterval())
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.scrapeMetrics(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

// Close
func (c *Scraper) Close(ctx context.Context) error {
	c.cancel()
	return c.close()
}

func (c *Scraper) scrapeMetrics(ctx context.Context) {
	ctx, span := trace.StartSpan(ctx, "memoryscraper.scrapeMetrics")
	defer span.End()

	metricData := data.NewMetricData()
	metrics := pdatautil.InitializeMetricSlice(metricData)

	err := c.scrapeAndAppendMetrics(ctx, metrics)
	if err != nil {
		span.SetStatus(trace.Status{Code: trace.StatusCodeDataLoss, Message: fmt.Sprintf("Error(s) when scraping memory metrics: %v", err)})
		return
	}

	if metrics.Len() > 0 {
		err := c.consumer.ConsumeMetrics(ctx, pdatautil.MetricsFromInternalMetrics(metricData))
		if err != nil {
			span.SetStatus(trace.Status{Code: trace.StatusCodeDataLoss, Message: fmt.Sprintf("Unable to process metrics: %v", err)})
			return
		}
	}
}

func (c *Scraper) scrapeAndAppendMetrics(ctx context.Context, metrics pdata.MetricSlice) error {
	_, span := trace.StartSpan(ctx, "memoryscraper.scrapeMetrics")
	defer span.End()

	err := c.scrapeHostMemoryMetrics(ctx, metrics)
	if err != nil {
		return err
	}

	if c.config.ReportPerProcess {
		err = c.scrapeMemoryPerProcessMetric(ctx, metrics)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Scraper) scrapeHostMemoryMetrics(ctx context.Context, metrics pdata.MetricSlice) error {
	_, span := trace.StartSpan(ctx, "memoryscraper.scrapeTotalAndUtilizationMemoryMetrics")
	defer span.End()

	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return err
	}

	metric := pdatautil.AddMetric(metrics)
	initializeMetricUsedMemoryFrom(memInfo, c.startTime, metric)

	metric = pdatautil.AddMetric(metrics)
	initializeMetricTotalMemoryFrom(memInfo, c.startTime, metric)

	metric = pdatautil.AddMetric(metrics)
	initializeMetricMemoryUtilizationFrom(memInfo, c.startTime, metric)
	return nil
}

func initializeMetricUsedMemoryFrom(memInfo *mem.VirtualMemoryStat, startTime time.Time, metric pdata.Metric) {
	InitializeMetricMemoryUsedDescriptor(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(1)
	idp := idps.At(0)
	idp.LabelsMap().InitFromMap(map[string]string{})
	idp.SetStartTime(pdata.TimestampUnixNano(uint64(startTime.UnixNano())))
	idp.SetTimestamp(pdata.TimestampUnixNano(uint64(time.Now().UnixNano())))
	idp.SetValue(int64(memInfo.Used))
}

func initializeMetricTotalMemoryFrom(memInfo *mem.VirtualMemoryStat, startTime time.Time, metric pdata.Metric) {
	InitializeMetricMemoryTotalDescriptor(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(1)
	idp := idps.At(0)
	idp.SetStartTime(pdata.TimestampUnixNano(uint64(startTime.UnixNano())))
	idp.SetTimestamp(pdata.TimestampUnixNano(uint64(time.Now().UnixNano())))
	idp.SetValue(int64(memInfo.Total))
}

func initializeMetricMemoryUtilizationFrom(memInfo *mem.VirtualMemoryStat, startTime time.Time, metric pdata.Metric) {
	InitializeMetricMemoryUtilizationDescriptor(metric.MetricDescriptor())

	idps := metric.DoubleDataPoints()
	idps.Resize(1)
	idp := idps.At(0)
	idp.SetStartTime(pdata.TimestampUnixNano(uint64(startTime.UnixNano())))
	idp.SetTimestamp(pdata.TimestampUnixNano(uint64(time.Now().UnixNano())))
	idp.SetValue(float64(memInfo.Used) / float64(memInfo.Total))
}
