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

package service

import (
	"fmt"
	"log"
	"syscall"

	"go.uber.org/zap/zapcore"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
)

type WindowsService struct {
	app    *Application
	params Parameters
}

func NewWindowsService(params Parameters) *WindowsService {
	return &WindowsService{params: params}
}

func (s *WindowsService) Execute(args []string, requests <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	if len(args) == 0 {
		log.Fatal("expected first argument supplied to service.Execute to be the service name")
	}

	changes <- svc.Status{State: svc.StartPending}
	s.start(args[0], s.params)
	changes <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}

	for req := range requests {
		switch req.Cmd {
		case svc.Interrogate:
			changes <- req.CurrentStatus
		case svc.Stop, svc.Shutdown:
			s.stop()
			return
		default:
			log.Fatalf(fmt.Sprintf("unexpected control request #%d", req))
		}
	}

	return
}

func (s *WindowsService) start(logSourceName string, params Parameters) {
	var err error
	s.app, err = newWithEventViewerLoggingHook(logSourceName, params)
	if err != nil {
		log.Fatal(err)
	}

	// app.Start blocks until receiving a SIGTERM signal, so we need to start it asynchronously
	go func() {
		err = s.app.Start()
		if err != nil {
			log.Fatalf("application run finished with error: %v", err)
		}
	}()

	// wait until the app is in the Running state
	for state := range s.app.GetStateChannel() {
		if state == Running {
			break
		}
	}
}

func (s *WindowsService) stop() {
	// simulate a SIGTERM signal to terminate the application
	s.app.signalsChannel <- syscall.SIGTERM

	// wait until the app is in the Closed state
	for state := range s.app.GetStateChannel() {
		if state == Closed {
			break
		}
	}

	s.app = nil
}

func newWithEventViewerLoggingHook(serviceName string, params Parameters) (*Application, error) {
	elog, err := eventlog.Open(serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open event log: %v", err)
	}

	params.LoggingHooks = append(
		params.LoggingHooks,
		func(entry zapcore.Entry) error {
			msg := fmt.Sprintf("%v\r\n\r\nStack Trace:\r\n%v", entry.Message, entry.Stack)

			switch entry.Level {
			case zapcore.FatalLevel, zapcore.PanicLevel, zapcore.DPanicLevel:
				// golang.org/x/sys/windows/svc/eventlog does not support Critical level event logs
				return elog.Error(3, msg)
			case zapcore.ErrorLevel:
				return elog.Error(3, msg)
			case zapcore.WarnLevel:
				return elog.Warning(2, msg)
			case zapcore.InfoLevel:
				return elog.Info(1, msg)
			}

			// ignore Debug level logs
			return nil
		},
	)

	app, err := New(params)
	if err != nil {
		return nil, err
	}

	return app, nil
}
