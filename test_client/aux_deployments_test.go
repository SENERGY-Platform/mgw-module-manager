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
	"time"

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
)

func TestGetAuxiliaryDeployment(t *testing.T) {
	res, err := auxDepClient.GetAuxiliaryDeployment(
		t.Context(),
		"",
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), res)
}

func TestGetAuxiliaryDeployments(t *testing.T) {
	res, err := auxDepClient.GetAuxiliaryDeployments(
		t.Context(),
		"",
		lib_models.AuxiliaryDeploymentsFilterWithState{},
	)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), res)
}

func TestGetReducedAuxiliaryDeployments(t *testing.T) {
	res, err := auxDepClient.GetReducedAuxiliaryDeployments(
		t.Context(),
		"",
		lib_models.AuxiliaryDeploymentsFilterWithState{},
	)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), res)
}

func TestGetAuxiliaryDeploymentVolumesWithMounts(t *testing.T) {
	res, err := auxDepClient.GetAuxiliaryDeploymentVolumesWithMounts(
		t.Context(),
		"",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), res)
}

// only available for deployments
func TestCreateAuxiliaryDeployment(t *testing.T) {
	job, err := auxDepClient.CreateAuxiliaryDeployment(
		t.Context(),
		"",
		lib_models.AuxiliaryDeploymentInput{},
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(job.Id)
	_, err = auxDepClient.AwaitJob(t.Context(), job.Id, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	res, err := auxDepClient.GetCreateAuxiliaryDeploymentJobResult(t.Context(), job.Id)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), res)
}

// only available for deployments
func TestUpdateAuxiliaryDeployment(t *testing.T) {
	job, err := auxDepClient.UpdateAuxiliaryDeployment(
		t.Context(),
		"",
		"",
		lib_models.AuxiliaryDeploymentInput{},
		false,
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(job.Id)
	_, err = auxDepClient.AwaitJob(t.Context(), job.Id, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	res, err := auxDepClient.GetUpdateAuxiliaryDeploymentJobResult(t.Context(), job.Id)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), res)
}

// only available for deployments
func TestRecreateAuxiliaryDeployments(t *testing.T) {
	job, err := auxDepClient.RecreateAuxiliaryDeployments(
		t.Context(),
		"",
		lib_models.AuxiliaryDeploymentsFilterWithState{},
	)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(job.Id)
	_, err = auxDepClient.AwaitJob(t.Context(), job.Id, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	res, err := auxDepClient.GetAuxiliaryDeploymentsJobResult(t.Context(), job.Id)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), res)
}

// only available for deployments
func TestDeleteAuxiliaryDeployment(t *testing.T) {
	err := auxDepClient.DeleteAuxiliaryDeployment(
		t.Context(),
		"",
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
}

// only available for deployments
func TestDeleteAuxiliaryDeployments(t *testing.T) {
	job, err := auxDepClient.DeleteAuxiliaryDeployments(
		t.Context(),
		"",
		lib_models.AuxiliaryDeploymentsFilterWithState{},
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(job.Id)
	_, err = auxDepClient.AwaitJob(t.Context(), job.Id, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	res, err := auxDepClient.GetAuxiliaryDeploymentsJobResult(t.Context(), job.Id)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), res)
}

// only available for deployments
func TestEnableAuxiliaryDeployments(t *testing.T) {
	res, err := auxDepClient.EnableAuxiliaryDeployments(
		t.Context(),
		"",
		lib_models.AuxiliaryDeploymentsFilterWithState{},
	)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), res)
}

// only available for deployments
func TestDisableAuxiliaryDeployments(t *testing.T) {
	res, err := auxDepClient.DisableAuxiliaryDeployments(
		t.Context(),
		"",
		lib_models.AuxiliaryDeploymentsFilterWithState{},
	)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), res)
}

// only available for deployments
func TestGetAuxiliaryDeploymentVolumes(t *testing.T) {
	res, err := auxDepClient.GetAuxiliaryDeploymentVolumes(
		t.Context(),
		"",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), res)
}

// only available for deployments
func TestDeleteAuxiliaryDeploymentVolume(t *testing.T) {
	err := auxDepClient.DeleteAuxiliaryDeploymentVolume(
		t.Context(),
		"",
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
}

// only available for deployments
func TestDeleteAuxiliaryDeploymentVolumes(t *testing.T) {
	res, err := auxDepClient.DeleteAuxiliaryDeploymentVolumes(
		t.Context(),
		"",
		nil,
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), res)
}

// only available for deployments
func TestDeleteUnusedAuxiliaryDeploymentVolumes(t *testing.T) {
	res, err := auxDepClient.DeleteUnusedAuxiliaryDeploymentVolumes(
		t.Context(),
		"",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), res)
}
