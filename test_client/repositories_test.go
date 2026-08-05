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
	"encoding/json"
	"fmt"
	"testing"

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
)

func TestRefreshRepositories(t *testing.T) {
	job, err := client.RefreshRepositories(
		t.Context(),
		lib_models.RepositoriesRefreshFilter{},
	)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(job.Id)
	_, err = client.AwaitJob(t.Context(), job.Id)
	if err != nil {
		t.Fatal(err)
	}
	res, err := client.GetRefreshRepositoriesJobResult(t.Context(), job.Id)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), res)
}

func TestGetRepositories(t *testing.T) {
	repositories, err := client.GetRepositories(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), repositories)
}

func TestCreateRepository(t *testing.T) {
	b, err := json.Marshal(Source{
		Owner:      "SENERGY-Platform",
		Repository: "mgw-module-repository",
		Reference:  "main-validated",
		Priority:   100,
		Channels: []Channel{
			{
				Name:     "main",
				Priority: 2,
			},
			{
				Name:     "testing",
				Priority: 1,
			},
			{
				Name:     "legacy",
				Priority: 0,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	err = client.CreateRepository(t.Context(), "github.com", b)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDeleteRepository(t *testing.T) {
	err := client.DeleteRepository(
		t.Context(),
		"",
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestGetRepositoryModules(t *testing.T) {
	repoMods, err := client.GetRepositoryModules(
		t.Context(),
		lib_models.RepoModulesFilter{},
	)
	if err != nil {
		t.Fatal(err)
	}
	writeToJson(t.Name(), repoMods)
}

type Source struct {
	Owner      string    `json:"owner"`
	Repository string    `json:"repository"`
	Reference  string    `json:"reference"`
	Priority   int       `json:"priority"`
	Channels   []Channel `json:"channels"`
}

type Channel struct {
	Name      string   `json:"name"`
	Priority  int      `json:"priority"`
	Blacklist []string `json:"blacklist"`
}
