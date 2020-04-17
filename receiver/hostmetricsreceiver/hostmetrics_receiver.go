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

package hostmetricsreceiver

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opencensus.io/trace"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/component/componenterror"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdatautil"
	"github.com/open-telemetry/opentelemetry-collector/internal/data"
	hmcomponent "github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/component"
)

// Receiver is the type used to handle metrics from VM metrics.
type Receiver struct {
	consumer consumer.MetricsConsumer

	tickerFn getTickerC

	config   *Config
	scrapers []hmcomponent.Scraper

	startOnce sync.Once
	stopOnce  sync.Once

	done chan struct{}
}

type getTickerC func() <-chan time.Time

// NewHostMetricsReceiver creates a new set of VM and Process Metrics
func NewHostMetricsReceiver(
	logger *zap.Logger,
	config *Config,
	factories map[string]hmcomponent.ScraperFactory,
	consumer consumer.MetricsConsumer,
	tickerFn getTickerC,
) (*Receiver, error) {

	scrapers := make([]hmcomponent.Scraper, 0)
	for key, cfg := range config.Scrapers {
		scraper, err := factories[key].CreateMetricsScraper(logger, cfg)
		if err != nil {
			return nil, fmt.Errorf("cannot create scraper: %s", err.Error())
		}
		scrapers = append(scrapers, scraper)
	}

	if tickerFn == nil {
		tickerFn = func() <-chan time.Time { return time.NewTicker(config.ScrapeInterval).C }
	}

	hmr := &Receiver{
		consumer: consumer,
		tickerFn: tickerFn,
		config:   config,
		scrapers: scrapers,
		done:     make(chan struct{}),
	}

	return hmr, nil
}

// Start scrapes Host metrics based on the OS platform.
func (hmr *Receiver) Start(ctx context.Context, host component.Host) error {
	var err = componenterror.ErrAlreadyStarted
	hmr.startOnce.Do(func() {
		err = hmr.initializeScrapers()
		if err != nil {
			return
		}

		go func() {
			tickerC := hmr.tickerFn()
			for {
				select {
				case <-tickerC:
					hmr.scrapeAndExport()

				case <-hmr.done:
					return
				}
			}
		}()

		err = nil
	})
	return err
}

// Shutdown stops and cancels the underlying Host metrics scrapers.
func (hmr *Receiver) Shutdown(context.Context) error {
	var err = componenterror.ErrAlreadyStopped
	hmr.stopOnce.Do(func() {
		defer close(hmr.done)
		err = hmr.closeScrapers()
	})
	return err
}

func (hmr *Receiver) initializeScrapers() error {
	for _, scraper := range hmr.scrapers {
		err := scraper.Initialize()
		if err != nil {
			return err
		}
	}

	return nil
}

func (hmr *Receiver) closeScrapers() error {
	var errs []error

	for _, scraper := range hmr.scrapers {
		err := scraper.Close()
		if err != nil {
			errs = append(errs, err)
			continue
		}
	}

	if len(errs) > 0 {
		return componenterror.CombineErrors(errs)
	}

	return nil
}

func (hmr *Receiver) scrapeAndExport() {
	ctx, span := trace.StartSpan(context.Background(), "hostmetricsreceiver.scrapeAndExport")
	defer span.End()

	var errs []error
	metricData := data.NewMetricData()
	metrics := initializeMetricSlice(metricData)

	for _, scraper := range hmr.scrapers {
		err := scraper.ScrapeMetrics(ctx, metrics)
		if err != nil {
			errs = append(errs, err)
			continue
		}
	}

	if len(errs) > 0 {
		span.SetStatus(trace.Status{Code: trace.StatusCodeDataLoss, Message: fmt.Sprintf("Error(s) when scraping host metrics: %v", componenterror.CombineErrors(errs))})
	}

	if metrics.Len() > 0 {
		err := hmr.consumer.ConsumeMetrics(ctx, pdatautil.MetricsFromInternalMetrics(metricData))
		if err != nil {
			span.SetStatus(trace.Status{Code: trace.StatusCodeDataLoss, Message: fmt.Sprintf("Unable to process metrics: %v", err)})
			return
		}
	}
}

func initializeMetricSlice(metricData data.MetricData) pdata.MetricSlice {
	rms := metricData.ResourceMetrics()
	rms.Resize(1)
	rm := rms.At(0)
	ilms := rm.InstrumentationLibraryMetrics()
	ilms.Resize(1)
	ilm := ilms.At(0)
	return ilm.Metrics()
}
