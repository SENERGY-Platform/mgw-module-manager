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

func TestCreateGlobalConfig(t *testing.T) {
	id, err := client.CreateGlobalConfig(
		t.Context(),
		lib_models.GlobalConfigInput{},
	)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), id)
}

func TestGetGlobalConfig(t *testing.T) {
	cfg, err := client.GetGlobalConfig(
		t.Context(),
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), cfg)
}

func TestGetGlobalConfigs(t *testing.T) {
	cfgs, err := client.GetGlobalConfigs(
		t.Context(),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), cfgs)
}

func TestUpdateGlobalConfig(t *testing.T) {
	err := client.UpdateGlobalConfig(
		t.Context(),
		lib_models.GlobalConfig{},
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDeleteGlobalConfig(t *testing.T) {
	err := client.DeleteGlobalConfig(
		t.Context(),
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDeleteGlobalConfigs(t *testing.T) {
	err := client.DeleteGlobalConfigs(
		t.Context(),
		nil,
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
}
