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

package vmemscraper

import "github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/scraper"

// Config relating to VMem Metric Scrapeion.
type Config struct {
	scraper.ConfigBase `mapstructure:"squash"`

	// If `true`, stats will be generated for the system as a whole _as well
	// as_ for each individual VMem/core in the system and will be distinguished
	// by the `vmem` dimension.  If `false`, stats will only be generated for
	// the system as a whole that will not include a `vmem` dimension.
	ReportPerVMem bool `mapstructure:"report_per_vmem"`

	// If `true`, additional stats will be generated for each running process.
	ReportPerProcess bool `mapstructure:"report_per_process"`
}
