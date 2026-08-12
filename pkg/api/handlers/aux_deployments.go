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

	lib_constants "github.com/SENERGY-Platform/mgw-module-manager/lib/constants"
	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/service"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// @Summary		Get auxiliary deployment
// @Description	Get a single auxiliary deployment of a deployment.
// @Tags		auxiliary-deployments
// @Produce		json
// @Param		DEP_ID	path	string	true	"deployment ID"
// @Param		AUX_DEP_ID	path	string	true	"auxiliary deployment ID"
// @Success		200	{object}	lib_models.AuxiliaryDeployment	"auxiliary deployment"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/deployments/{DEP_ID}/auxiliary/deployments/{AUX_DEP_ID} [get]
// @Router		/restricted/deployments/{DEP_ID}/auxiliary/deployments/{AUX_DEP_ID} [get]
func GetAuxiliaryDeployment(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_constants.HttpPathAuxiliaryDeploymentResource, func(gc *gin.Context) {
		res, err := srv.GetAuxiliaryDeployment(gc, gc.Param("DEP_ID"), gc.Param("AUX_DEP_ID"))
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

// @Summary		Get auxiliary deployments
// @Description	Get the auxiliary deployments of a deployment.
// @Tags		auxiliary-deployments
// @Produce		json
// @Param		DEP_ID	path	string	true	"deployment ID"
// @Param		ids	query	[]string	false	"filter by auxiliary deployment IDs"	collectionFormat(csv)
// @Param		labels	query	[]string	false	"filter by labels, item format: key|value"	collectionFormat(csv)
// @Param		image	query	string	false	"filter by container image"
// @Param		enabled	query	int	false	"filter by enabled state"
// @Param		recreate	query	int	false	"filter by recreate state"
// @Param		state	query	string	false	"filter by container state"
// @Success		200	{object}	map[string]lib_models.AuxiliaryDeployment	"auxiliary deployments by ID"
// @Failure		400	{string}	string	"error message"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/deployments/{DEP_ID}/auxiliary/deployments [get]
// @Router		/restricted/deployments/{DEP_ID}/auxiliary/deployments [get]
func GetAuxiliaryDeployments(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_constants.HttpPathAuxiliaryDeploymentsCollection, func(gc *gin.Context) {
		filter, err := getAuxiliaryDeploymentsFilter(gc)
		if err != nil {
			return
		}
		res, err := srv.GetAuxiliaryDeployments(gc, gc.Param("DEP_ID"), filter)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

// @Summary		Get reduced auxiliary deployments
// @Description	Get the auxiliary deployments of a deployment with a reduced set of fields.
// @Tags		auxiliary-deployments
// @Produce		json
// @Param		DEP_ID	path	string	true	"deployment ID"
// @Param		ids	query	[]string	false	"filter by auxiliary deployment IDs"	collectionFormat(csv)
// @Param		labels	query	[]string	false	"filter by labels, item format: key|value"	collectionFormat(csv)
// @Param		image	query	string	false	"filter by container image"
// @Param		enabled	query	int	false	"filter by enabled state"
// @Param		recreate	query	int	false	"filter by recreate state"
// @Param		state	query	string	false	"filter by container state"
// @Success		200	{object}	map[string]lib_models.AuxiliaryDeploymentReduced	"reduced auxiliary deployments by ID"
// @Failure		400	{string}	string	"error message"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/deployments/{DEP_ID}/auxiliary/deployments-reduced [get]
// @Router		/restricted/deployments/{DEP_ID}/auxiliary/deployments-reduced [get]
func GetReducedAuxiliaryDeployments(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_constants.HttpPathReducedAuxiliaryDeploymentsCollection, func(gc *gin.Context) {
		filter, err := getAuxiliaryDeploymentsFilter(gc)
		if err != nil {
			return
		}
		res, err := srv.GetReducedAuxiliaryDeployments(gc, gc.Param("DEP_ID"), filter)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

// @Summary		Create auxiliary deployment
// @Description	Create an auxiliary deployment for a deployment as a job.
// @Tags		auxiliary-deployments
// @Accept		json
// @Produce		json
// @Param		DEP_ID	path	string	true	"deployment ID"
// @Param		pull_image	query	bool	false	"pull the container image"
// @Param		request	body	lib_models.AuxiliaryDeploymentInput	true	"auxiliary deployment"
// @Success		200	{object}	lib_models.Job	"job"
// @Failure		400	{string}	string	"error message"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Failure		503	{string}	string	"error message"
// @Router		/restricted/deployments/{DEP_ID}/auxiliary/deployments [post]
func CreateAuxiliaryDeployment(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodPost, lib_constants.HttpPathAuxiliaryDeploymentsCollection, func(gc *gin.Context) {
		var query struct {
			PullImage bool `form:"pull_image"`
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
		res, err := srv.CreateAuxiliaryDeployment(gc, gc.Param("DEP_ID"), body, query.PullImage)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

// @Summary		Update auxiliary deployment
// @Description	Update an auxiliary deployment of a deployment as a job.
// @Tags		auxiliary-deployments
// @Accept		json
// @Produce		json
// @Param		DEP_ID	path	string	true	"deployment ID"
// @Param		AUX_DEP_ID	path	string	true	"auxiliary deployment ID"
// @Param		incremental	query	bool	false	"only apply the given fields"
// @Param		pull_image	query	bool	false	"pull the container image"
// @Param		request	body	lib_models.AuxiliaryDeploymentInput	true	"auxiliary deployment"
// @Success		200	{object}	lib_models.Job	"job"
// @Failure		400	{string}	string	"error message"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Failure		503	{string}	string	"error message"
// @Router		/restricted/deployments/{DEP_ID}/auxiliary/deployments/{AUX_DEP_ID} [put]
func UpdateAuxiliaryDeployment(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodPut, lib_constants.HttpPathAuxiliaryDeploymentResource, func(gc *gin.Context) {
		var query struct {
			Incremental bool `form:"incremental"`
			PullImage   bool `form:"pull_image"`
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
		res, err := srv.UpdateAuxiliaryDeployment(
			gc,
			gc.Param("DEP_ID"),
			gc.Param("AUX_DEP_ID"),
			body,
			query.Incremental,
			query.PullImage,
		)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

// @Summary		Recreate auxiliary deployments
// @Description	Recreate the auxiliary deployments of a deployment matching the given filter as a job.
// @Tags		auxiliary-deployments
// @Accept		json
// @Produce		json
// @Param		DEP_ID	path	string	true	"deployment ID"
// @Param		request	body	lib_models.AuxiliaryDeploymentsFilterWithState	true	"auxiliary deployments filter"
// @Success		200	{object}	lib_models.Job	"job"
// @Failure		400	{string}	string	"error message"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Failure		503	{string}	string	"error message"
// @Router		/restricted/deployments/{DEP_ID}/auxiliary/deployments-recreate [post]
func RecreateAuxiliaryDeployments(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodPost, lib_constants.HttpPathRecreateAuxiliaryDeployments, func(gc *gin.Context) {
		var body lib_models.AuxiliaryDeploymentsFilterWithState
		err := gc.MustBindWith(&body, binding.JSON)
		if err != nil {
			return
		}
		res, err := srv.RecreateAuxiliaryDeployments(gc, gc.Param("DEP_ID"), body)
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

// @Summary		Delete auxiliary deployment
// @Description	Delete a single auxiliary deployment of a deployment.
// @Tags		auxiliary-deployments
// @Produce		plain
// @Param		DEP_ID	path	string	true	"deployment ID"
// @Param		AUX_DEP_ID	path	string	true	"auxiliary deployment ID"
// @Success		200	"auxiliary deployment deleted"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/restricted/deployments/{DEP_ID}/auxiliary/deployments/{AUX_DEP_ID} [delete]
func DeleteAuxiliaryDeployment(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodDelete, lib_constants.HttpPathAuxiliaryDeploymentResource, func(gc *gin.Context) {
		err := srv.DeleteAuxiliaryDeployment(gc, gc.Param("DEP_ID"), gc.Param("AUX_DEP_ID"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}

// @Summary		Delete auxiliary deployments
// @Description	Delete the auxiliary deployments of a deployment matching the given filter as a job. All auxiliary deployments are deleted if allow_all is set and no filter is given.
// @Tags		auxiliary-deployments
// @Produce		json
// @Param		DEP_ID	path	string	true	"deployment ID"
// @Param		ids	query	[]string	false	"filter by auxiliary deployment IDs"	collectionFormat(csv)
// @Param		labels	query	[]string	false	"filter by labels, item format: key|value"	collectionFormat(csv)
// @Param		image	query	string	false	"filter by container image"
// @Param		enabled	query	int	false	"filter by enabled state"
// @Param		recreate	query	int	false	"filter by recreate state"
// @Param		state	query	string	false	"filter by container state"
// @Param		allow_all	query	bool	false	"allow deletion of all auxiliary deployments"
// @Success		200	{object}	lib_models.Job	"job"
// @Failure		400	{string}	string	"error message"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/restricted/deployments/{DEP_ID}/auxiliary/deployments [delete]
func DeleteAuxiliaryDeployments(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodDelete, lib_constants.HttpPathAuxiliaryDeploymentsCollection, func(gc *gin.Context) {
		filter, allowAll, err := getDeleteAuxiliaryDeploymentsFilter(gc)
		if err != nil {
			return
		}
		res, err := srv.DeleteAuxiliaryDeployments(gc, gc.Param("DEP_ID"), filter, allowAll)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

// @Summary		Enable auxiliary deployments
// @Description	Enable the auxiliary deployments of a deployment matching the given filter.
// @Tags		auxiliary-deployments
// @Accept		json
// @Produce		json
// @Param		DEP_ID	path	string	true	"deployment ID"
// @Param		request	body	lib_models.AuxiliaryDeploymentsFilterWithState	true	"auxiliary deployments filter"
// @Success		200	{array}	string	"IDs of the enabled auxiliary deployments"
// @Failure		400	{string}	string	"error message"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/restricted/deployments/{DEP_ID}/auxiliary/deployments-enable [post]
func EnableAuxiliaryDeployments(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodPost, lib_constants.HttpPathEnableAuxiliaryDeployments, func(gc *gin.Context) {
		var body lib_models.AuxiliaryDeploymentsFilterWithState
		err := gc.MustBindWith(&body, binding.JSON)
		if err != nil {
			return
		}
		res, err := srv.EnableAuxiliaryDeployments(gc, gc.Param("DEP_ID"), body)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

// @Summary		Disable auxiliary deployments
// @Description	Disable the auxiliary deployments of a deployment matching the given filter.
// @Tags		auxiliary-deployments
// @Accept		json
// @Produce		json
// @Param		DEP_ID	path	string	true	"deployment ID"
// @Param		request	body	lib_models.AuxiliaryDeploymentsFilterWithState	true	"auxiliary deployments filter"
// @Success		200	{array}	string	"IDs of the disabled auxiliary deployments"
// @Failure		400	{string}	string	"error message"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/restricted/deployments/{DEP_ID}/auxiliary/deployments-disable [post]
func DisableAuxiliaryDeployments(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodPost, lib_constants.HttpPathDisableAuxiliaryDeployments, func(gc *gin.Context) {
		var body lib_models.AuxiliaryDeploymentsFilterWithState
		err := gc.MustBindWith(&body, binding.JSON)
		if err != nil {
			return
		}
		res, err := srv.DisableAuxiliaryDeployments(gc, gc.Param("DEP_ID"), body)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

// @Summary		Get auxiliary deployment volumes
// @Description	Get the auxiliary deployment volumes of a deployment.
// @Tags		auxiliary-deployments
// @Produce		json
// @Param		DEP_ID	path	string	true	"deployment ID"
// @Param		references	query	[]string	false	"filter by volume references"	collectionFormat(csv)
// @Success		200	{object}	map[string]lib_models.AuxiliaryDeploymentVolume	"auxiliary deployment volumes by reference"
// @Failure		400	{string}	string	"error message"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/deployments/{DEP_ID}/auxiliary/volumes [get]
// @Router		/restricted/deployments/{DEP_ID}/auxiliary/volumes [get]
func GetAuxiliaryDeploymentVolumes(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_constants.HttpPathAuxiliaryDeploymentVolumesCollection, func(gc *gin.Context) {
		var query struct {
			References []string `form:"references" collection_format:"csv"`
		}
		err := gc.MustBindWith(&query, binding.Query)
		if err != nil {
			return
		}
		res, err := srv.GetAuxiliaryDeploymentVolumes(gc, gc.Param("DEP_ID"), query.References)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

// @Summary		Get auxiliary deployment volumes with mounts
// @Description	Get the auxiliary deployment volumes of a deployment including the auxiliary deployments they are mounted in.
// @Tags		auxiliary-deployments
// @Produce		json
// @Param		DEP_ID	path	string	true	"deployment ID"
// @Param		references	query	[]string	false	"filter by volume references"	collectionFormat(csv)
// @Success		200	{object}	map[string]lib_models.AuxiliaryDeploymentVolumeWithMounts	"auxiliary deployment volumes with mounts by reference"
// @Failure		400	{string}	string	"error message"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/deployments/{DEP_ID}/auxiliary/volumes-with-mounts [get]
// @Router		/restricted/deployments/{DEP_ID}/auxiliary/volumes-with-mounts [get]
func GetAuxiliaryDeploymentVolumesWithMounts(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_constants.HttpPathAuxiliaryDeploymentVolumesWithMountsCollection, func(gc *gin.Context) {
		var query struct {
			References []string `form:"references" collection_format:"csv"`
		}
		err := gc.MustBindWith(&query, binding.Query)
		if err != nil {
			return
		}
		res, err := srv.GetAuxiliaryDeploymentVolumesWithMounts(gc, gc.Param("DEP_ID"), query.References)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

// @Summary		Delete auxiliary deployment volumes
// @Description	Delete the auxiliary deployment volumes of a deployment. If only_unsued is set, the given references are excluded and only unused volumes are deleted.
// @Tags		auxiliary-deployments
// @Produce		json
// @Param		DEP_ID	path	string	true	"deployment ID"
// @Param		references	query	[]string	false	"filter by volume references"	collectionFormat(csv)
// @Param		allow_all	query	bool	false	"allow deletion of all auxiliary deployment volumes"
// @Param		only_unsued	query	bool	false	"only delete unused auxiliary deployment volumes"
// @Success		200	{array}	lib_models.AuxiliaryDeploymentVolumeResult	"auxiliary deployment volume results"
// @Failure		400	{string}	string	"error message"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/restricted/deployments/{DEP_ID}/auxiliary/volumes [delete]
func DeleteAuxiliaryDeploymentVolumes(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodDelete, lib_constants.HttpPathAuxiliaryDeploymentVolumesCollection, func(gc *gin.Context) {
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
			res, err = srv.DeleteUnusedAuxiliaryDeploymentVolumes(gc, gc.Param("DEP_ID"), query.References)
		} else {
			res, err = srv.DeleteAuxiliaryDeploymentVolumes(gc, gc.Param("DEP_ID"), query.References, query.AllowAll)
		}
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.JSON(http.StatusOK, res)
	}
}

// @Summary		Delete auxiliary deployment volume
// @Description	Delete a single auxiliary deployment volume of a deployment.
// @Tags		auxiliary-deployments
// @Produce		plain
// @Param		DEP_ID	path	string	true	"deployment ID"
// @Param		AUX_VOL_REF	path	string	true	"auxiliary deployment volume reference"
// @Success		200	"auxiliary deployment volume deleted"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/restricted/deployments/{DEP_ID}/auxiliary/volumes/{AUX_VOL_REF} [delete]
func DeleteAuxiliaryDeploymentVolume(srv *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodDelete, lib_constants.HttpPathAuxiliaryDeploymentVolumeResource, func(gc *gin.Context) {
		err := srv.DeleteAuxiliaryDeploymentVolume(gc, gc.Param("DEP_ID"), gc.Param("AUX_VOL_REF"))
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}
