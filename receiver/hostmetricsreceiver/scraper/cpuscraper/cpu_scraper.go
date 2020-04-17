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
	"errors"
	"sync"
	"time"

	"go.opencensus.io/trace"

	"github.com/open-telemetry/opentelemetry-collector/component/componenterror"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
)

// Scraper for CPU Metrics
type Scraper struct {
	startTime time.Time

	config *Config

	initializeOnce sync.Once
	closeOnce      sync.Once
}

// NewCPUScraper creates a set of CPU related metrics
func NewCPUScraper(cfg *Config) (*Scraper, error) {
	return &Scraper{config: cfg}, nil
}

// Initialize
func (c *Scraper) Initialize() error {
	var err = componenterror.ErrAlreadyStarted
	c.initializeOnce.Do(func() {
		c.startTime = time.Now()
		err = c.initialize()
	})
	return err
}

// Close
func (c *Scraper) Close() error {
	var err = componenterror.ErrAlreadyStarted
	c.closeOnce.Do(func() {
		err = c.close()
	})
	return err
}

// ScrapeMetrics returns a list of collected CPU related metrics.
func (c *Scraper) ScrapeMetrics(ctx context.Context, metrics pdata.MetricSlice) error {
	if c.startTime.IsZero() {
		return errors.New("cpu scraper has not been initialized")
	}

	ctx, span := trace.StartSpan(ctx, "cpuscraper.ScrapeMetrics")
	defer span.End()

	err := c.scrapePerCoreMetrics(ctx, metrics)
	if err != nil {
		return err
	}

	err = c.scrapePerProcessMetrics(ctx, metrics)
	if err != nil {
		return err
	}

	return nil
}
