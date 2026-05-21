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

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/service"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

func ServiceHealth(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, "health/service", func(gc *gin.Context) {
		err := srv.ServiceHealth(gc)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}

func getDeploymentsHealthFilter(gc *gin.Context) (lib_models.DeploymentsHealthInfoFilter, error) {
	var query struct {
		ModuleIds               []string `form:"module_ids" collection_format:"csv"`
		ExclModuleIds           []string `form:"excl_module_ids" collection_format:"csv"`
		AuxiliaryDeployments    bool     `form:"auxiliary_deployments"`
		AuxDeploymentsOfIds     []string `form:"auxiliary_deployments_of_ids" collection_format:"csv"`
		ExclAuxDeploymentsOfIds []string `form:"excl_auxiliary_deployments_of_ids" collection_format:"csv"`
		IncludeHealthy          bool     `form:"include_healthy"`
	}
	err := gc.MustBindWith(&query, binding.Query)
	if err != nil {
		return lib_models.DeploymentsHealthInfoFilter{}, err
	}
	return lib_models.DeploymentsHealthInfoFilter{
		ModuleIds:               query.ModuleIds,
		ExclModuleIds:           query.ExclModuleIds,
		AuxiliaryDeployments:    query.AuxiliaryDeployments,
		AuxDeploymentsOfIds:     query.AuxDeploymentsOfIds,
		ExclAuxDeploymentsOfIds: query.ExclAuxDeploymentsOfIds,
		IncludeHealthy:          query.IncludeHealthy,
	}, nil
}

func DeploymentsHealth(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, "health/deployments", func(gc *gin.Context) {
		filter, err := getDeploymentsHealthFilter(gc)
		if err != nil {
			return
		}
		res, err := srv.DeploymentsHealth(gc, filter)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}
