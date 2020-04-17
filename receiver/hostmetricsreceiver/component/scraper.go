// Copyright  OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package component

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
)

// Scraper gathers metrics from the host machine and converts
// these into internal metrics format.
type Scraper interface {
	// Initialize performs any timely initialization tasks such as
	// setting up performance counters for initial collection.
	Initialize() error
	// Close cleans up any unmanaged resources such as performance
	// counter handles.
	Close() error
	// ScrapeMetrics appends scraped metrics to the provided metric slice
	ScrapeMetrics(ctx context.Context, metrics pdata.MetricSlice) error
}

// ScraperFactory can create a Scraper.
type ScraperFactory interface {
	component.Factory

	// CreateDefaultConfig creates the default configuration for the Scraper.
	CreateDefaultConfig() ScraperConfig

	// CreateMetricsScraper creates a scraper based on this config.
	// If the config is not valid, error will be returned instead.
	CreateMetricsScraper(logger *zap.Logger,
		cfg ScraperConfig) (Scraper, error)
}

// ScraperConfig is the configuration of a scraper.
type ScraperConfig interface {
}

// MakeScraperFactoryMap takes a list of scraper factories and returns a map
// with factory type as keys. It returns a non-nil error when more than one factories
// have the same type.
func MakeScraperFactoryMap(factories ...ScraperFactory) (map[string]ScraperFactory, error) {
	fMap := map[string]ScraperFactory{}
	for _, f := range factories {
		if _, ok := fMap[f.Type()]; ok {
			return fMap, fmt.Errorf("duplicate collector factory %q", f.Type())
		}
		fMap[f.Type()] = f
	}
	return fMap, nil
}
