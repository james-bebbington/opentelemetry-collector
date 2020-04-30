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
	"github.com/shirou/gopsutil/cpu"

	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
)

func initializeCPUSecondsMetric(metric pdata.Metric, startTime pdata.TimestampUnixNano, cpuTimes []cpu.TimesStat) {
	MetricCPUSecondsDescriptor.CopyTo(metric.MetricDescriptor())

	idps := metric.Int64DataPoints()
	idps.Resize(4 * len(cpuTimes))
	for i, cpuTime := range cpuTimes {
		initializeCPUSecondsDataPoint(idps.At(4*i+0), startTime, cpuTime.CPU, UserStateLabelValue, int64(cpuTime.User))
		initializeCPUSecondsDataPoint(idps.At(4*i+1), startTime, cpuTime.CPU, SystemStateLabelValue, int64(cpuTime.System))
		initializeCPUSecondsDataPoint(idps.At(4*i+2), startTime, cpuTime.CPU, IdleStateLabelValue, int64(cpuTime.Idle))
		initializeCPUSecondsDataPoint(idps.At(4*i+3), startTime, cpuTime.CPU, InterruptStateLabelValue, int64(cpuTime.Irq))
	}
}
