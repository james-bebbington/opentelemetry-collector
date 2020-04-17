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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector/component/componenttest"
	"github.com/open-telemetry/opentelemetry-collector/exporter/exportertest"
	"github.com/open-telemetry/opentelemetry-collector/receiver/hostmetricsreceiver/scraper"
)

func TestGatherMetrics_EndToEnd(t *testing.T) {
	sink := &exportertest.SinkMetricsExporter{}

	config := &Config{
		DefaultCollectionInterval: 0,
		Scrapers:                  map[string]scraper.Config{},
	}

	factories := map[string]scraper.Factory{}

	receiver, err := NewHostMetricsReceiver(context.Background(), zap.NewNop(), config, factories, sink)
	require.NoError(t, err, "Failed to create metrics receiver: %v", err)

	err = receiver.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err, "Failed to start metrics receiver: %v", err)
	defer func() { assert.NoError(t, receiver.Shutdown(context.Background())) }()

	// TODO consider adding mechanism to get notifed when metrics are processed so we
	// dont have to wait here
	<-time.After(110 * time.Millisecond)

	got := sink.AllMetrics()

	// expect 0 MetricData objects
	assert.Equal(t, 0, len(got))
}
