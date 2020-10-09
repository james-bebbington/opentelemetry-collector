// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package windowsperfcountersreceiver

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.opentelemetry.io/collector/receiver/windowsperfcountersreceiver/internal/third_party/telegraf/win_perf_counters"
)

type mockPerfCounter struct {
	path      string
	scrapeErr error
	closeErr  error
}

func newMockPerfCounter(path string, scrapeErr, closeErr error) *mockPerfCounter {
	return &mockPerfCounter{path: path, scrapeErr: scrapeErr, closeErr: closeErr}
}

// Path
func (mpc *mockPerfCounter) Path() string {
	return mpc.path
}

// ScrapeData
func (mpc *mockPerfCounter) ScrapeData() ([]win_perf_counters.CounterValue, error) {
	return []win_perf_counters.CounterValue{{Value: 0}}, mpc.scrapeErr
}

// Close
func (mpc *mockPerfCounter) Close() error {
	return mpc.closeErr
}

func Test_WindowsPerfCounterScraper(t *testing.T) {
	type expectedMetric struct {
		name                string
		instanceLabelValues []string
	}

	type testCase struct {
		name string
		cfg  *Config

		newErr          string
		initializeErr   string
		mockCounterPath string
		scrapeErr       error
		closeErr        error

		expectedMetrics []expectedMetric
	}

	defaultConfig := &Config{
		PerfCounters:    []PerfCounterConfig{{Object: "Memory", Counters: []string{"Committed Bytes"}}},
		ScraperSettings: receiverhelper.ScraperSettings{CollectionIntervalVal: time.Minute},
	}

	testCases := []testCase{
		{
			name: "Standard",
			cfg: &Config{
				PerfCounters: []PerfCounterConfig{
					{Object: "Memory", Counters: []string{"Committed Bytes"}},
					{Object: "Processor", Instances: []string{"*"}, Counters: []string{"% Processor Time"}},
					{Object: "Processor", Instances: []string{"1", "2"}, Counters: []string{"% Idle Time"}},
				},
				ScraperSettings: receiverhelper.ScraperSettings{CollectionIntervalVal: time.Minute},
			},
			expectedMetrics: []expectedMetric{
				{name: `\Memory\Committed Bytes`},
				{name: `\Processor(*)\% Processor Time`, instanceLabelValues: []string{"*"}},
				{name: `\Processor(1)\% Idle Time`, instanceLabelValues: []string{"1"}},
				{name: `\Processor(2)\% Idle Time`, instanceLabelValues: []string{"2"}},
			},
		},
		{
			name: "InvalidCounter",
			cfg: &Config{
				PerfCounters: []PerfCounterConfig{
					{
						Object:   "Invalid Object",
						Counters: []string{"Invalid Counter"},
					},
				},
				ScraperSettings: receiverhelper.ScraperSettings{CollectionIntervalVal: time.Minute},
			},
			initializeErr: "error initializing counter \\Invalid Object\\Invalid Counter: The specified object was not found on the computer.\r\n",
		},
		{
			name:      "ScrapeError",
			scrapeErr: errors.New("err1"),
		},
		{
			name:            "CloseError",
			expectedMetrics: []expectedMetric{{name: ""}},
			closeErr:        errors.New("err1"),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			cfg := test.cfg
			if cfg == nil {
				cfg = defaultConfig
			}
			scraper, err := newScraper(cfg)
			if test.newErr != "" {
				assert.EqualError(t, err, test.newErr)
				return
			}

			err = scraper.initialize(context.Background())
			if test.initializeErr != "" {
				assert.EqualError(t, err, test.initializeErr)
				return
			}
			require.NoError(t, err)

			if test.mockCounterPath != "" || test.scrapeErr != nil || test.closeErr != nil {
				for i := range scraper.counters {
					scraper.counters[i] = newMockPerfCounter(test.mockCounterPath, test.scrapeErr, test.closeErr)
				}
			}

			metrics, err := scraper.scrape(context.Background())
			if test.scrapeErr != nil {
				assert.Equal(t, err, test.scrapeErr)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, len(test.expectedMetrics), metrics.Len())
			for i, e := range test.expectedMetrics {
				metric := metrics.At(i)
				assert.Equal(t, e.name, metric.Name())

				ddp := metric.DoubleGauge().DataPoints()

				var allInstances bool
				for _, v := range e.instanceLabelValues {
					if v == "*" {
						allInstances = true
						break
					}
				}

				if allInstances {
					require.GreaterOrEqual(t, ddp.Len(), 1)
				} else {
					expectedDataPoints := 1
					if len(e.instanceLabelValues) > 0 {
						expectedDataPoints = len(e.instanceLabelValues)
					}

					require.Equal(t, expectedDataPoints, ddp.Len())
				}

				if len(e.instanceLabelValues) > 0 {
					instanceLabelValues := make([]string, 0, ddp.Len())
					for i := 0; i < ddp.Len(); i++ {
						instanceLabelValue, ok := ddp.At(i).LabelsMap().Get(instanceLabelName)
						require.Truef(t, ok, "data point was missing %q label", instanceLabelName)
						instanceLabelValues = append(instanceLabelValues, instanceLabelValue.Value())
					}

					if !allInstances {
						for _, v := range e.instanceLabelValues {
							assert.Contains(t, instanceLabelValues, v)
						}
					}
				}
			}

			err = scraper.close(context.Background())
			if test.closeErr != nil {
				assert.Equal(t, err, test.closeErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
