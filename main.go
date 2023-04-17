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
	"github.com/SENERGY-Platform/mgw-container-engine-wrapper/client"
	"github.com/SENERGY-Platform/mgw-modfile-lib/modfile"
	"github.com/SENERGY-Platform/mgw-modfile-lib/v1/v1dec"
	"github.com/SENERGY-Platform/mgw-modfile-lib/v1/v1gen"
	"github.com/SENERGY-Platform/mgw-module-manager/api"
	"github.com/SENERGY-Platform/mgw-module-manager/api/http_engine"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/deployment"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/deployment/dep_storage"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/module"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/validation"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/validation/validators"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"github.com/gin-gonic/gin"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

var version string

var inputValidators = map[string]handler.Validator{
	"regex":            validators.Regex,
	"number_compare":   validators.NumberCompare,
	"text_len_compare": validators.TextLenCompare,
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

	mfDecoders := make(modfile.Decoders)
	mfDecoders.Add(v1dec.GetDecoder)
	mfGenerators := make(modfile.Generators)
	mfGenerators.Add(v1gen.GetGenerator)

	moduleStorageHandler, err := module.NewStorageHandler(config.ModuleFileHandler.WorkdirPath, config.ModuleFileHandler.Delimiter, mfDecoders, mfGenerators, 0660)
	if err != nil {
		srv_base.Logger.Error(err)
		return
	}
	if err := moduleStorageHandler.InitWorkspace(); err != nil {
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

	moduleHandler := module.NewHandler(moduleStorageHandler, nil, configValidationHandler)

	dbCtx, dbCtxCf := context.WithCancel(context.Background())
	defer dbCtxCf()
	db, err := util.InitDB(dbCtx, config.Database.Host, config.Database.Port, config.Database.User, config.Database.Passwd, config.Database.Name, 10, 10, time.Duration(config.Database.Timeout))
	if err != nil {
		srv_base.Logger.Error(err)
		return
	}
	defer db.Close()

	cewClient := client.New(http.DefaultClient, config.HttpClient.CewBaseUrl)

	depStorageHandler := dep_storage.NewStorageHandler(db)
	deploymentHandler := deployment.NewHandler(depStorageHandler, configValidationHandler, cewClient, time.Duration(config.Database.Timeout), time.Duration(config.HttpClient.Timeout))

	mApi := api.New(moduleHandler, deploymentHandler)

	gin.SetMode(gin.ReleaseMode)
	httpEngine := gin.New()
	httpEngine.Use(gin_mw.LoggerHandler(srv_base.Logger), gin_mw.ErrorHandler(http_engine.GetStatusCode, ", "), gin.Recovery())
	httpEngine.UseRawPath = true

	http_engine.SetRoutes(httpEngine, mApi)

	listener, err := net.Listen("tcp", ":"+strconv.FormatInt(int64(config.ServerPort), 10))
	if err != nil {
		srv_base.Logger.Error(err)
		return
	}
	srv_base.StartServer(&http.Server{Handler: httpEngine}, listener, srv_base_types.DefaultShutdownSignals)
}
