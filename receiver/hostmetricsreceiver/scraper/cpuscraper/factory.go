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
	"errors"
	"runtime"

	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/component"
)

// This file implements Factory for CPU scraper.

const (
	// The value of "type" key in configuration.
	TypeStr = "cpu"
)

// Factory is the Factory for scraper.
type Factory struct {
}

// Type gets the type of the scraper config created by this Factory.
func (f *Factory) Type() string {
	return TypeStr
}

// CreateDefaultConfig creates the default configuration for the Scraper.
func (f *Factory) CreateDefaultConfig() component.ScraperConfig {
	return &Config{
		ReportPerCPU: true,
	}
}

// CreateMetricsScraper creates a scraper based on provided config.
func (f *Factory) CreateMetricsScraper(
	logger *zap.Logger,
	config component.ScraperConfig,
) (component.Scraper, error) {
	if runtime.GOOS != "windows" {
		return nil, errors.New("cpu scraper is currently only supported on windows")
	}

	cfg := config.(*Config)

	return NewCPUScraper(cfg)
}
