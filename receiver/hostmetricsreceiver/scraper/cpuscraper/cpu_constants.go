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
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
)

// cpu metric constants.

var (
	StateLabel       = "state"
	CPULabel         = "cpu"
	ProcessNameLabel = "process_name"
)

var (
	UserStateLabelValue      = "user"
	SystemStateLabelValue    = "system"
	IdleStateLabelValue      = "idle"
	InterruptStateLabelValue = "interrupt"
	IowaitStateLabelValue    = "iowait"
)

func InitializeMetricCPUSecondsDescriptor(descriptor pdata.MetricDescriptor) pdata.MetricDescriptor {
	descriptor.InitEmpty()
	descriptor.SetName("host/cpu/time")
	descriptor.SetDescription("Total CPU ticks or jiffies broken down by different states")
	descriptor.SetUnit("1")
	descriptor.SetType(pdata.MetricTypeCounterInt64)
	descriptor.LabelsMap().InitFromMap(map[string]string{
		StateLabel: "State of CPU time, e.g. user, system, idle",
		CPULabel:   "CPU Logical Number, e.g. 0, 1, 2",
	})
	return descriptor
}

func InitializeMetricCPUUtilizationDescriptor(descriptor pdata.MetricDescriptor) pdata.MetricDescriptor {
	descriptor.InitEmpty()
	descriptor.SetName("host/cpu/utilization")
	descriptor.SetDescription("Percent of processor utilized per process")
	descriptor.SetUnit("s")
	descriptor.SetType(pdata.MetricTypeGaugeDouble)
	descriptor.LabelsMap().InitFromMap(map[string]string{
		CPULabel: "CPU Logical Number, e.g. 0, 1, 2",
	})
	return descriptor
}

func InitializeMetricProcessUtilizationDescriptor(descriptor pdata.MetricDescriptor) pdata.MetricDescriptor {
	descriptor.InitEmpty()
	descriptor.SetName("process/cpu/utilization")
	descriptor.SetDescription("Percent of processor utilized per process")
	descriptor.SetUnit("1")
	descriptor.SetType(pdata.MetricTypeGaugeDouble)
	descriptor.LabelsMap().InitFromMap(map[string]string{
		ProcessNameLabel: "Name of the process",
	})
	return descriptor
}
