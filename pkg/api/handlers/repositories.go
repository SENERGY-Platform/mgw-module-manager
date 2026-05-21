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

package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/service"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

func RefreshRepositories(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodPatch, "repositories", func(gc *gin.Context) {
		res, err := srv.RefreshRepositories(gc)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func getGetRepositoryModulesFilter(gc *gin.Context) (lib_models.RepoModulesFilter, error) {
	var query struct {
		Ids                []string `form:"ids" collection_format:"csv"`
		Name               string   `form:"name"`
		Repositories       []string `form:"repositories" collection_format:"csv"`
		RepositoryChannels []string `form:"repository_channels" collection_format:"csv"` // ITEM FORMAT -> source|channel
		Installed          bool     `form:"installed"`
		UpdateAvailable    bool     `form:"update_available"`
	}
	err := gc.MustBindWith(&query, binding.Query)
	if err != nil {
		return lib_models.RepoModulesFilter{}, err
	}
	repositoryFilter, err := getRepositoryFilter(query.Repositories, query.RepositoryChannels)
	if err != nil {
		gc.AbortWithError(http.StatusBadRequest, err)
		return lib_models.RepoModulesFilter{}, err
	}
	return lib_models.RepoModulesFilter{
		Ids:             query.Ids,
		Name:            query.Name,
		Repositories:    repositoryFilter,
		Installed:       query.Installed,
		UpdateAvailable: query.UpdateAvailable,
	}, nil
}

func getRepositoryFilter(sources, sourceChannels []string) ([]lib_models.RepositoryFilter, error) {
	repoChannels := make(map[string][]string)
	for _, item := range sourceChannels {
		if item == "" {
			continue
		}
		parts := strings.Split(item, "|")
		if len(parts) != 2 {
			return nil, errors.New(fmt.Sprintf("invalid repository channel format: %s", item))
		}
		repoChannels[parts[0]] = append(repoChannels[parts[0]], parts[1])
	}
	var repositoryFilter []lib_models.RepositoryFilter
	for _, source := range sources {
		repositoryFilter = append(repositoryFilter, lib_models.RepositoryFilter{
			Source:   source,
			Channels: repoChannels[source],
		})
	}
	return repositoryFilter, nil
}

func GetRepositoryModules(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, "repositories/modules", func(gc *gin.Context) {
		filter, err := getGetRepositoryModulesFilter(gc)
		if err != nil {
			return
		}
		res, err := srv.GetRepositoryModules(gc, filter)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}
