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
	return http.MethodGet, lib_constants.HttpPathSwaggerDoc, ginSwagger.WrapHandler(swaggerFiles.NewHandler())
}
