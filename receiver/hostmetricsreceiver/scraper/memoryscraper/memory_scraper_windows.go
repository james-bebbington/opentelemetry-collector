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

package memoryscraper

import (
	"context"

	"go.opencensus.io/trace"

	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdatautil"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal/windows"
)

var processMemoryDescriptor = &windows.PerfCounterDescriptor{
	Path:            `\Process(*)\Working Set - Private`,
	ScrapeOnStartup: true,
}

var processMemoryCounter *windows.PerfCounter

func (c *Scraper) initialize() error {
	var err error
	if c.config.ReportPerProcess {
		processMemoryCounter, err = windows.NewPerfCounter(processMemoryDescriptor)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Scraper) close() error {
	if processMemoryCounter != nil {
		err := processMemoryCounter.Close()
		if err != nil {
			return err
		}
		processMemoryCounter = nil
	}

	return nil
}

func (c *Scraper) scrapeMemoryPerProcessMetric(ctx context.Context, metrics pdata.MetricSlice) error {
	_, span := trace.StartSpan(ctx, "memoryscraper.scrapeMemoryPerProcessMetric")
	defer span.End()

	proccessUtilizations, err := processMemoryCounter.ScrapeData()
	if err != nil {
		return err
	}

	metric := pdatautil.AddMetric(metrics)
	InitializeMetricProcessUsedDescriptor(metric.MetricDescriptor())
	windows.InitializeMetricFromPerfCounterData(metric, proccessUtilizations, ProcessNameLabel, c.startTime, nil)
	return nil
}
