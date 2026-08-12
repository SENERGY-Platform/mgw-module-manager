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
		References []string `form:"references" collection_format:"csv"`
	}
	err := gc.MustBindWith(&query, binding.Query)
	if err != nil {
		return lib_models.DeploymentAdvertisementsFilter{}, err
	}
	return lib_models.DeploymentAdvertisementsFilter{
		Ids:        query.Ids,
		ModuleIds:  query.ModuleIds,
		References: query.References,
	}, nil
}

// @Summary		Query deployment advertisements
// @Description	Query the advertisements of all deployments with a reduced set of fields.
// @Tags		deployment-advertisements
// @Produce		json
// @Param		ids	query	[]string	false	"filter by advertisement IDs"	collectionFormat(csv)
// @Param		module_ids	query	[]string	false	"filter by module IDs"	collectionFormat(csv)
// @Param		references	query	[]string	false	"filter by advertisement references"	collectionFormat(csv)
// @Success		200	{array}	lib_models.DeploymentAdvertisementReduced	"reduced deployment advertisements"
// @Failure		400	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/deployment-advertisements [get]
// @Router		/restricted/deployment-advertisements [get]
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

// @Summary		Query deployment advertisement
// @Description	Query a single deployment advertisement by its ID with a reduced set of fields.
// @Tags		deployment-advertisements
// @Produce		json
// @Param		ADV_ID	path	string	true	"advertisement ID"
// @Success		200	{object}	lib_models.DeploymentAdvertisementReduced	"reduced deployment advertisement"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/deployment-advertisements/{ADV_ID} [get]
// @Router		/restricted/deployment-advertisements/{ADV_ID} [get]
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

// @Summary		Get deployment advertisement
// @Description	Get a single advertisement of a deployment by its reference.
// @Tags		deployment-advertisements
// @Produce		json
// @Param		DEP_ID	path	string	true	"deployment ID"
// @Param		ADV_REF	path	string	true	"advertisement reference"
// @Success		200	{object}	lib_models.DeploymentAdvertisement	"deployment advertisement"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/restricted/deployments/{DEP_ID}/advertisements/{ADV_REF} [get]
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

// @Summary		Get deployment advertisement by ID
// @Description	Get a single advertisement of a deployment by its ID.
// @Tags		deployment-advertisements
// @Produce		json
// @Param		DEP_ID	path	string	true	"deployment ID"
// @Param		ADV_ID	path	string	true	"advertisement ID"
// @Success		200	{object}	lib_models.DeploymentAdvertisement	"deployment advertisement"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/restricted/deployments/{DEP_ID}/advertisements-by-id/{ADV_ID} [get]
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

// @Summary		Get deployment advertisements
// @Description	Get the advertisements of a deployment.
// @Tags		deployment-advertisements
// @Produce		json
// @Param		DEP_ID	path	string	true	"deployment ID"
// @Param		ids	query	[]string	false	"filter by advertisement IDs"	collectionFormat(csv)
// @Param		references	query	[]string	false	"filter by advertisement references"	collectionFormat(csv)
// @Success		200	{object}	map[string]lib_models.DeploymentAdvertisement	"deployment advertisements by reference"
// @Failure		400	{string}	string	"error message"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/restricted/deployments/{DEP_ID}/advertisements [get]
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

// @Summary		Put deployment advertisement
// @Description	Create or replace a single advertisement of a deployment identified by its reference.
// @Tags		deployment-advertisements
// @Accept		json
// @Produce		json
// @Param		DEP_ID	path	string	true	"deployment ID"
// @Param		ADV_REF	path	string	true	"advertisement reference"
// @Param		request	body	map[string]string	true	"advertisement items"
// @Success		200	{string}	string	"advertisement ID"
// @Failure		400	{string}	string	"error message"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/restricted/deployments/{DEP_ID}/advertisements/{ADV_REF} [put]
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

// @Summary		Put deployment advertisements
// @Description	Create or replace the advertisements of a deployment. Advertisements not contained in the request are removed unless incremental is set.
// @Tags		deployment-advertisements
// @Accept		json
// @Produce		json
// @Param		DEP_ID	path	string	true	"deployment ID"
// @Param		incremental	query	bool	false	"keep advertisements not contained in the request"
// @Param		request	body	[]lib_models.DeploymentAdvertisementInput	true	"deployment advertisements"
// @Success		200	{object}	map[string]string	"advertisement IDs by reference"
// @Failure		400	{string}	string	"error message"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/restricted/deployments/{DEP_ID}/advertisements [put]
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

// @Summary		Delete deployment advertisement
// @Description	Delete a single advertisement of a deployment by its reference.
// @Tags		deployment-advertisements
// @Produce		plain
// @Param		DEP_ID	path	string	true	"deployment ID"
// @Param		ADV_REF	path	string	true	"advertisement reference"
// @Success		200	"deployment advertisement deleted"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/restricted/deployments/{DEP_ID}/advertisements/{ADV_REF} [delete]
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

// @Summary		Delete deployment advertisements
// @Description	Delete the advertisements of a deployment matching the given filter. All advertisements are deleted if allow_all is set and no filter is given.
// @Tags		deployment-advertisements
// @Produce		plain
// @Param		DEP_ID	path	string	true	"deployment ID"
// @Param		ids	query	[]string	false	"filter by advertisement IDs"	collectionFormat(csv)
// @Param		references	query	[]string	false	"filter by advertisement references"	collectionFormat(csv)
// @Param		allow_all	query	bool	false	"allow deletion of all deployment advertisements"
// @Success		200	"deployment advertisements deleted"
// @Failure		400	{string}	string	"error message"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/restricted/deployments/{DEP_ID}/advertisements [delete]
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
