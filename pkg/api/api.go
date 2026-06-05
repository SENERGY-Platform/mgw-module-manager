/*
 * Copyright 2025 InfAI (CC SES)
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

package api

import (
	"net/http"

	gin_mw "github.com/SENERGY-Platform/gin-middleware"
	sb_slog_attributes "github.com/SENERGY-Platform/go-service-base/struct-logger/attributes"
	lib_constants "github.com/SENERGY-Platform/mgw-module-manager/lib/constants"
	helper_naming "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/naming"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/service"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

const ContextKeyRequestId = "request_id"

func CreateHandler(srv *service.Service, srvName, srvVersion string, accessLog bool) (http.Handler, error) {
	ginEngine := gin.New()
	ginEngine.RedirectTrailingSlash = false
	var middleware []gin.HandlerFunc
	if accessLog {
		middleware = append(
			middleware,
			gin_mw.StructLoggerHandler(
				accessLogger,
				sb_slog_attributes.Provider,
				[]string{"health/service"},
				nil,
			),
		)
	}
	middleware = append(middleware,
		runtimeIdContextHandler,
		requestid.New(
			requestid.WithCustomHeaderStrKey(lib_constants.HttpHeaderRequestId),
			requestid.WithHandler(requestIdContextHandler),
		),
		gin_mw.StaticHeaderHandler(map[string]string{
			lib_constants.HttpHeaderApiVer:    srvVersion,
			lib_constants.HttpHeaderSrvName:   srvName,
			lib_constants.HttpHeaderRuntimeId: helper_naming.RuntimeId,
			lib_constants.HttpHeaderCoreId:    helper_naming.CoreId,
			lib_constants.HttpHeaderManagerId: helper_naming.ManagerId,
		}),
		errorHandler("Err%d: %s"),
		gin_mw.StructRecoveryHandler(logger, gin_mw.DefaultRecoveryFunc),
	)
	ginEngine.Use(middleware...)
	ginEngine.UseRawPath = true
	err := registerHandlers(ginEngine, srv, append(standardApiHandlers, sharedApiHandlers...)...)
	if err != nil {
		return nil, err
	}
	err = registerHandlers(ginEngine.Group("restricted"), srv, append(restrictedApiHandlers, sharedApiHandlers...)...)
	if err != nil {
		return nil, err
	}
	return ginEngine, nil
}

func requestIdContextHandler(ctx *gin.Context, requestId string) {
	ctx.Set(ContextKeyRequestId, requestId)
}

func runtimeIdContextHandler(ctx *gin.Context) {
	ctx.Set(helper_naming.RuntimeIdKey, helper_naming.RuntimeId)
	ctx.Next()
}
