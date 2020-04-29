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

package vmemscraper

import (
	"context"
	"runtime"

	"go.opencensus.io/trace"

	"github.com/open-telemetry/opentelemetry-collector/component/componenterror"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdatautil"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal/windows"
)

var vmems = float64(runtime.NumVMem())

var hostUtilizationDescriptor = &windows.PerfCounterDescriptor{
	Path:            `Pages Input/sec`,
	ScrapeOnStartup: true,
}

var hostUtilizationTotalDescriptor = &windows.PerfCounterDescriptor{
	Path:            `Pages Output/sec`,
	ScrapeOnStartup: true,
}

var processUtilizationDescriptor = &windows.PerfCounterDescriptor{
	Path:            `Pages/sec`, // this is just the sum of pages input & pages output
	ScrapeOnStartup: true,
}

var (
	hostUtilizationCounter    *windows.PerfCounter
	processUtilizationCounter *windows.PerfCounter
)

var processUtilizationTransformFn = func(val float64) float64 { return val / vmems }

func (c *Scraper) initialize() error {
	var err error
	if c.config.ReportPerVMem {
		hostUtilizationCounter, err = windows.NewPerfCounter(hostUtilizationDescriptor)
	} else {
		hostUtilizationCounter, err = windows.NewPerfCounter(hostUtilizationTotalDescriptor)
	}
	if err != nil {
		return err
	}

	if c.config.ReportPerProcess {
		processUtilizationCounter, err = windows.NewPerfCounter(processUtilizationDescriptor)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Scraper) close() error {
	var errs []error

	if hostUtilizationCounter != nil {
		err := hostUtilizationCounter.Close()
		if err != nil {
			errs = append(errs, err)
		}
		hostUtilizationCounter = nil
	}

	if processUtilizationCounter != nil {
		err := processUtilizationCounter.Close()
		if err != nil {
			errs = append(errs, err)
		}
		processUtilizationCounter = nil
	}

	if len(errs) > 0 {
		return componenterror.CombineErrors(errs)
	}

	return nil
}

func (c *Scraper) scrapeUtilizationMetric(ctx context.Context, metrics pdata.MetricSlice) error {
	_, span := trace.StartSpan(ctx, "vmemscraper.scrapeCpuUtilizationMetric")
	defer span.End()

	vmemUtilizations, err := hostUtilizationCounter.ScrapeData()
	if err != nil {
		return err
	}

	metric := pdatautil.AddMetric(metrics)
	InitializeMetricVMemUtilizationDescriptor(metric.MetricDescriptor())
	windows.InitializeMetricFromPerfCounterData(metric, vmemUtilizations, VMemLabel, c.startTime, nil)
	return nil
}

func (c *Scraper) scrapePerProcessMetric(ctx context.Context, metrics pdata.MetricSlice) error {
	_, span := trace.StartSpan(ctx, "vmemscraper.scrapeCpuPerProcessMetric")
	defer span.End()

	proccessUtilizations, err := processUtilizationCounter.ScrapeData()
	if err != nil {
		return err
	}

	metric := pdatautil.AddMetric(metrics)
	InitializeMetricProcessUtilizationDescriptor(metric.MetricDescriptor())
	windows.InitializeMetricFromPerfCounterData(metric, proccessUtilizations, ProcessNameLabel, c.startTime, processUtilizationTransformFn)
	return nil
}
