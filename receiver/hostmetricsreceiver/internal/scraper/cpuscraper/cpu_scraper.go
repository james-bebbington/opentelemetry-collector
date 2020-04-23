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
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal"
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
	metrics := internal.InitializeMetricSlice(metricData)

	err := c.scrapeAndAppendMetrics(metrics)
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

func (c *Scraper) scrapeAndAppendMetrics(metrics pdata.MetricSlice) error {
	cpuTimes, err := cpu.Times(c.config.ReportPerCPU)
	if err != nil {
		return err
	}

	cpuSecondsData := flattenCPUTimesStatData(cpuTimes)
	metric := internal.AddMetric(metrics)
	initializeMetricCPUSecondsFrom(cpuSecondsData, c.startTime, metric)
	return nil
}

type cpuSecondsData struct {
	cpuLabel   string
	stateLabel string
	value      int64
}

func flattenCPUTimesStatData(cpuTimes []cpu.TimesStat) []cpuSecondsData {
	result := []cpuSecondsData{}

	for _, cpuTime := range cpuTimes {
		result = append(result, cpuSecondsData{cpuLabel: cpuTime.CPU, stateLabel: UserStateLabelValue, value: int64(cpuTime.User)})
		result = append(result, cpuSecondsData{cpuLabel: cpuTime.CPU, stateLabel: SystemStateLabelValue, value: int64(cpuTime.System)})
		result = append(result, cpuSecondsData{cpuLabel: cpuTime.CPU, stateLabel: IdleStateLabelValue, value: int64(cpuTime.Idle)})
		result = append(result, cpuSecondsData{cpuLabel: cpuTime.CPU, stateLabel: InterruptStateLabelValue, value: int64(cpuTime.Irq)})
		result = append(result, cpuSecondsData{cpuLabel: cpuTime.CPU, stateLabel: IowaitStateLabelValue, value: int64(cpuTime.Iowait)})
	}

	return result
}

func initializeMetricCPUSecondsFrom(cpuData []cpuSecondsData, startTime time.Time, metric pdata.Metric) {
	InitializeMetricCPUSecondsDescriptor(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(len(cpuData))
	for i, c := range cpuData {
		dataPoint := idps.At(i)
		initializeCPUSecondsMetricDatapoint(startTime, c, dataPoint)
	}
}

const gopsCPUTotal string = "cpu-total"

func initializeCPUSecondsMetricDatapoint(startTime time.Time, cpuData cpuSecondsData, dataPoint pdata.Int64DataPoint) {
	labelsMap := dataPoint.LabelsMap()
	if cpuData.cpuLabel != gopsCPUTotal {
		labelsMap.Insert(CPULabel, cpuData.cpuLabel)
	}
	labelsMap.Insert(StateLabel, cpuData.stateLabel)

	dataPoint.SetStartTime(pdata.TimestampUnixNano(uint64(startTime.UnixNano())))
	dataPoint.SetTimestamp(pdata.TimestampUnixNano(uint64(time.Now().UnixNano())))
	dataPoint.SetValue(cpuData.value)
}
