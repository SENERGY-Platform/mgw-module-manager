/*
 * Copyright 2026 InfAI (CC SES)
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

package test_client

import (
	"fmt"
	"testing"

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
)

func TestGetModules(t *testing.T) {
	modules, err := client.GetModules(
		t.Context(),
		lib_models.ModulesFilter{},
	)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), modules)
}

func TestGetModule(t *testing.T) {
	module, err := client.GetModule(
		t.Context(),
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), module)
}

func TestGetModulesChangeRequest(t *testing.T) {
	changeRequest, err := client.GetModulesChangeRequest(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), changeRequest)
}

func TestCreateModulesChangeRequest(t *testing.T) {
	changeRequest, err := client.CreateModulesChangeRequest(
		t.Context(),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), changeRequest)
}

func TestExecModulesChangeRequest(t *testing.T) {
	job, err := client.ExecModulesChangeRequest(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(job.Id)
	_, err = client.AwaitJob(t.Context(), job.Id)
	if err != nil {
		t.Fatal(err)
	}
	res, err := client.GetModuleChangeJobResult(t.Context(), job.Id)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), res)
}

func TestCancelModulesChangeRequest(t *testing.T) {
	err := client.CancelModulesChangeRequest(t.Context())
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetModulesAvailableUpdatesCount(t *testing.T) {
	c, err := client.GetModulesAvailableUpdatesCount(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), c)
}

func TestCreateModulesUpdateAllChangeRequest(t *testing.T) {
	changeRequest, err := client.CreateModulesUpdateAllChangeRequest(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), changeRequest)
}
