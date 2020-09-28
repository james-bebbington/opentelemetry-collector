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

package windowsperfcountersreceiver

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/consumer/pdata"
	"go.opentelemetry.io/collector/receiver/windowsperfcountersreceiver/internal/pdh"
	"go.opentelemetry.io/collector/receiver/windowsperfcountersreceiver/internal/third_party/telegraf/win_perf_counters"
)

const instanceLabelName = "instance"

type PerfCounterScraper interface {
	// Path returns the counter path
	Path() string
	// ScrapeData collects a measurement and returns the value(s).
	ScrapeData() ([]win_perf_counters.CounterValue, error)
	// Close all counters/handles related to the query and free all associated memory.
	Close() error
}

// scraper is the type that scrapes various host metrics.
type scraper struct {
	cfg      *Config
	counters []PerfCounterScraper
}

func newScraper(cfg *Config) (*scraper, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	s := &scraper{cfg: cfg}
	return s, nil
}

func (s *scraper) initialize(ctx context.Context) error {
	var errors []error

	for _, perfCounterCfg := range s.cfg.PerfCounters {
		for _, counterName := range perfCounterCfg.Counters {
			c, err := pdh.NewPerfCounter(fmt.Sprintf("\\%s\\%s", perfCounterCfg.Object, counterName), true)
			if err != nil {
				errors = append(errors, err)
			} else {
				s.counters = append(s.counters, c)
			}
		}
	}

	return componenterror.CombineErrors(errors)
}

func (s *scraper) close(ctx context.Context) error {
	var errors []error

	for _, counter := range s.counters {
		if err := counter.Close(); err != nil {
			errors = append(errors, err)
		}
	}

	return componenterror.CombineErrors(errors)
}

func (s *scraper) scrape(ctx context.Context) (pdata.MetricSlice, error) {
	metrics := pdata.NewMetricSlice()

	now := pdata.TimestampUnixNano(uint64(time.Now().UnixNano()))

	var errors []error

	metrics.Resize(len(s.counters))
	idx := 0
	for _, counter := range s.counters {
		counterValues, err := counter.ScrapeData()
		if err != nil {
			errors = append(errors, err)
		}

		initializeDoubleGaugeMetric(metrics.At(idx), now, counter.Path(), counterValues)
		idx++
	}
	metrics.Resize(len(s.counters) - len(errors))

	if len(errors) > 0 {
		return metrics, componenterror.CombineErrors(errors)
	}

	return metrics, nil
}

func initializeDoubleGaugeMetric(metric pdata.Metric, now pdata.TimestampUnixNano, name string, counterValues []win_perf_counters.CounterValue) {
	metric.SetName(name)
	metric.SetDataType(pdata.MetricDataTypeDoubleGauge)

	dg := metric.DoubleGauge()
	dg.InitEmpty()

	ddps := dg.DataPoints()
	ddps.Resize(len(counterValues))
	for i, counterValue := range counterValues {
		initializeDoubleDataPoint(ddps.At(i), now, counterValue.InstanceName, counterValue.Value)
	}
}

func initializeDoubleDataPoint(dataPoint pdata.DoubleDataPoint, now pdata.TimestampUnixNano, instanceLabel string, value float64) {
	if instanceLabel != "" {
		labelsMap := dataPoint.LabelsMap()
		labelsMap.Insert(instanceLabelName, instanceLabel)
	}

	dataPoint.SetTimestamp(now)
	dataPoint.SetValue(value)
}
