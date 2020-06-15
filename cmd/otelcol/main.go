// Copyright The OpenTelemetry Authors
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

// Program otelcol is the OpenTelemetry Collector that collects stats
// and traces and exports to a configured backend.
package main

import (
	"fmt"
	"log"

	"go.opentelemetry.io/collector/internal/version"
	"go.opentelemetry.io/collector/service"
	"go.opentelemetry.io/collector/service/defaultcomponents"
)

func applicationParameters() (service.Parameters, error) {
	factories, err := defaultcomponents.Components()
	if err != nil {
		return service.Parameters{}, fmt.Errorf("failed to build default components: %v", err)
	}

	info := service.ApplicationStartInfo{
		ExeName:  "otelcol",
		LongName: "OpenTelemetry Collector",
		Version:  version.Version,
		GitHash:  version.GitHash,
	}

	return service.Parameters{Factories: factories, ApplicationStartInfo: info}, nil
}

func runInteractive(params service.Parameters) {
	app, err := service.New(params)
	if err != nil {
		log.Fatalf("failed to construct the application: %v", err)
	}

	err = app.Start()
	if err != nil {
		log.Fatalf("application run finished with error: %v", err)
	}
}
