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
	"fmt"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"go.opencensus.io/trace"

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
}

// NewCPUScraper creates a set of CPU related metrics
func NewCPUScraper(ctx context.Context, cfg *Config, consumer consumer.MetricsConsumer) (*Scraper, error) {
	return &Scraper{config: cfg, consumer: consumer}, nil
}

// Start
func (c *Scraper) Start(ctx context.Context) error {
	go func() {
		c.startTime = time.Now()

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
	return nil
}

func (c *Scraper) scrapeMetrics(ctx context.Context) {
	ctx, span := trace.StartSpan(ctx, "cpuscraper.scrapeMetrics")
	defer span.End()

	metricData := data.NewMetricData()
	metrics := pdatautil.InitializeMetricSlice(metricData)

	err := c.scrapeAndAppendMetrics(ctx, metrics)
	if err != nil {
		span.SetStatus(trace.Status{Code: trace.StatusCodeDataLoss, Message: fmt.Sprintf("Error(s) when scraping cpu metrics: %v", err)})
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
	err := c.scrapeSecondsMetric(ctx, metrics)
	if err != nil {
		return err
	}

	err = c.scrapeUtilizationMetric(ctx, metrics)
	if err != nil {
		return err
	}

	if c.config.ReportPerProcess {
		err = c.scrapePerProcessMetric(ctx, metrics)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Scraper) scrapeSecondsMetric(ctx context.Context, metrics pdata.MetricSlice) error {
	_, span := trace.StartSpan(ctx, "cpuscraper.scrapeCpuTimesMetric")
	defer span.End()

	cpuTimes, err := cpu.Times(c.config.ReportPerCPU)
	if err != nil {
		return err
	}

	metric := pdatautil.AddMetric(metrics)
	initializeMetricCPUSecondsFrom(cpuTimes, c.startTime, metric)
	return nil
}

func initializeMetricCPUSecondsFrom(cpuTimes []cpu.TimesStat, startTime time.Time, metric pdata.Metric) {
	InitializeMetricCPUSecondsDescriptor(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(5 * len(cpuTimes))
	for i, cpuTime := range cpuTimes {
		for j, stateTime := range []struct {
			label   string
			timeVal float64
		}{
			{UserStateLabelValue, cpuTime.User},
			{SystemStateLabelValue, cpuTime.System},
			{IdleStateLabelValue, cpuTime.Idle},
			{InterruptStateLabelValue, cpuTime.Irq},
			{IowaitStateLabelValue, cpuTime.Iowait},
		} {
			idp := idps.At(i*5 + j)
			idp.LabelsMap().InitFromMap(map[string]string{StateLabel: stateTime.label, CPULabel: cpuTime.CPU})
			idp.SetStartTime(pdata.TimestampUnixNano(uint64(startTime.UnixNano())))
			idp.SetTimestamp(pdata.TimestampUnixNano(uint64(time.Now().UnixNano())))
			idp.SetValue(int64(stateTime.timeVal))
		}
	}
}
