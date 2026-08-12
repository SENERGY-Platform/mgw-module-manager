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

func getRepositoriesRefreshFilter(gc *gin.Context) (lib_models.RepositoriesRefreshFilter, error) {
	var query struct {
		Types   []string `form:"types" collection_format:"csv"`
		Sources []string `form:"sources"  collection_format:"csv"`
	}
	err := gc.MustBindWith(&query, binding.Query)
	if err != nil {
		return lib_models.RepositoriesRefreshFilter{}, err
	}
	return lib_models.RepositoriesRefreshFilter{
		Types:   query.Types,
		Sources: query.Sources,
	}, nil
}

// @Summary		Refresh repositories
// @Description	Refresh all repositories or only the repositories matching the given filter as a job.
// @Tags		repositories
// @Produce		json
// @Param		types	query	[]string	false	"filter by repository types"	collectionFormat(csv)
// @Param		sources	query	[]string	false	"filter by repository sources"	collectionFormat(csv)
// @Success		200	{object}	lib_models.Job	"job"
// @Failure		400	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Failure		503	{string}	string	"error message"
// @Router		/repositories [patch]
func RefreshRepositories(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodPatch, lib_constants.HttpPathRepositoriesCollection, func(gc *gin.Context) {
		filter, err := getRepositoriesRefreshFilter(gc)
		if err != nil {
			return
		}
		res, err := srv.RefreshRepositories(gc, filter)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

// @Summary		Get repositories
// @Description	Get all configured module repositories.
// @Tags		repositories
// @Produce		json
// @Success		200	{array}	lib_models.Repository	"repositories"
// @Failure		500	{string}	string	"error message"
// @Failure		503	{string}	string	"error message"
// @Router		/repositories [get]
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

// @Summary		Create repository
// @Description	Create a module repository of the given type. The request body is passed to the repository type handler.
// @Tags		repositories
// @Accept		json
// @Produce		plain
// @Param		type	query	string	true	"repository type"
// @Param		request	body	string	true	"repository source definition"
// @Success		200	"repository created"
// @Failure		400	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Failure		503	{string}	string	"error message"
// @Router		/repositories [post]
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

// @Summary		Delete repository
// @Description	Delete a module repository by its source.
// @Tags		repositories
// @Produce		plain
// @Param		SOURCE	path	string	true	"repository source"
// @Success		200	"repository deleted"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Failure		503	{string}	string	"error message"
// @Router		/repositories/{SOURCE} [delete]
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

// @Summary		Get repository modules
// @Description	Get the modules provided by the configured repositories.
// @Tags		repositories
// @Produce		json
// @Param		ids	query	[]string	false	"filter by module IDs"	collectionFormat(csv)
// @Param		name	query	string	false	"filter by module name"
// @Param		repositories	query	[]string	false	"filter by repository sources"	collectionFormat(csv)
// @Param		repository_channels	query	[]string	false	"filter by repository channels, item format: source|channel"	collectionFormat(csv)
// @Param		installed	query	bool	false	"only return installed modules"
// @Param		update_available	query	bool	false	"only return modules with an available update"
// @Success		200	{array}	lib_models.RepoModule	"repository modules"
// @Failure		400	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Failure		503	{string}	string	"error message"
// @Router		/repository-modules [get]
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
