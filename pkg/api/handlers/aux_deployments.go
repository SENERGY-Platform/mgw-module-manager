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

func GetAuxiliaryDeployment(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, "deployments/:dep_id/auxiliary/deployments/:aux_dep_id", func(gc *gin.Context) {
		res, err := srv.GetAuxiliaryDeployment(gc, gc.Param("dep_id"), gc.Param("aux_dep_id"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func getAuxiliaryDeploymentsFilter(gc *gin.Context) (lib_models.AuxiliaryDeploymentsFilterWithState, error) {
	var query struct {
		Ids      []string `form:"ids" collection_format:"csv"`
		Labels   []string `form:"labels" collection_format:"csv"` // ITEM FORMAT -> key|value
		Image    string   `form:"image"`
		Enabled  int      `form:"enabled"`
		Recreate int      `form:"recreate"`
		State    string   `form:"state"`
	}
	err := gc.MustBindWith(&query, binding.Query)
	if err != nil {
		return lib_models.AuxiliaryDeploymentsFilterWithState{}, err
	}
	labels, err := getAuxiliaryDeploymentsFilterLabels(query.Labels)
	if err != nil {
		gc.AbortWithError(http.StatusBadRequest, err)
		return lib_models.AuxiliaryDeploymentsFilterWithState{}, err
	}
	return lib_models.AuxiliaryDeploymentsFilterWithState{
		AuxiliaryDeploymentsFilter: lib_models.AuxiliaryDeploymentsFilter{
			Ids:      query.Ids,
			Labels:   labels,
			Image:    query.Image,
			Enabled:  query.Enabled,
			Recreate: query.Recreate,
		},
		State: query.State,
	}, nil
}

func getAuxiliaryDeploymentsFilterLabels(queryLabels []string) (map[string]string, error) {
	labels := make(map[string]string)
	for _, item := range queryLabels {
		if item == "" {
			continue
		}
		parts := strings.Split(item, "|")
		if len(parts) != 2 {
			return nil, errors.New(fmt.Sprintf("invalid label format: %s", item))
		}
		labels[parts[0]] = parts[1]
	}
	return labels, nil
}

func GetAuxiliaryDeployments(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, "deployments/:dep_id/auxiliary/deployments", func(gc *gin.Context) {
		filter, err := getAuxiliaryDeploymentsFilter(gc)
		if err != nil {
			return
		}
		res, err := srv.GetAuxiliaryDeployments(gc, gc.Param("dep_id"), filter)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func GetReducedAuxiliaryDeployments(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, "deployments/:dep_id/auxiliary/deployments-reduced", func(gc *gin.Context) {
		filter, err := getAuxiliaryDeploymentsFilter(gc)
		if err != nil {
			return
		}
		res, err := srv.GetReducedAuxiliaryDeployments(gc, gc.Param("dep_id"), filter)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func CreateAuxiliaryDeployment(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodPost, "deployments/:dep_id/auxiliary/deployments", func(gc *gin.Context) {
		var body lib_models.AuxiliaryDeploymentInput
		err := gc.MustBindWith(&body, binding.JSON)
		if err != nil {
			return
		}
		res, err := srv.CreateAuxiliaryDeployment(gc, gc.Param("dep_id"), body)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func UpdateAuxiliaryDeployment(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodPut, "deployments/:dep_id/auxiliary/deployments/:aux_dep_id", func(gc *gin.Context) {
		var query struct {
			Incremental bool `form:"incremental"`
		}
		err := gc.MustBindWith(&query, binding.Query)
		if err != nil {
			return
		}
		var body lib_models.AuxiliaryDeploymentInput
		err = gc.MustBindWith(&body, binding.JSON)
		if err != nil {
			return
		}
		res, err := srv.UpdateAuxiliaryDeployment(gc, gc.Param("dep_id"), gc.Param("aux_dep_id"), body, query.Incremental)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func RecreateAuxiliaryDeployments(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodPost, "deployments/:dep_id/auxiliary/deployments-recreate", func(gc *gin.Context) {
		var body lib_models.AuxiliaryDeploymentsFilterWithState
		err := gc.MustBindWith(&body, binding.JSON)
		if err != nil {
			return
		}
		res, err := srv.RecreateAuxiliaryDeployments(gc, gc.Param("dep_id"), body)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func getDeleteAuxiliaryDeploymentsFilter(gc *gin.Context) (lib_models.AuxiliaryDeploymentsFilterWithState, bool, error) {
	var query struct {
		Ids      []string `form:"ids" collection_format:"csv"`
		Labels   []string `form:"labels" collection_format:"csv"` // ITEM FORMAT -> key|value
		Image    string   `form:"image"`
		Enabled  int      `form:"enabled"`
		Recreate int      `form:"recreate"`
		State    string   `form:"state"`
		AllowAll bool     `form:"allow_all"`
	}
	err := gc.MustBindWith(&query, binding.Query)
	if err != nil {
		return lib_models.AuxiliaryDeploymentsFilterWithState{}, false, err
	}
	labels, err := getAuxiliaryDeploymentsFilterLabels(query.Labels)
	if err != nil {
		gc.AbortWithError(http.StatusBadRequest, err)
		return lib_models.AuxiliaryDeploymentsFilterWithState{}, false, err
	}
	return lib_models.AuxiliaryDeploymentsFilterWithState{
		AuxiliaryDeploymentsFilter: lib_models.AuxiliaryDeploymentsFilter{
			Ids:      query.Ids,
			Labels:   labels,
			Image:    query.Image,
			Enabled:  query.Enabled,
			Recreate: query.Recreate,
		},
		State: query.State,
	}, query.AllowAll, nil
}

func DeleteAuxiliaryDeployment(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodDelete, "deployments/:dep_id/auxiliary/deployments/:aux_dep_id", func(gc *gin.Context) {
		err := srv.DeleteAuxiliaryDeployment(gc, gc.Param("dep_id"), gc.Param("aux_dep_id"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}

func DeleteAuxiliaryDeployments(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodDelete, "deployments/:dep_id/auxiliary/deployments", func(gc *gin.Context) {
		filter, allowAll, err := getDeleteAuxiliaryDeploymentsFilter(gc)
		if err != nil {
			return
		}
		res, err := srv.DeleteAuxiliaryDeployments(gc, gc.Param("dep_id"), filter, allowAll)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func EnableAuxiliaryDeployments(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodPost, "deployments/:dep_id/auxiliary/deployments-enable", func(gc *gin.Context) {
		var body lib_models.AuxiliaryDeploymentsFilterWithState
		err := gc.MustBindWith(&body, binding.JSON)
		if err != nil {
			return
		}
		res, err := srv.EnableAuxiliaryDeployments(gc, gc.Param("dep_id"), body)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func DisableAuxiliaryDeployments(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodPost, "deployments/:dep_id/auxiliary/deployments-disable", func(gc *gin.Context) {
		var body lib_models.AuxiliaryDeploymentsFilterWithState
		err := gc.MustBindWith(&body, binding.JSON)
		if err != nil {
			return
		}
		res, err := srv.DisableAuxiliaryDeployments(gc, gc.Param("dep_id"), body)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func GetAuxiliaryDeploymentVolumes(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, "deployments/:dep_id/auxiliary/volumes", func(gc *gin.Context) {
		var query struct {
			References []string `form:"references" collection_format:"csv"`
		}
		err := gc.MustBindWith(&query, binding.Query)
		if err != nil {
			return
		}
		res, err := srv.GetAuxiliaryDeploymentVolumes(gc, gc.Param("dep_id"), query.References)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func GetAuxiliaryDeploymentVolumesWithMounts(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, "deployments/:dep_id/auxiliary/volumes-with-mounts", func(gc *gin.Context) {
		var query struct {
			References []string `form:"references" collection_format:"csv"`
		}
		err := gc.MustBindWith(&query, binding.Query)
		if err != nil {
			return
		}
		res, err := srv.GetAuxiliaryDeploymentVolumesWithMounts(gc, gc.Param("dep_id"), query.References)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func DeleteAuxiliaryDeploymentVolumes(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodDelete, "deployments/:dep_id/auxiliary/volumes", func(gc *gin.Context) {
		var query struct {
			References []string `form:"references" collection_format:"csv"`
			AllowAll   bool     `form:"allow_all"`
			OnlyUnsued bool     `form:"only_unsued"`
		}
		err := gc.MustBindWith(&query, binding.Query)
		if err != nil {
			return
		}
		var res []lib_models.AuxiliaryDeploymentVolumeResult
		if query.OnlyUnsued {
			res, err = srv.DeleteUnusedAuxiliaryDeploymentVolumes(gc, gc.Param("dep_id"), query.References)
		} else {
			res, err = srv.DeleteAuxiliaryDeploymentVolumes(gc, gc.Param("dep_id"), query.References, query.AllowAll)
		}
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

func DeleteAuxiliaryDeploymentVolume(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodDelete, "deployments/:dep_id/auxiliary/volumes/:ref", func(gc *gin.Context) {
		err := srv.DeleteAuxiliaryDeploymentVolume(gc, gc.Param("dep_id"), gc.Param("ref"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}
