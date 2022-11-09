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
	TextData  = "https://schema.org/Text"
	BoolData  = "https://schema.org/Boolean"
	IntData   = "https://schema.org/Integer"
	FloatData = "https://schema.org/Float"
)

var DataTypeMap = map[string]struct{}{
	TextData:  {},
	BoolData:  {},
	IntData:   {},
	FloatData: {},
}
