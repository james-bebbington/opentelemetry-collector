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
	"runtime"
	"time"

	"github.com/shirou/gopsutil/mem"
	"go.opencensus.io/trace"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdatautil"
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
)

// Scraper for CPU Metrics
type Scraper struct {
	config    *Config
	consumer  consumer.MetricsConsumer
	startTime time.Time
	cancel    context.CancelFunc
}

// NewCPUScraper creates a set of CPU related metrics
func NewCPUScraper(ctx context.Context, cfg *Config, consumer consumer.MetricsConsumer) (*Scraper, error) {
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
	_, span := trace.StartSpan(ctx, "cpuscraper.scrapeMemoryMetrics")
	defer span.End()

	memInfo, err := mem.VirtualMemory()
	if err != nil {
		return err
	}

	metric := pdatautil.AddMetric(metrics)
	initializeMetricVirtualMemoryFrom(memInfo, c.startTime, metric)
	return nil
}

func initializeMetricVirtualMemoryFrom(memInfo *mem.VirtualMemoryStat, startTime time.Time, metric pdata.Metric) {
	InitializeMetricMemoryUsedDescriptor(metric.MetricDescriptor())

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	idps := metric.Int64DataPoints()
	idps.Resize(4)
	for i, stateTime := range []struct {
		label    string
		bytesVal uint64
	}{
		{"total", memInfo.Total},
		{"available", memInfo.Available},
		{"used", memInfo.Used},
		{"utilization", uint64(memInfo.UsedPercent)},
	} {
		idp := idps.At(i)
		idp.LabelsMap().InitFromMap(map[string]string{StateLabel: stateTime.label})
		idp.SetStartTime(pdata.TimestampUnixNano(uint64(startTime.UnixNano())))
		idp.SetTimestamp(pdata.TimestampUnixNano(uint64(time.Now().UnixNano())))
		idp.SetValue(int64(stateTime.bytesVal))
	}
}
