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
	"path"

	lib_constants "github.com/SENERGY-Platform/mgw-module-manager/lib/constants"
	_ "github.com/SENERGY-Platform/mgw-module-manager/pkg/api/swagger-docs"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/service"
	"github.com/gin-gonic/gin"
	"github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"
)

// @Summary		Get swagger documentation
// @Description	Serve the swagger UI and the generated OpenAPI documentation of this API.
// @Tags		swagger
// @Produce		html
// @Param		any	path	string	true	"swagger UI resource"
// @Success		200	{string}	string	"swagger UI resource"
// @Failure		404	{string}	string	"error message"
// @Failure		500	{string}	string	"error message"
// @Router		/swagger/{any} [get]
func SwaggerDoc(_ *service.Service) (string, string, gin.HandlerFunc) {
	swaggerHandler := ginSwagger.WrapHandler(swaggerFiles.NewHandler())
	return http.MethodGet, lib_constants.HttpPathSwaggerDoc, func(gc *gin.Context) {
		// The swagger handler derives the prefix used for file lookups from the first request it
		// receives and then keeps it for the lifetime of the process. Duplicate slashes are matched
		// literally, so without normalisation a single malformed request breaks all later ones.
		gc.Request.URL.Path = path.Clean(gc.Request.URL.Path)
		gc.Request.URL.RawPath = ""
		gc.Request.RequestURI = gc.Request.URL.RequestURI()
		swaggerHandler(gc)
	}
}

// @Summary		Get swagger UI entrypoint
// @Description	Redirect to the swagger UI entrypoint. Required because the swagger handler only serves named resources and the engine does not append trailing slashes.
// @Tags		swagger
// @Success		301	{string}	string	"redirect to the swagger UI entrypoint"
// @Router		/swagger [get]
func SwaggerDocBase(_ *service.Service) (string, string, gin.HandlerFunc) {
	return http.MethodGet, lib_constants.HttpPathSwaggerDocBase, func(gc *gin.Context) {
		gc.Redirect(http.StatusMovedPermanently, gc.Request.URL.Path+"/index.html")
	}
}
