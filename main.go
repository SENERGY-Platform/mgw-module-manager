/*
 * Copyright 2022 InfAI (CC SES)
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

package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/gin-middleware"
	"github.com/SENERGY-Platform/go-service-base/srv-base"
	"github.com/SENERGY-Platform/go-service-base/srv-base/types"
	"github.com/SENERGY-Platform/mgw-modfile-lib/modfile"
	"github.com/SENERGY-Platform/mgw-modfile-lib/v1/v1dec"
	"github.com/SENERGY-Platform/mgw-modfile-lib/v1/v1gen"
	"github.com/gin-gonic/gin"
	"module-manager/api"
	"module-manager/handler/deployment"
	"module-manager/handler/deployment/dep_storage"
	"module-manager/handler/http_engine"
	"module-manager/handler/module"
	"module-manager/handler/validation"
	"module-manager/handler/validation/validators"
	"module-manager/itf"
	"module-manager/model"
	"module-manager/util"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

var version string

var inputValidators = map[string]itf.Validator{
	"regex":            validators.Regex,
	"number_compare":   validators.NumberCompare,
	"text_len_compare": validators.TextLenCompare,
}

func setRoutes(e *gin.Engine, a itf.Api) {
	e.GET("modules", http_engine.GenHandler(a.GetModules))
	e.GET("modules/:m", http_engine.GenHandlerP(a.GetModule, func(gc *gin.Context) (string, error) {
		return http_engine.GetUrlParam(gc, "m")
	}))
	e.GET("modules/:m/input_template", http_engine.GenHandlerP(a.GetInputTemplate, func(gc *gin.Context) (string, error) {
		return http_engine.GetUrlParam(gc, "m")
	}))
	e.GET("deployments", http_engine.GenHandler(a.GetDeployments))
	e.GET("deployments/:d", http_engine.GenHandlerP(a.GetDeployment, func(gc *gin.Context) (string, error) {
		return http_engine.GetUrlParam(gc, "d")
	}))
	e.DELETE("deployments/:d", http_engine.GenNRHandlerP(a.DeleteDeployment, func(gc *gin.Context) (string, error) {
		return http_engine.GetUrlParam(gc, "d")
	}))
	e.POST("deployments", http_engine.GenHandlerP(a.AddDeployment, http_engine.GetJsonBody[model.DepRequest]))
}

func main() {
	srv_base.PrintInfo("mgw-module-manager", version)

	flags := util.NewFlags()

	config, err := util.NewConfig(flags.ConfPath)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	logFile, err := srv_base.InitLogger(config.Logger)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		var logFileError *srv_base.LogFileError
		if errors.As(err, &logFileError) {
			os.Exit(1)
		}
	}
	if logFile != nil {
		defer logFile.Close()
	}

	srv_base.Logger.Debugf("config: %s", srv_base.ToJsonStr(config))

	moduleStorageHandler, err := module.NewStorageHandler(config.ModuleFileHandler.WorkdirPath, config.ModuleFileHandler.Delimiter)
	if err != nil {
		srv_base.Logger.Error(err)
		return
	}

	cfgDefs, err := validation.LoadDefs(config.ConfigDefsPath)
	if err != nil {
		srv_base.Logger.Error(err)
		return
	}
	configValidationHandler, err := validation.NewConfigValidationHandler(cfgDefs, inputValidators)
	if err != nil {
		srv_base.Logger.Error(err)
		return
	}

	mfDecoders := make(modfile.Decoders)
	mfDecoders.Add(v1dec.GetDecoder)
	mfGenerators := make(modfile.Generators)
	mfGenerators.Add(v1gen.GetGenerator)

	moduleHandler := module.NewHandler(moduleStorageHandler, nil, configValidationHandler, mfDecoders, mfGenerators)

	dbCtx, dbCtxCf := context.WithCancel(context.Background())
	defer dbCtxCf()
	db, err := util.InitDB(dbCtx, config.DB.Host, config.DB.Port, config.DB.User, config.DB.Passwd, config.DB.Name, 10, 10, time.Duration(config.DB.Timeout))
	if err != nil {
		srv_base.Logger.Error(err)
		return
	}
	defer db.Close()

	depStorageHandler := dep_storage.NewStorageHandler(db, dbCtx, time.Duration(config.DB.Timeout))
	deploymentHandler := deployment.NewHandler(depStorageHandler, configValidationHandler)

	mApi := api.New(moduleHandler, deploymentHandler)

	gin.SetMode(gin.ReleaseMode)
	httpEngine := gin.New()
	httpEngine.Use(gin_mw.LoggerHandler(srv_base.Logger), gin_mw.ErrorHandler, gin.Recovery())
	httpEngine.UseRawPath = true

	setRoutes(httpEngine, mApi)

	listener, err := net.Listen("tcp", ":"+strconv.FormatInt(int64(config.ServerPort), 10))
	if err != nil {
		srv_base.Logger.Error(err)
		return
	}
	srv_base.StartServer(&http.Server{Handler: httpEngine}, listener, srv_base_types.DefaultShutdownSignals)
}
