/*
 * Copyright 2022 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package module

const (
	AddOnModule           = "add-on"
	DeviceConnectorModule = "device-connector"
)

var ModuleTypeMap = map[string]struct{}{
	AddOnModule:           {},
	DeviceConnectorModule: {},
}

const (
	SingleDeployment   = "single"
	MultipleDeployment = "multiple"
)

var DeploymentTypeMap = map[string]struct{}{
	SingleDeployment:   {},
	MultipleDeployment: {},
}

const (
	RunningCondition = "running"
	StoppedCondition = "stopped"
)

var SrvDepConditionMap = map[string]struct{}{
	RunningCondition: {},
	StoppedCondition: {},
}

const (
	Greater      = ">"
	Less         = "<"
	Equal        = "="
	GreaterEqual = ">="
	LessEqual    = "<="
)

var OperatorMap = map[string]struct{}{
	Greater:      {},
	Less:         {},
	Equal:        {},
	GreaterEqual: {},
	LessEqual:    {},
}

const (
	RestartNever      = "never"
	RestartAlways     = "always"
	RestartNotStopped = "not-stopped"
	RestartOnFail     = "on-fail"
)

var RestartStrategyMap = map[string]struct{}{
	RestartNever:      {},
	RestartAlways:     {},
	RestartNotStopped: {},
	RestartOnFail:     {},
}

const (
	TcpPort = "tcp"
	UdpPort = "udp"
)

var PortTypeMap = map[string]struct{}{
	TcpPort: {},
	UdpPort: {},
}

const (
	Bool DataType = iota
	Int64
	Float64
	String
)

var DataTypeRef = []string{
	Bool:    "bool",
	Int64:   "int",
	Float64: "float",
	String:  "string",
}

var DataTypeRefMap = func() map[string]DataType {
	m := make(map[string]DataType)
	for i := 0; i < len(DataTypeRef); i++ {
		m[DataTypeRef[i]] = DataType(i)
	}
	return m
}()

const (
	Text ConfigType = iota
	Date
	Time
	Number
	Toggle
)

var ConfigTypeRef = []string{
	Text:   "text",
	Date:   "date",
	Time:   "time",
	Number: "number",
	Toggle: "toggle",
}

var ConfigTypeRefMap = func() map[string]ConfigType {
	m := make(map[string]ConfigType)
	for i := 0; i < len(ConfigTypeRef); i++ {
		m[ConfigTypeRef[i]] = ConfigType(i)
	}
	return m
}()
