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

package cpuscraper

import (
	"context"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"go.opencensus.io/trace"

	"github.com/open-telemetry/opentelemetry-collector/component/componenterror"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/internal/windows"
)

var cpus = float64(runtime.NumCPU())

var hostUtilizationDescriptor = &windows.PerfCounterDescriptor{
	Name:            "host/cpu/utilization",
	Path:            `\Processor(*)\% Processor Time`,
	ScrapeOnStartup: true,
}

var hostUtilizationTotalDescriptor = &windows.PerfCounterDescriptor{
	Name:            "host/cpu/utilization",
	Path:            `\Processor(_Total)\% Processor Time`,
	ScrapeOnStartup: true,
}

var processUtilizationDescriptor = &windows.PerfCounterDescriptor{
	Name:            "process/cpu/utilization",
	Path:            `\Process(*)\% Processor Time`,
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
	if c.config.ReportPerCPU {
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

func (c *Scraper) scrapePerCoreMetrics(ctx context.Context, metrics pdata.MetricSlice) error {
	_, span := trace.StartSpan(ctx, "cpuscraper.scrapePerCoreMetrics")
	defer span.End()

	var errs []error

	// host/cpu/time

	cpuTimes, err := cpu.Times(c.config.ReportPerCPU)
	if err != nil {
		errs = append(errs, err)
	} else {
		metric := internal.AddMetric(metrics)
		InitializeMetricCPUSecondsDescriptor(metric.MetricDescriptor())
		initializeMetricCPUSecondsFrom(cpuTimes, c.startTime, metric)
	}

	// host/cpu/utilization

	cpuUtilizations, err := hostUtilizationCounter.ScrapeData()
	if err != nil {
		errs = append(errs, err)
	} else {
		metric := internal.AddMetric(metrics)
		InitializeMetricCPUUtilizationDescriptor(metric.MetricDescriptor())
		windows.InitializeMetricFromPerfCounterData(metric, cpuUtilizations, CPULabel, c.startTime, nil)
	}

	if len(errs) > 0 {
		err = componenterror.CombineErrors(errs)
	}

	return err
}

func (c *Scraper) scrapePerProcessMetrics(ctx context.Context, metrics pdata.MetricSlice) error {
	if !c.config.ReportPerProcess {
		return nil
	}

	_, span := trace.StartSpan(ctx, "cpuscraper.scrapePerProcessMetrics")
	defer span.End()

	// process/cpu/utilization

	proccessUtilizations, err := processUtilizationCounter.ScrapeData()
	if err != nil {
		return err
	}

	metric := internal.AddMetric(metrics)
	InitializeMetricProcessUtilizationDescriptor(metric.MetricDescriptor())
	windows.InitializeMetricFromPerfCounterData(metric, proccessUtilizations, ProcessNameLabel, c.startTime, processUtilizationTransformFn)

	return nil
}

func initializeMetricCPUSecondsFrom(cpuTimes []cpu.TimesStat, startTime time.Time, metric pdata.Metric) {
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
