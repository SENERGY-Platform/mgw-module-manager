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
	"io"
	"net/http"
	"strings"

	lib_constants "github.com/SENERGY-Platform/mgw-module-manager/lib/constants"
	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/service"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

func RefreshRepositories(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodPatch, lib_constants.HttpPathRepositoriesCollection, func(gc *gin.Context) {
		res, err := srv.RefreshRepositories(gc)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func GetRepositories(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_constants.HttpPathRepositoriesCollection, func(gc *gin.Context) {
		res, err := srv.GetRepositories(gc)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func CreateRepository(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodPost, lib_constants.HttpPathRepositoriesCollection, func(gc *gin.Context) {
		var query struct {
			Type string `form:"type"`
		}
		err := gc.MustBindWith(&query, binding.Query)
		if err != nil {
			return
		}
		defer gc.Request.Body.Close()
		data, err := io.ReadAll(gc.Request.Body)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		err = srv.CreateRepository(gc, query.Type, data)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}

func DeleteRepository(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodDelete, lib_constants.HttpPathRepositoryResource, func(gc *gin.Context) {
		err := srv.DeleteRepository(gc, gc.Param("SOURCE"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
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

func getRepositoryFilter(sources, sourceChannels []string) ([]lib_models.RepoModuleRepositoriesFilter, error) {
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
	var repositoryFilter []lib_models.RepoModuleRepositoriesFilter
	for _, source := range sources {
		repositoryFilter = append(repositoryFilter, lib_models.RepoModuleRepositoriesFilter{
			Source:   source,
			Channels: repoChannels[source],
		})
	}
	return repositoryFilter, nil
}

func GetRepositoryModules(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_constants.HttpPathRepositoryModulesCollection, func(gc *gin.Context) {
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
