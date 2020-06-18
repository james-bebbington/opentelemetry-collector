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

	svc "github.com/kardianos/service"
	"go.uber.org/zap/zapcore"
)

type WindowsService struct {
	app    *Application
	params Parameters
}

func RunWindowsService(params Parameters) error {
	// Can supply any non-empty service name when startup is invoked through Service Control Manager
	// directly, however that name will be returned when the service name is referenced internally
	// (e.g. as the Event Viewer Logger Source)
	s, err := svc.New(&WindowsService{params: params}, &svc.Config{Name: "OpenTelemetry Collector"})
	if err != nil {
		return err
	}

	return s.Run()
}

func (m *WindowsService) Start(s svc.Service) error {
	logger, err := s.Logger(make(chan error, 500))
	if err != nil {
		log.Fatal(err)
	}

	m.app, err = newWithEventViewerLoggingHook(logger, m.params)
	if err != nil {
		log.Fatal(err)
	}

	// app.Start blocks until receiving a SIGTERM signal, so we need to start it asynchronously
	go func() {
		err = m.app.Start()
		if err != nil {
			log.Fatalf("application run finished with error: %v", err)
		}
	}()

	// wait until the app is in the Running state
	for state := range m.app.GetStateChannel() {
		if state == Running {
			break
		}
	}

	return nil
}

func (m *WindowsService) Stop(s svc.Service) error {
	m.app.signalsChannel <- syscall.SIGTERM

	// wait until the app is in the Closed state
	for state := range m.app.GetStateChannel() {
		if state == Closed {
			break
		}
	}

	return nil
}

func newWithEventViewerLoggingHook(logger svc.Logger, params Parameters) (*Application, error) {
	params.LoggingHooks = append(
		params.LoggingHooks,
		func(entry zapcore.Entry) error {
			msg := fmt.Sprintf("%v\r\n\r\nStack Trace:\r\n%v", entry.Message, entry.Stack)

			switch entry.Level {
			case zapcore.FatalLevel, zapcore.PanicLevel, zapcore.DPanicLevel:
				// golang.org/x/sys/windows/svc/eventlog does not support Critical level event logs
				return logger.Error(msg)
			case zapcore.ErrorLevel:
				return logger.Error(msg)
			case zapcore.WarnLevel:
				return logger.Warning(msg)
			case zapcore.InfoLevel:
				return logger.Info(msg)
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
