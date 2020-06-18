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

// +build windows

package main

import (
	"log"

	"golang.org/x/sys/windows/svc"

	"go.opentelemetry.io/collector/service"
)

func main() {
	params, err := applicationParameters()
	if err != nil {
		log.Fatal(err)
	}

	isInteractive, err := svc.IsAnInteractiveSession()
	if err != nil {
		log.Fatalf("failed to determine if we are running in an interactive session: %v", err)
	}

	if isInteractive {
		runInteractive(params)
	} else {
		runService(params)
	}
}

func runService(params service.Parameters) {
	err := service.RunWindowsService(params)
	if err != nil {
		log.Fatalf("failed to start service: %v", err)
	}
}
