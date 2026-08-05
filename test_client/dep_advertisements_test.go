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
	"testing"

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
)

func TestQueryDeploymentAdvertisements(t *testing.T) {
	res, err := depAdvClient.QueryDeploymentAdvertisements(
		t.Context(),
		lib_models.DeploymentAdvertisementsFilter{},
	)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), res)
}

func TestQueryDeploymentAdvertisement(t *testing.T) {
	res, err := depAdvClient.QueryDeploymentAdvertisement(
		t.Context(),
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), res)
}

// only available for deployments
func TestGetDeploymentAdvertisement(t *testing.T) {
	res, err := depAdvClient.GetDeploymentAdvertisement(
		t.Context(),
		"",
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), res)
}

// only available for deployments
func TestGetDeploymentAdvertisementById(t *testing.T) {
	res, err := depAdvClient.GetDeploymentAdvertisementById(
		t.Context(),
		"",
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), res)
}

// only available for deployments
func TestGetDeploymentAdvertisements(t *testing.T) {
	res, err := depAdvClient.GetDeploymentAdvertisements(
		t.Context(),
		"",
		lib_models.DeploymentAdvertisementsFilterReduced{},
	)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), res)
}

// only available for deployments
func TestPutDeploymentAdvertisement(t *testing.T) {
	res, err := depAdvClient.PutDeploymentAdvertisement(
		t.Context(),
		"",
		"",
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), res)
}

// only available for deployments
func TestPutDeploymentAdvertisements(t *testing.T) {
	res, err := depAdvClient.PutDeploymentAdvertisements(
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
func TestDeleteDeploymentAdvertisement(t *testing.T) {
	err := depAdvClient.DeleteDeploymentAdvertisement(
		t.Context(),
		"",
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
}

// only available for deployments
func TestDeleteDeploymentAdvertisements(t *testing.T) {
	err := depAdvClient.DeleteDeploymentAdvertisements(
		t.Context(),
		"",
		lib_models.DeploymentAdvertisementsFilterReduced{},
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
}
