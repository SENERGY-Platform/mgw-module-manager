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
	"github.com/SENERGY-Platform/go-cc-job-handler/ccjh"
	"github.com/SENERGY-Platform/go-service-base/job-hdl"
	sb_util "github.com/SENERGY-Platform/go-service-base/util"
	"github.com/SENERGY-Platform/go-service-base/watchdog"
	cew_client "github.com/SENERGY-Platform/mgw-container-engine-wrapper/client"
	hm_client "github.com/SENERGY-Platform/mgw-host-manager/client"
	"github.com/SENERGY-Platform/mgw-modfile-lib/modfile"
	"github.com/SENERGY-Platform/mgw-modfile-lib/v1/v1dec"
	"github.com/SENERGY-Platform/mgw-modfile-lib/v1/v1gen"
	"github.com/SENERGY-Platform/mgw-module-manager/api"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/cfg_valid_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/cfg_valid_hdl/validators"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/dep_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/http_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/mod_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/mod_staging_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/mod_storage_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/mod_transfer_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/mod_update_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/modfile_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/storage_hdl"
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"github.com/SENERGY-Platform/mgw-module-manager/util/db_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/util/db_hdl/instances_migr"
	"github.com/SENERGY-Platform/mgw-module-manager/util/naming_hdl"
	sm_client "github.com/SENERGY-Platform/mgw-secret-manager/pkg/client"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"net"
	"net/http"
	"os"
	"strconv"
	"syscall"
	"time"
)

var version string

var inputValidators = map[string]handler.Validator{
	"regex":            validators.Regex,
	"number_compare":   validators.NumberCompare,
	"text_len_compare": validators.TextLenCompare,
}

func main() {
	ec := 0
	defer func() {
		os.Exit(ec)
	}()

	sb_util.PrintInfo(lib_model.ServiceName, version)

	util.ParseFlags()

	config, err := util.NewConfig(util.Flags.ConfPath)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		ec = 1
		return
	}

	logFile, err := util.InitLogger(config.Logger)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		var logFileError *sb_util.LogFileError
		if errors.As(err, &logFileError) {
			ec = 1
			return
		}
	}
	if logFile != nil {
		defer logFile.Close()
	}

	util.Logger.Debugf("config: %s", sb_util.ToJsonStr(config))

	managerID, err := util.GetManagerID(config.ManagerIDPath, util.Flags.ManagerID)
	if err != nil {
		util.Logger.Error(err)
		ec = 1
		return
	}

	util.Logger.Debugf("manager ID: %s", managerID)

	naming_hdl.Init(config.CoreID, "mgw")

	mfDecoders := make(modfile.Decoders)
	mfDecoders.Add(v1dec.GetDecoder)
	mfGenerators := make(modfile.Generators)
	mfGenerators.Add(v1gen.GetGenerator)

	modFileHandler := modfile_hdl.New(mfDecoders, mfGenerators)

	modStorageHandler := mod_storage_hdl.New(config.ModStorageHandler.WorkdirPath, modFileHandler)
	if err = modStorageHandler.Init(0770); err != nil {
		util.Logger.Error(err)
		ec = 1
		return
	}

	modTransferHandler := mod_transfer_hdl.New(config.ModTransferHandler.WorkdirPath, time.Duration(config.ModTransferHandler.Timeout))
	if err = modTransferHandler.InitWorkspace(0770); err != nil {
		util.Logger.Error(err)
		ec = 1
		return
	}

	cewClient := cew_client.New(http.DefaultClient, config.HttpClient.CewBaseUrl)

	modHandler := mod_hdl.New(modStorageHandler, cewClient, time.Duration(config.HttpClient.Timeout))

	watchdog.Logger = util.Logger
	wtchdg := watchdog.New(syscall.SIGINT, syscall.SIGTERM)

	db, err := db_hdl.NewDB(config.Database.Host, config.Database.Port, config.Database.User, config.Database.Passwd.String(), config.Database.Name)
	if err != nil {
		util.Logger.Error(err)
		ec = 1
		return
	}
	defer db.Close()

	storageHandler := storage_hdl.New(db)

	cfgDefs, err := cfg_valid_hdl.LoadDefs(config.ConfigDefsPath)
	if err != nil {
		util.Logger.Error(err)
		ec = 1
		return
	}
	cfgValidHandler, err := cfg_valid_hdl.New(cfgDefs, inputValidators)
	if err != nil {
		util.Logger.Error(err)
		ec = 1
		return
	}

	hmClient := hm_client.New(http.DefaultClient, config.HttpClient.HmBaseUrl)

	smClient := sm_client.NewClient(config.HttpClient.SmBaseUrl, http.DefaultClient)

	depHandler := dep_hdl.New(storageHandler, cfgValidHandler, cewClient, hmClient, smClient, time.Duration(config.Database.Timeout), time.Duration(config.HttpClient.Timeout), config.DepHandler.WorkdirPath, config.DepHandler.HostDepPath, config.DepHandler.HostSecPath, managerID, config.DepHandler.ModuleNet, config.CoreID)
	if err = depHandler.InitWorkspace(0770); err != nil {
		util.Logger.Error(err)
		ec = 1
		return
	}

	ccHandler := ccjh.New(config.Jobs.BufferSize)

	job_hdl.Logger = util.Logger
	job_hdl.ErrCodeMapper = util.GetErrCode
	job_hdl.NewNotFoundErr = lib_model.NewNotFoundError
	job_hdl.NewInvalidInputError = lib_model.NewInvalidInputError
	job_hdl.NewInternalErr = lib_model.NewInternalError
	jobCtx, jobCF := context.WithCancel(context.Background())
	jobHandler := job_hdl.New(jobCtx, ccHandler)

	wtchdg.RegisterStopFunc(func() error {
		ccHandler.Stop()
		jobCF()
		if ccHandler.Active() > 0 {
			util.Logger.Info("waiting for active jobs to cancel ...")
			ctx, cf := context.WithTimeout(context.Background(), 5*time.Second)
			defer cf()
			for ccHandler.Active() != 0 {
				select {
				case <-ctx.Done():
					return fmt.Errorf("canceling jobs took too long")
				default:
					time.Sleep(50 * time.Millisecond)
				}
			}
			util.Logger.Info("jobs canceled")
		}
		return nil
	})

	modStagingHandler := mod_staging_hdl.New(config.ModStagingHandler.WorkdirPath, modTransferHandler, modFileHandler, cfgValidHandler, cewClient, time.Duration(config.HttpClient.Timeout))
	if err := modStagingHandler.InitWorkspace(0770); err != nil {
		util.Logger.Error(err)
		ec = 1
		return
	}

	modUpdateHandler := mod_update_hdl.New(modTransferHandler, modFileHandler)

	mApi := api.New(modHandler, modStagingHandler, modUpdateHandler, depHandler, jobHandler)

	gin.SetMode(gin.ReleaseMode)
	httpHandler := gin.New()
	staticHeader := map[string]string{
		lib_model.HeaderApiVer:  version,
		lib_model.HeaderSrvName: lib_model.ServiceName,
	}
	httpHandler.Use(gin_mw.StaticHeaderHandler(staticHeader), requestid.New(requestid.WithCustomHeaderStrKey(lib_model.HeaderRequestID)), gin_mw.LoggerHandler(util.Logger, http_hdl.GetPathFilter(), func(gc *gin.Context) string {
		return requestid.Get(gc)
	}), gin_mw.ErrorHandler(util.GetStatusCode, ", "), gin.Recovery())
	httpHandler.UseRawPath = true

	http_hdl.SetRoutes(httpHandler, mApi)
	util.Logger.Debugf("routes: %s", sb_util.ToJsonStr(http_hdl.GetRoutes(httpHandler)))

	listener, err := net.Listen("tcp", ":"+strconv.FormatInt(int64(config.ServerPort), 10))
	if err != nil {
		util.Logger.Error(err)
		ec = 1
		return
	}
	server := &http.Server{Handler: httpHandler}
	srvCtx, srvCF := context.WithCancel(context.Background())
	wtchdg.RegisterStopFunc(func() error {
		if srvCtx.Err() == nil {
			ctxWt, cf := context.WithTimeout(context.Background(), time.Second*5)
			defer cf()
			if err := server.Shutdown(ctxWt); err != nil {
				return err
			}
			util.Logger.Info("http server shutdown complete")
		}
		return nil
	})
	wtchdg.RegisterHealthFunc(func() bool {
		if srvCtx.Err() == nil {
			return true
		}
		util.Logger.Error("http server closed unexpectedly")
		return false
	})

	wtchdg.Start()

	dbCtx, dbCF := context.WithCancel(context.Background())
	wtchdg.RegisterStopFunc(func() error {
		dbCF()
		return nil
	})

	err = ccHandler.RunAsync(config.Jobs.MaxNumber, time.Duration(config.Jobs.JHInterval*1000))
	if err != nil {
		util.Logger.Error(err)
		ec = 1
		return
	}

	go func() {
		defer dbCF()
		if err = db_hdl.InitDB(dbCtx, db, config.Database.SchemaPath, time.Second*5, &instances_migr.Migration{
			Addr:    config.Database.Host,
			Port:    config.Database.Port,
			User:    config.Database.User,
			PW:      config.Database.Passwd.String(),
			Timeout: time.Duration(config.Database.Timeout),
		}); err != nil {
			util.Logger.Error(err)
			ec = 1
			wtchdg.Trigger()
			return
		}
		if err = mApi.StartupDeployments(smClient, time.Second*5, 3); err != nil {
			util.Logger.Error(err)
		}
	}()

	go func() {
		defer srvCF()
		util.Logger.Info("starting http server ...")
		if err := server.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
			util.Logger.Error(err)
			ec = 1
			return
		}
	}()

	ec = wtchdg.Join()
}
