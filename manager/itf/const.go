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

package itf

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
	SerialPortResource  ResourceType = "serial-port"
	UdsSocketResource   ResourceType = "uds-socket"
	CertFileResource    ResourceType = "cert-file"
	KeyFileResource     ResourceType = "key-file"
	NetworkNodeResource ResourceType = "network-node"
)

var MountResourceTypeMap = map[string]ResourceType{
	string(SerialPortResource): SerialPortResource,
	string(UdsSocketResource):  UdsSocketResource,
	string(CertFileResource):   CertFileResource,
	string(KeyFileResource):    KeyFileResource,
}

var LinkResourceTypeMap = map[string]ResourceType{
	string(NetworkNodeResource): NetworkNodeResource,
}

const (
	Greater      VersionOperator = ">"
	Less         VersionOperator = "<"
	Equal        VersionOperator = "="
	GreaterEqual VersionOperator = ">="
	LessEqual    VersionOperator = "<="
)

var OperatorMap = map[string]VersionOperator{
	string(Greater):      Greater,
	string(Less):         Less,
	string(Equal):        Equal,
	string(GreaterEqual): GreaterEqual,
	string(LessEqual):    LessEqual,
}
