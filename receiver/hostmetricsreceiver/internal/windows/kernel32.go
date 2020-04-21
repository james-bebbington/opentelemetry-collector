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

// +build windows

package windows

import (
	"syscall"
)

type SYSTEMTIME struct {
	wYear         uint16
	wMonth        uint16
	wDayOfWeek    uint16
	wDay          uint16
	wHour         uint16
	wMinute       uint16
	wSecond       uint16
	wMilliseconds uint16
}

type FILETIME struct {
	dwLowDateTime  uint32
	dwHighDateTime uint32
}

var (
	// Library
	libkrnDll *syscall.DLL

	// Functions
	krn_FileTimeToSystemTime    *syscall.Proc
	krn_FileTimeToLocalFileTime *syscall.Proc
	krn_LocalFileTimeToFileTime *syscall.Proc
	krn_WideCharToMultiByte     *syscall.Proc
)

func init() {
	libkrnDll = syscall.MustLoadDLL("Kernel32.dll")

	krn_FileTimeToSystemTime = libkrnDll.MustFindProc("FileTimeToSystemTime")
	krn_FileTimeToLocalFileTime = libkrnDll.MustFindProc("FileTimeToLocalFileTime")
	krn_LocalFileTimeToFileTime = libkrnDll.MustFindProc("LocalFileTimeToFileTime")
	krn_WideCharToMultiByte = libkrnDll.MustFindProc("WideCharToMultiByte")
}
