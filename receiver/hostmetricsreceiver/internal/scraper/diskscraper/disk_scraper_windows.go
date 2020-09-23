// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package diskscraper

import (
	"context"
	"fmt"
	"time"

	"github.com/shirou/gopsutil/host"

	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/internal/processor/filterset"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal"
	"go.opentelemetry.io/collector/receiver/hostmetricsreceiver/internal/perfcounters"
)

const (
	logicalDiskObject = "LogicalDisk"

	readsPerSec  = "Disk Reads/sec"
	writesPerSec = "Disk Writes/sec"

	readBytesPerSec  = "Disk Read Bytes/sec"
	writeBytesPerSec = "Disk Write Bytes/sec"

	avgDiskSecsPerRead  = "Avg. Disk sec/Read"
	avgDiskSecsPerWrite = "Avg. Disk sec/Write"

	queueLength = "Current Disk Queue Length"
)

// scraper for Disk Metrics
type scraper struct {
	config    *Config
	startTime pdata.TimestampUnixNano
	includeFS filterset.FilterSet
	excludeFS filterset.FilterSet

	perfCounterScraper perfcounters.PerfCounterScraper

	// for mocking
	bootTime func() (uint64, error)
}

// newDiskScraper creates a Disk Scraper
func newDiskScraper(_ context.Context, cfg *Config) (*scraper, error) {
	scraper := &scraper{config: cfg, perfCounterScraper: &perfcounters.PerfLibScraper{}, bootTime: host.BootTime}

	var err error

	if len(cfg.Include.Devices) > 0 {
		scraper.includeFS, err = filterset.CreateFilterSet(cfg.Include.Devices, &cfg.Include.Config)
		if err != nil {
			return nil, fmt.Errorf("error creating device include filters: %w", err)
		}
	}

	if len(cfg.Exclude.Devices) > 0 {
		scraper.excludeFS, err = filterset.CreateFilterSet(cfg.Exclude.Devices, &cfg.Exclude.Config)
		if err != nil {
			return nil, fmt.Errorf("error creating device exclude filters: %w", err)
		}
	}

	return scraper, nil
}

// Initialize
func (s *scraper) Initialize(_ context.Context) error {
	bootTime, err := s.bootTime()
	if err != nil {
		return err
	}

	s.startTime = pdata.TimestampUnixNano(bootTime * 1e9)

	return s.perfCounterScraper.Initialize(logicalDiskObject)
}

// Close
func (s *scraper) Close(_ context.Context) error {
	return nil
}

// ScrapeMetrics
func (s *scraper) ScrapeMetrics(ctx context.Context) (pdata.MetricSlice, error) {
	metrics := pdata.NewMetricSlice()

	now := internal.TimeToUnixNano(time.Now())

	counters, err := s.perfCounterScraper.Scrape()
	if err != nil {
		return metrics, err
	}

	logicalDiskObject, err := counters.GetObject("LogicalDisk")
	if err != nil {
		return metrics, err
	}

	// filter devices by name
	logicalDiskObject.Filter(s.includeFS, s.excludeFS, false)

	logicalDiskCounterValues, err := logicalDiskObject.GetValues(readsPerSec, writesPerSec, readBytesPerSec, writeBytesPerSec, avgDiskSecsPerRead, avgDiskSecsPerWrite, queueLength)
	if err != nil {
		return metrics, err
	}

	if len(logicalDiskCounterValues) > 0 {
		metrics.Resize(4)
		initializeDiskIOMetric(metrics.At(0), s.startTime, now, logicalDiskCounterValues)
		initializeDiskOpsMetric(metrics.At(1), s.startTime, now, logicalDiskCounterValues)
		initializeDiskTimeMetric(metrics.At(2), s.startTime, now, logicalDiskCounterValues)
		initializeDiskPendingOperationsMetric(metrics.At(3), now, logicalDiskCounterValues)
	}

	return metrics, nil
}

func initializeDiskIOMetric(metric pdata.Metric, startTime, now pdata.TimestampUnixNano, logicalDiskCounterValues []*perfcounters.CounterValues) {
	diskIODescriptor.CopyTo(metric)

	idps := metric.IntSum().DataPoints()
	idps.Resize(2 * len(logicalDiskCounterValues))
	for idx, logicalDiskCounter := range logicalDiskCounterValues {
		initializeInt64DataPoint(idps.At(2*idx+0), startTime, now, logicalDiskCounter.InstanceName, readDirectionLabelValue, logicalDiskCounter.Values[readBytesPerSec])
		initializeInt64DataPoint(idps.At(2*idx+1), startTime, now, logicalDiskCounter.InstanceName, writeDirectionLabelValue, logicalDiskCounter.Values[writeBytesPerSec])
	}
}

func initializeDiskOpsMetric(metric pdata.Metric, startTime, now pdata.TimestampUnixNano, logicalDiskCounterValues []*perfcounters.CounterValues) {
	diskOpsDescriptor.CopyTo(metric)

	idps := metric.IntSum().DataPoints()
	idps.Resize(2 * len(logicalDiskCounterValues))
	for idx, logicalDiskCounter := range logicalDiskCounterValues {
		initializeInt64DataPoint(idps.At(2*idx+0), startTime, now, logicalDiskCounter.InstanceName, readDirectionLabelValue, logicalDiskCounter.Values[readsPerSec])
		initializeInt64DataPoint(idps.At(2*idx+1), startTime, now, logicalDiskCounter.InstanceName, writeDirectionLabelValue, logicalDiskCounter.Values[writesPerSec])
	}
}

func initializeDiskTimeMetric(metric pdata.Metric, startTime, now pdata.TimestampUnixNano, logicalDiskCounterValues []*perfcounters.CounterValues) {
	diskTimeDescriptor.CopyTo(metric)

	ddps := metric.DoubleSum().DataPoints()
	ddps.Resize(2 * len(logicalDiskCounterValues))
	for idx, logicalDiskCounter := range logicalDiskCounterValues {
		initializeDoubleDataPoint(ddps.At(2*idx+0), startTime, now, logicalDiskCounter.InstanceName, readDirectionLabelValue, float64(logicalDiskCounter.Values[avgDiskSecsPerRead])/1e7)
		initializeDoubleDataPoint(ddps.At(2*idx+1), startTime, now, logicalDiskCounter.InstanceName, writeDirectionLabelValue, float64(logicalDiskCounter.Values[avgDiskSecsPerWrite])/1e7)
	}
}

func initializeDiskPendingOperationsMetric(metric pdata.Metric, now pdata.TimestampUnixNano, logicalDiskCounterValues []*perfcounters.CounterValues) {
	diskPendingOperationsDescriptor.CopyTo(metric)

	idps := metric.IntSum().DataPoints()
	idps.Resize(len(logicalDiskCounterValues))
	for idx, logicalDiskCounter := range logicalDiskCounterValues {
		initializeDiskPendingDataPoint(idps.At(idx), now, logicalDiskCounter.InstanceName, logicalDiskCounter.Values[queueLength])
	}
}

func initializeInt64DataPoint(dataPoint pdata.IntDataPoint, startTime, now pdata.TimestampUnixNano, deviceLabel string, directionLabel string, value int64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(deviceLabelName, deviceLabel)
	labelsMap.Insert(directionLabelName, directionLabel)
	dataPoint.SetStartTime(startTime)
	dataPoint.SetTimestamp(now)
	dataPoint.SetValue(value)
}

func initializeDoubleDataPoint(dataPoint pdata.DoubleDataPoint, startTime, now pdata.TimestampUnixNano, deviceLabel string, directionLabel string, value float64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(deviceLabelName, deviceLabel)
	labelsMap.Insert(directionLabelName, directionLabel)
	dataPoint.SetStartTime(startTime)
	dataPoint.SetTimestamp(now)
	dataPoint.SetValue(value)
}

func initializeDiskPendingDataPoint(dataPoint pdata.IntDataPoint, now pdata.TimestampUnixNano, deviceLabel string, value int64) {
	labelsMap := dataPoint.LabelsMap()
	labelsMap.Insert(deviceLabelName, deviceLabel)
	dataPoint.SetTimestamp(now)
	dataPoint.SetValue(value)
}
