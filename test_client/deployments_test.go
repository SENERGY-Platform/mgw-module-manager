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

func TestGetDeploymentRequest(t *testing.T) {
	depReq, err := client.GetDeploymentRequest(
		t.Context(),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), depReq)
}

func TestCreateDeployments(t *testing.T) {
	job, err := client.CreateDeployments(
		t.Context(),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(job.Id)
	_, err = client.AwaitJob(t.Context(), job.Id)
	if err != nil {
		t.Fatal(err)
	}
	res, err := client.GetDeploymentsJobResult(t.Context(), job.Id)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), res)
}

func TestUpdateDeployments(t *testing.T) {
	job, err := client.UpdateDeployments(
		t.Context(),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(job.Id)
	_, err = client.AwaitJob(t.Context(), job.Id)
	if err != nil {
		t.Fatal(err)
	}
	res, err := client.GetUpdateDeploymentsJobResult(t.Context(), job.Id)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), res)
}

func TestRecreateDeployments(t *testing.T) {
	job, err := client.RecreateDeployments(
		t.Context(),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(job.Id)
	_, err = client.AwaitJob(t.Context(), job.Id)
	if err != nil {
		t.Fatal(err)
	}
	res, err := client.GetDeploymentsJobResult(t.Context(), job.Id)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), res)
}

func TestDeleteDeployments(t *testing.T) {
	job, err := client.DeleteDeployments(
		t.Context(),
		nil,
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(job.Id)
	_, err = client.AwaitJob(t.Context(), job.Id)
	if err != nil {
		t.Fatal(err)
	}
	res, err := client.GetDeleteDeploymentsJobResult(t.Context(), job.Id)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), res)
}

func TestEnableDeployments(t *testing.T) {
	res, err := client.EnableDeployments(
		t.Context(),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), res)
}

func TestDisableDeployments(t *testing.T) {
	res, err := client.DisableDeployments(
		t.Context(),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), res)
}

func TestDeploymentsHealth(t *testing.T) {
	depReq, err := client.DeploymentsHealth(
		t.Context(),
		lib_models.DeploymentsHealthInfoFilter{},
	)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), depReq)
}
