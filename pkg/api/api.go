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
	"log/slog"

	gin_mw "github.com/SENERGY-Platform/gin-middleware"
	sb_slog_attributes "github.com/SENERGY-Platform/go-service-base/struct-logger/attributes"
	lib_constants "github.com/SENERGY-Platform/mgw-module-manager/lib/models/constants"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants/attr_keys"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

type Api struct {
	service   serviceItf
	infoHdl   infoHandler
	ginEngine *gin.Engine
}

func New(service serviceItf, infoHdl infoHandler, logger *slog.Logger, accessLog bool) (*Api, error) {
	ginEngine := gin.New()
	var middleware []gin.HandlerFunc
	if accessLog {
		middleware = append(
			middleware,
			gin_mw.StructLoggerHandler(
				logger.With(sb_slog_attributes.LogRecordTypeKey, sb_slog_attributes.HttpAccessLogRecordTypeVal),
				sb_slog_attributes.Provider,
				nil,
				nil,
				requestIdGenerator,
			),
		)
	}
	middleware = append(middleware,
		gin_mw.StaticHeaderHandler(map[string]string{
			lib_constants.HttpHeaderApiVer:  infoHdl.Version(),
			lib_constants.HttpHeaderSrvName: infoHdl.Name(),
		}),
		requestid.New(requestid.WithCustomHeaderStrKey(lib_constants.HttpHeaderRequestId)),
		gin_mw.ErrorHandler(getStatusCode, ", "),
		gin_mw.StructRecoveryHandler(logger, gin_mw.DefaultRecoveryFunc),
	)
	ginEngine.Use(middleware...)
	ginEngine.UseRawPath = true
	a := &Api{
		service:   service,
		infoHdl:   infoHdl,
		ginEngine: ginEngine,
	}
	setRoutes, err := routes.Set(a, ginEngine)
	if err != nil {
		return nil, err
	}
	for _, route := range setRoutes {
		logger.Debug("http route", attr_keys.Method, route[0], attr_keys.Path, route[1])
	}
	return a, nil
}

func (a *Api) Handler() *gin.Engine {
	return a.ginEngine
}

func requestIdGenerator(gc *gin.Context) (string, any) {
	return attr_keys.RequestId, requestid.Get(gc)
}
