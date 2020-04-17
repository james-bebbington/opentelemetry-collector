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

package internal

import (
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/stretchr/testify/assert"

	"github.com/open-telemetry/opentelemetry-collector/consumer/pdata"
)

func TestAddMetric(t *testing.T) {
	metrics := pdata.NewMetricSlice()

	metric1 := AddMetric(metrics)

	assert.Equal(t, 1, metrics.Len())
	assert.Equal(t, metric1, metrics.At(0))

	metric2 := AddMetric(metrics)

	assert.Equal(t, 2, metrics.Len())
	assert.Equal(t, metric1, metrics.At(0))
	assert.Equal(t, metric2, metrics.At(0))
}

func TestTimeConverters(t *testing.T) {
	// Ensure that nanoseconds are also preserved.
	t1 := time.Date(2018, 10, 31, 19, 43, 35, 789, time.UTC)

	assert.EqualValues(t, int64(1541015015000000789), t1.UnixNano())
	tp := TimeToTimestamp(t1)
	assert.EqualValues(t, &timestamp.Timestamp{Seconds: 1541015015, Nanos: 789}, tp)
	assert.EqualValues(t, int64(1541015015000000789), TimestampToTime(tp).UnixNano())
}
