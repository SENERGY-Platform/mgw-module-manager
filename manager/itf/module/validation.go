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

import (
	"golang.org/x/mod/semver"
	"strings"
)

func IsValidModuleType(s string) bool {
	_, ok := ModuleTypeMap[s]
	return ok
}

func IsValidDeploymentType(s string) bool {
	_, ok := DeploymentTypeMap[s]
	return ok
}

func IsValidModuleID(s string) bool {
	if !strings.Contains(s, "/") || strings.Contains(s, "//") || strings.HasPrefix(s, "/") {
		return false
	}
	return true
}

func IsValidSrvDepCondition(s string) bool {
	_, ok := SrvDepConditionMap[s]
	return ok
}

func IsValidRestartStrategy(s string) bool {
	_, ok := RestartStrategyMap[s]
	return ok
}

func IsValidPortType(s string) bool {
	_, ok := PortTypeMap[s]
	return ok
}

func IsValidSemVer(s string) bool {
	return semver.IsValid(s)
}

func IsValidOperator(s string) bool {
	_, ok := OperatorMap[s]
	return ok
}

func ValidateSemVerRange(s string) error {
	_, _, err := semVerRangeParse(s)
	return err
}
