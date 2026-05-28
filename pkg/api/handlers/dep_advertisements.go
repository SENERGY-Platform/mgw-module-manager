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
	"net/http"

	lib_constants "github.com/SENERGY-Platform/mgw-module-manager/lib/constants"
	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/service"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

func getQueryDeploymentAdvertisementsFilter(gc *gin.Context) (lib_models.DeploymentAdvertisementsFilter, error) {
	var query struct {
		Ids        []string `form:"ids" collection_format:"csv"`
		ModuleIds  []string `form:"module_ids" collection_format:"csv"`
		Origins    []string `form:"origins" collection_format:"csv"`
		References []string `form:"references" collection_format:"csv"`
	}
	err := gc.MustBindWith(&query, binding.Query)
	if err != nil {
		return lib_models.DeploymentAdvertisementsFilter{}, err
	}
	return lib_models.DeploymentAdvertisementsFilter{
		Ids:        query.Ids,
		ModuleIds:  query.ModuleIds,
		Origins:    query.Origins,
		References: query.References,
	}, nil
}

func QueryDeploymentAdvertisements(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_constants.HttpPathDeploymentAdvertisementsQueryCollection, func(gc *gin.Context) {
		filter, err := getQueryDeploymentAdvertisementsFilter(gc)
		if err != nil {
			return
		}
		res, err := srv.QueryDeploymentAdvertisements(gc, filter)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func QueryDeploymentAdvertisement(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_constants.HttpPathDeploymentAdvertisementQueryResource, func(gc *gin.Context) {
		res, err := srv.QueryDeploymentAdvertisement(gc, gc.Param("ADV_ID"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func GetDeploymentAdvertisement(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_constants.HttpPathDeploymentAdvertisementResource, func(gc *gin.Context) {
		res, err := srv.GetDeploymentAdvertisement(gc, gc.Param("DEP_ID"), gc.Param("ADV_REF"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func GetDeploymentAdvertisementById(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_constants.HttpPathDeploymentAdvertisementByIdResource, func(gc *gin.Context) {
		res, err := srv.GetDeploymentAdvertisementById(gc, gc.Param("DEP_ID"), gc.Param("ADV_ID"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func getDeploymentsAdvertisementsFilter(gc *gin.Context) (lib_models.DeploymentAdvertisementsFilterReduced, error) {
	var query struct {
		Ids        []string `form:"ids" collection_format:"csv"`
		References []string `form:"references" collection_format:"csv"`
	}
	err := gc.MustBindWith(&query, binding.Query)
	if err != nil {
		return lib_models.DeploymentAdvertisementsFilterReduced{}, err
	}
	return lib_models.DeploymentAdvertisementsFilterReduced{
		Ids:        query.Ids,
		References: query.References,
	}, nil
}

func GetDeploymentAdvertisements(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_constants.HttpPathDeploymentAdvertisementsCollection, func(gc *gin.Context) {
		filter, err := getDeploymentsAdvertisementsFilter(gc)
		if err != nil {
			return
		}
		res, err := srv.GetDeploymentAdvertisements(gc, gc.Param("DEP_ID"), filter)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func PutDeploymentAdvertisement(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodPut, lib_constants.HttpPathDeploymentAdvertisementResource, func(gc *gin.Context) {
		var body map[string]string
		err := gc.MustBindWith(&body, binding.JSON)
		if err != nil {
			return
		}
		res, err := srv.PutDeploymentAdvertisement(gc, gc.Param("DEP_ID"), gc.Param("ADV_REF"), body)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func PutDeploymentAdvertisements(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodPut, lib_constants.HttpPathDeploymentAdvertisementsCollection, func(gc *gin.Context) {
		var query struct {
			Incremental bool `form:"incremental"`
		}
		err := gc.MustBindWith(&query, binding.Query)
		if err != nil {
			return
		}
		var body []lib_models.DeploymentAdvertisementInput
		err = gc.MustBindWith(&body, binding.JSON)
		if err != nil {
			return
		}
		res, err := srv.PutDeploymentAdvertisements(gc, gc.Param("DEP_ID"), body, query.Incremental)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func getDeleteDeploymentAdvertisementsFilter(gc *gin.Context) (lib_models.DeploymentAdvertisementsFilterReduced, bool, error) {
	var query struct {
		Ids        []string `form:"ids" collection_format:"csv"`
		References []string `form:"references" collection_format:"csv"`
		AllowAll   bool     `form:"allow_all"`
	}
	err := gc.MustBindWith(&query, binding.Query)
	if err != nil {
		return lib_models.DeploymentAdvertisementsFilterReduced{}, false, err
	}
	return lib_models.DeploymentAdvertisementsFilterReduced{
		Ids:        query.Ids,
		References: query.References,
	}, query.AllowAll, nil
}

func DeleteDeploymentAdvertisement(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodDelete, lib_constants.HttpPathDeploymentAdvertisementResource, func(gc *gin.Context) {
		err := srv.DeleteDeploymentAdvertisement(gc, gc.Param("DEP_ID"), gc.Param("ADV_REF"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}

func DeleteDeploymentAdvertisements(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodDelete, lib_constants.HttpPathDeploymentAdvertisementsCollection, func(gc *gin.Context) {
		filter, allowAll, err := getDeleteDeploymentAdvertisementsFilter(gc)
		if err != nil {
			return
		}
		err = srv.DeleteDeploymentAdvertisements(gc, gc.Param("DEP_ID"), filter, allowAll)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}
