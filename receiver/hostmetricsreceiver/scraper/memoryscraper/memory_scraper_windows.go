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
	"runtime"

	"go.opencensus.io/trace"

	"github.com/open-telemetry/opentelemetry-collector/component/componenterror"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal/windows"
)

var cpus = float64(runtime.NumCPU())

// get total & used from gopsutil

var hostCachedDescriptor = &windows.PerfCounterDescriptor{
	Path:            `\Memory\Cache Bytes`, // bytes of system data currently cached in memory
	ScrapeOnStartup: true,
}

var hostNonpagedDescriptor = &windows.PerfCounterDescriptor{
	Path:            `\Memory\Pool Nonpaged Bytes`, // bytes of system data stored in memory that cannot be paged (written to disk)
	ScrapeOnStartup: true,
}

var processMemoryDescriptor = &windows.PerfCounterDescriptor{
	Path:            `\Process(*)\Working Set - Private`,
	ScrapeOnStartup: true,
}

var (
	hostUtilizationCounter    *windows.PerfCounter
	processUtilizationCounter *windows.PerfCounter
)

// "Process\% Processor time" is reported per cpu so we need to
// use this function to scale the value based on #cpu cores
var processUtilizationTransformFn = func(val float64) float64 { return val / cpus }

func (c *Scraper) initialize() error {
	var err error
	hostUtilizationCounter, err = windows.NewPerfCounter(hostUtilizationDescriptor)
	if err != nil {
		return err
	}

	processUtilizationCounter, err = windows.NewPerfCounter(processUtilizationDescriptor)
	if err != nil {
		return err
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
	_, span := trace.StartSpan(ctx, "cpuscraper.scrapeCpuUtilizationMetric")
	defer span.End()

	_, err := hostUtilizationCounter.ScrapeData()
	if err != nil {
		return err
	}

	return nil
}

func (c *Scraper) scrapePerProcessMetric(ctx context.Context, metrics pdata.MetricSlice) error {
	_, span := trace.StartSpan(ctx, "cpuscraper.scrapeCpuPerProcessMetric")
	defer span.End()

	_, err := processUtilizationCounter.ScrapeData()
	if err != nil {
		return err
	}

	return nil
}
