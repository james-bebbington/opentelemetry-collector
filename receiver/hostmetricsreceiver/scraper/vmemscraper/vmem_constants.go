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

import (
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
)

// vmem metric constants.

var (
	StateLabel  = "state"
	LabelType   = "vmem"
	PagingLabel = "process_name"
)

var (
	UserStateLabelValue      = "user"
	SystemStateLabelValue    = "system"
	IdleStateLabelValue      = "idle"
	InterruptStateLabelValue = "interrupt"
	IowaitStateLabelValue    = "iowait"
)

func InitializeMetricVMemSecondsDescriptor(descriptor pdata.MetricDescriptor) pdata.MetricDescriptor {
	descriptor.InitEmpty()
	descriptor.SetName("host/memory/page_faults")
	descriptor.SetUnit("1")
	descriptor.SetType(pdata.MetricTypeCounterInt64)
	descriptor.LabelsMap().InitFromMap(map[string]string{
		LabelType: "e.g. minor, major",
	})
	return descriptor
}

/*
func InitializeMetricVMemUtilizationDescriptor(descriptor pdata.MetricDescriptor) pdata.MetricDescriptor {
	descriptor.InitEmpty()
	descriptor.SetName("host/memory/paging")
	descriptor.SetUnit("s")
	descriptor.SetType(pdata.MetricTypeDoubleInt64)
	descriptor.LabelsMap().InitFromMap(map[string]string{
		PagingLabel: "e.g. page_in, page_out",
	})
	return descriptor
}
*/

func InitializeMetricVMemUtilizationDescriptor(descriptor pdata.MetricDescriptor) pdata.MetricDescriptor {
	descriptor.InitEmpty()
	descriptor.SetName("host/swap/paging")
	descriptor.SetUnit("s")
	descriptor.SetType(pdata.MetricTypeDoubleInt64)
	descriptor.LabelsMap().InitFromMap(map[string]string{
		PagingLabel: "e.g. page_in, page_out",
	})
	return descriptor
}

func InitializeMetricProcessUtilizationDescriptor(descriptor pdata.MetricDescriptor) pdata.MetricDescriptor {
	descriptor.InitEmpty()
	descriptor.SetName("host/memory/pages")
	descriptor.SetUnit("1")
	descriptor.SetType(pdata.MetricTypeGaugeDouble)
	descriptor.LabelsMap().InitFromMap(map[string]string{
		StateLabel: "e.g. free, mapped",
	})
	return descriptor
}
