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

import "github.com/SENERGY-Platform/mgw-container-engine-manager-lib/cem-lib"

const ModFile = "Modfile"

const (
	AddOnModule           ModuleType = "add-on"
	DeviceConnectorModule ModuleType = "device-connector"
)

var ModuleTypeMap = map[string]ModuleType{
	string(AddOnModule):           AddOnModule,
	string(DeviceConnectorModule): DeviceConnectorModule,
}

const (
	SingleDeployment   DeploymentType = "single"
	MultipleDeployment DeploymentType = "multiple"
)

var DeploymentTypeMap = map[string]DeploymentType{
	string(SingleDeployment):   SingleDeployment,
	string(MultipleDeployment): MultipleDeployment,
}

const (
	RunningCondition = SrvDepCondition(cem_lib.RunningState)
	StoppedCondition = SrvDepCondition(cem_lib.StoppedState)
)

var SrvDepConditionMap = map[string]SrvDepCondition{
	string(RunningCondition): RunningCondition,
	string(StoppedCondition): StoppedCondition,
}

const (
	TextData  DataType = "https://schema.org/Text"
	BoolData  DataType = "https://schema.org/Boolean"
	IntData   DataType = "https://schema.org/Integer"
	FloatData DataType = "https://schema.org/Float"
)

var DataTypeMap = map[string]DataType{
	string(TextData):  TextData,
	string(BoolData):  BoolData,
	string(IntData):   IntData,
	string(FloatData): FloatData,
}
