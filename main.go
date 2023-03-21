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
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/gin-middleware"
	"github.com/SENERGY-Platform/go-service-base/srv-base"
	"github.com/SENERGY-Platform/go-service-base/srv-base/types"
	"github.com/SENERGY-Platform/mgw-modfile-lib/modfile"
	"github.com/SENERGY-Platform/mgw-modfile-lib/v1/v1dec"
	"github.com/SENERGY-Platform/mgw-modfile-lib/v1/v1gen"
	"github.com/gin-gonic/gin"
	"module-manager/manager/api"
	"module-manager/manager/handler/deployment"
	"module-manager/manager/handler/http_engine"
	"module-manager/manager/handler/module"
	"module-manager/manager/handler/validation"
	"module-manager/manager/handler/validation/validators"
	"module-manager/manager/itf"
	"module-manager/manager/util"
	"net"
	"net/http"
	"os"
	"strconv"
)

var version string

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

	moduleStorageHandler, err := module.NewFileHandler(config.ModuleFileHandler.WorkdirPath, config.ModuleFileHandler.Delimiter)
	if err != nil {
		srv_base.Logger.Error(err)
		return
	}

	configValidationHandler, err := validation.NewConfigValidationHandler(config.ConfigDefsPath, map[string]itf.Validator{
		"regex":            validators.Regex,
		"number_compare":   validators.NumberCompare,
		"text_len_compare": validators.TextLenCompare,
	})
	if err != nil {
		srv_base.Logger.Error(err)
		return
	}

	mfDecoders := make(modfile.Decoders)
	mfDecoders.Add(v1dec.GetDecoder)
	mfGenerators := make(modfile.Generators)
	mfGenerators.Add(v1gen.GetGenerator)

	moduleHandler := module.NewHandler(moduleStorageHandler, configValidationHandler, mfDecoders, mfGenerators)
	deploymentHandler := deployment.NewHandler(nil)

	mApi := api.New(moduleHandler, deploymentHandler)

	gin.SetMode(gin.ReleaseMode)
	httpEngine := gin.New()
	httpEngine.Use(gin_mw.LoggerHandler(srv_base.Logger), gin_mw.ErrorHandler, gin.Recovery())
	httpEngine.UseRawPath = true

	httpEngine.GET("modules", http_engine.GenHandler(mApi.GetModules))
	httpEngine.GET("modules/:m", http_engine.GenHandlerP(mApi.GetModule, func(gc *gin.Context) (string, error) {
		return http_engine.GetUrlParam(gc, "m")
	}))
	httpEngine.GET("modules/:m/input_template", http_engine.GenHandlerP(mApi.GetInputTemplate, func(gc *gin.Context) (string, error) {
		return http_engine.GetUrlParam(gc, "m")
	}))

	listener, err := net.Listen("tcp", ":"+strconv.FormatInt(int64(config.ServerPort), 10))
	if err != nil {
		srv_base.Logger.Error(err)
		return
	}
	srv_base.StartServer(&http.Server{Handler: httpEngine}, listener, srv_base_types.DefaultShutdownSignals)
}
