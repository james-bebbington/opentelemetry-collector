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
	"errors"
	"fmt"
	"runtime"
	"time"

	"github.com/spf13/viper"

	"github.com/open-telemetry/opentelemetry-collector/component"
	"github.com/open-telemetry/opentelemetry-collector/config/configerror"
	"github.com/open-telemetry/opentelemetry-collector/config/configmodels"
	"github.com/open-telemetry/opentelemetry-collector/consumer"
	hmcomponent "github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/component"
)

// This file implements Factory for HostMetrics receiver.

const (
	// The value of "type" key in configuration.
	typeStr = "hostmetrics"
)

// Factory is the Factory for receiver.
type Factory struct {
	ScraperFactories map[string]hmcomponent.ScraperFactory
}

// Type gets the type of the Receiver config created by this Factory.
func (f *Factory) Type() string {
	return typeStr
}

// CustomUnmarshaler returns nil because we don't need custom unmarshaling for this factory.
func (f *Factory) CustomUnmarshaler() component.CustomUnmarshaler {
	return func(componentViperSection *viper.Viper, intoCfg interface{}) error {
		// load the non-dynamic config normally
		err := componentViperSection.UnmarshalExact(intoCfg)
		if err != nil {
			return err
		}

		config, ok := intoCfg.(*Config)
		if !ok {
			return fmt.Errorf("config type not hostmetrics.Config")
		}

		// dynamically load the individual collector configs based on the key name

		config.Scrapers = map[string]hmcomponent.ScraperConfig{}

		scrapersViperSection := componentViperSection.Sub("scrapers")
		if len(scrapersViperSection.AllKeys()) == 0 {
			return fmt.Errorf("must specify at least one scraper when using hostmetrics receiver")
		}

		for key := range componentViperSection.GetStringMap("scrapers") {
			factory, ok := f.ScraperFactories[key]
			if !ok {
				return fmt.Errorf("invalid hostmetrics scraper key: %s", key)
			}

			collectorCfg := factory.CreateDefaultConfig()
			collectorViperSection := scrapersViperSection.Sub(key)
			if collectorViperSection != nil {
				err := collectorViperSection.UnmarshalExact(collectorCfg)
				if err != nil {
					return fmt.Errorf("error reading settings for hostmetric scraper type %q: %v", key, err)
				}
			}

			config.Scrapers[key] = collectorCfg
		}

		return nil
	}
}

// CreateDefaultConfig creates the default configuration for receiver.
func (f *Factory) CreateDefaultConfig() configmodels.Receiver {
	return &Config{
		ReceiverSettings: configmodels.ReceiverSettings{
			TypeVal: typeStr,
			NameVal: typeStr,
		},
		ScrapeInterval: 10 * time.Second,
	}
}

// CreateTraceReceiver creates a trace receiver based on provided config.
func (f *Factory) CreateTraceReceiver(
	ctx context.Context,
	params component.ReceiverCreateParams,
	cfg configmodels.Receiver,
	consumer consumer.TraceConsumer,
) (component.TraceReceiver, error) {
	// Host Metrics does not support traces
	return nil, configerror.ErrDataTypeIsNotSupported
}

// CreateMetricsReceiver creates a metrics receiver based on provided config.
func (f *Factory) CreateMetricsReceiver(
	ctx context.Context,
	params component.ReceiverCreateParams,
	cfg configmodels.Receiver,
	consumer consumer.MetricsConsumer,
) (component.MetricsReceiver, error) {

	if runtime.GOOS != "windows" {
		return nil, errors.New("hostmetrics receiver is currently only supported on windows")
	}

	config := cfg.(*Config)

	hmr, err := NewHostMetricsReceiver(params.Logger, config, f.ScraperFactories, consumer, nil)
	if err != nil {
		return nil, err
	}

	return hmr, nil
}
