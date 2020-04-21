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

// Union specialization for double values
type PDH_FMT_COUNTERVALUE_DOUBLE struct {
	CStatus     uint32
	padding     [4]byte
	DoubleValue float64
}

// Union specialization for 64 bit integer values
type PDH_FMT_COUNTERVALUE_LARGE struct {
	CStatus    uint32
	padding    [4]byte
	LargeValue int64
}

// Union specialization for long values
type PDH_FMT_COUNTERVALUE_LONG struct {
	CStatus   uint32
	LongValue int32
	padding   [4]byte
}

type PDH_FMT_COUNTERVALUE_ITEM_DOUBLE struct {
	SzName   *uint16
	padding  [4]byte
	FmtValue PDH_FMT_COUNTERVALUE_DOUBLE
}

// Union specialization for 'large' values, used by PdhGetFormattedCounterArrayLarge()
type PDH_FMT_COUNTERVALUE_ITEM_LARGE struct {
	SzName   *uint16 // pointer to a string
	padding  [4]byte
	FmtValue PDH_FMT_COUNTERVALUE_LARGE
}

// Union specialization for long values, used by PdhGetFormattedCounterArrayLong()
type PDH_FMT_COUNTERVALUE_ITEM_LONG struct {
	SzName   *uint16 // pointer to a string
	padding  [4]byte
	FmtValue PDH_FMT_COUNTERVALUE_LONG
}

//PDH_COUNTER_INFO structure contains information describing the properties of a counter. This information also includes the counter path.
type PDH_COUNTER_INFO struct {
	//Size of the structure, including the appended strings, in bytes.
	DwLength uint32
	//Counter type. For a list of counter types, see the Counter Types section of the <a "href=http://go.microsoft.com/fwlink/p/?linkid=84422">Windows Server 2003 Deployment Kit</a>.
	//The counter type constants are defined in Winperf.h.
	DwType uint32
	//Counter version information. Not used.
	CVersion uint32
	//Counter status that indicates if the counter value is valid. For a list of possible values,
	//see <a href="https://msdn.microsoft.com/en-us/library/windows/desktop/aa371894(v=vs.85).aspx">Checking PDH Interface Return Values</a>.
	CStatus uint32
	//Scale factor to use when computing the displayable value of the counter. The scale factor is a power of ten.
	//The valid range of this parameter is PDH_MIN_SCALE (–7) (the returned value is the actual value times 10–⁷) to
	//PDH_MAX_SCALE (+7) (the returned value is the actual value times 10⁺⁷). A value of zero will set the scale to one, so that the actual value is returned
	LScale int32
	//Default scale factor as suggested by the counter's provider.
	LDefaultScale int32
	//The value passed in the dwUserData parameter when calling PdhAddCounter.
	DwUserData *uint32
	//The value passed in the dwUserData parameter when calling PdhOpenQuery.
	DwQueryUserData *uint32
	//Null-terminated string that specifies the full counter path. The string follows this structure in memory.
	SzFullPath *uint16 // pointer to a string
	//Null-terminated string that contains the name of the computer specified in the counter path. Is NULL, if the path does not specify a computer.
	//The string follows this structure in memory.
	SzMachineName *uint16 // pointer to a string
	//Null-terminated string that contains the name of the performance object specified in the counter path. The string follows this structure in memory.
	SzObjectName *uint16 // pointer to a string
	//Null-terminated string that contains the name of the object instance specified in the counter path. Is NULL, if the path does not specify an instance.
	//The string follows this structure in memory.
	SzInstanceName *uint16 // pointer to a string
	//Null-terminated string that contains the name of the parent instance specified in the counter path. Is NULL, if the path does not specify a parent instance.
	//The string follows this structure in memory.
	SzParentInstance *uint16 // pointer to a string
	//Instance index specified in the counter path. Is 0, if the path does not specify an instance index.
	DwInstanceIndex uint32 // pointer to a string
	//Null-terminated string that contains the counter name. The string follows this structure in memory.
	SzCounterName *uint16 // pointer to a string
	//padding
	Padding [4]byte
	//Help text that describes the counter. Is NULL if the source is a log file.
	SzExplainText *uint16 // pointer to a string
	//Start of the string data that is appended to the structure.
	DataBuffer [1]uint32 // pointer to an extra space
}
