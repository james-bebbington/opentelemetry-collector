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

package memoryscraper

import (
	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
)

// memory metric constants.

var (
	StateLabel       = "state"
	ProcessNameLabel = "process_name"
)

var (
	BufferedStateLabelValue          = "buffered"
	CachedStateLabelValue            = "cached"
	SlabReclaimableStateLabelValue   = "slab_reclaimable"
	SlabUnreclaimableStateLabelValue = "slab_unreclaimable"
	SystemStateLabelValue            = "system"
)

func InitializeMetricMemoryUsedDescriptor(descriptor pdata.MetricDescriptor) pdata.MetricDescriptor {
	descriptor.InitEmpty()
	descriptor.SetName("host/memory/used")
	descriptor.SetDescription("Bytes of memory in use by the system.")
	descriptor.SetUnit("bytes")
	descriptor.SetType(pdata.MetricTypeGaugeInt64)
	descriptor.LabelsMap().InitFromMap(map[string]string{
		StateLabel: "State of memory, e.g. system, slab_unreclaimable",
	})
	return descriptor
}

func InitializeMetricMemoryReclaimableDescriptor(descriptor pdata.MetricDescriptor) pdata.MetricDescriptor {
	descriptor.InitEmpty()
	descriptor.SetName("host/memory/reclaimable")
	descriptor.SetDescription("Bytes of memory that can be reclaimed.")
	descriptor.SetUnit("bytes")
	descriptor.SetType(pdata.MetricTypeGaugeInt64)
	descriptor.LabelsMap().InitFromMap(map[string]string{
		StateLabel: "State of memory, e.g. buffered, cached, slab_reclaimable",
	})
	return descriptor
}

func InitializeMetricMemoryTotalDescriptor(descriptor pdata.MetricDescriptor) pdata.MetricDescriptor {
	descriptor.InitEmpty()
	descriptor.SetName("host/memory/total")
	descriptor.SetDescription("Total bytes of memory available.")
	descriptor.SetUnit("bytes")
	descriptor.SetType(pdata.MetricTypeGaugeInt64)
	descriptor.LabelsMap().InitFromMap(map[string]string{})
	return descriptor
}

func InitializeMetricMemoryUtilizationDescriptor(descriptor pdata.MetricDescriptor) pdata.MetricDescriptor {
	descriptor.InitEmpty()
	descriptor.SetName("host/memory/utilization")
	descriptor.SetDescription("Percent of memory utilized")
	descriptor.SetUnit("1")
	descriptor.SetType(pdata.MetricTypeGaugeDouble)
	return descriptor
}

func InitializeMetricProcessUsedDescriptor(descriptor pdata.MetricDescriptor) pdata.MetricDescriptor {
	descriptor.InitEmpty()
	descriptor.SetName("process/memory/used")
	descriptor.SetDescription("Bytes of memory in use by the process.")
	descriptor.SetUnit("bytes")
	descriptor.SetType(pdata.MetricTypeGaugeInt64)
	descriptor.LabelsMap().InitFromMap(map[string]string{
		ProcessNameLabel: "Name of the process",
	})
	return descriptor
}
