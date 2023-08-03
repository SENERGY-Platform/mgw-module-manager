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
	"github.com/SENERGY-Platform/go-service-base/srv-base"
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
	"github.com/SENERGY-Platform/mgw-module-manager/handler/dep_health_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/dep_storage_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/http_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/job_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/mod_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/mod_staging_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/mod_storage_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/mod_transfer_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/mod_update_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/modfile_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
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
	srv_base.PrintInfo(model.ServiceName, version)

	util.ParseFlags()

	config, err := util.NewConfig(util.Flags.ConfPath)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	logFile, err := util.InitLogger(config.Logger)
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

	managerID, err := util.GetManagerID(config.ManagerIDPath, util.Flags.ManagerID)
	if err != nil {
		util.Logger.Fatal(err)
	}

	util.Logger.Debugf("manager ID: %s", managerID)
	util.Logger.Debugf("config: %s", srv_base.ToJsonStr(config))

	mfDecoders := make(modfile.Decoders)
	mfDecoders.Add(v1dec.GetDecoder)
	mfGenerators := make(modfile.Generators)
	mfGenerators.Add(v1gen.GetGenerator)

	modFileHandler := modfile_hdl.New(mfDecoders, mfGenerators)

	modStorageHandler := mod_storage_hdl.New(config.ModStorageHandler.WorkdirPath, modFileHandler)
	if err = modStorageHandler.Init(0770); err != nil {
		util.Logger.Fatal(err)
	}

	cfgDefs, err := cfg_valid_hdl.LoadDefs(config.ConfigDefsPath)
	if err != nil {
		util.Logger.Fatal(err)
	}
	cfgValidHandler, err := cfg_valid_hdl.New(cfgDefs, inputValidators)
	if err != nil {
		util.Logger.Fatal(err)
	}

	modTransferHandler := mod_transfer_hdl.New(config.ModTransferHandler.WorkdirPath, time.Duration(config.ModTransferHandler.Timeout))
	if err = modTransferHandler.InitWorkspace(0770); err != nil {
		util.Logger.Fatal(err)
	}

	cewClient := cew_client.New(http.DefaultClient, config.HttpClient.CewBaseUrl)

	modHandler := mod_hdl.New(modStorageHandler, cewClient, time.Duration(config.HttpClient.Timeout))

	watchdog := srv_base.NewWatchdog(util.Logger, syscall.SIGINT, syscall.SIGTERM)

	db, err := util.NewDB(config.Database.Host, config.Database.Port, config.Database.User, config.Database.Passwd, config.Database.Name)
	if err != nil {
		util.Logger.Fatal(err)
	}
	watchdog.RegisterStopFunc(func() error {
		return db.Close()
	})

	depStorageHandler := dep_storage_hdl.New(db)

	hmClient := hm_client.New(http.DefaultClient, config.HttpClient.HmBaseUrl)

	smClient := sm_client.NewClient(config.HttpClient.SmBaseUrl, http.DefaultClient)

	depHandler := dep_hdl.New(depStorageHandler, cfgValidHandler, cewClient, hmClient, smClient, time.Duration(config.Database.Timeout), time.Duration(config.HttpClient.Timeout), config.DepHandler.WorkdirPath, config.DepHandler.HostDepPath, config.DepHandler.HostSecPath, managerID)
	if err = depHandler.InitWorkspace(0770); err != nil {
		util.Logger.Fatal(err)
	}

	depHealthHandler := dep_health_hdl.New(cewClient, time.Duration(config.HttpClient.Timeout))

	ccHandler := ccjh.New(config.Jobs.BufferSize)

	jobCtx, cFunc := context.WithCancel(context.Background())
	jobHandler := job_hdl.New(jobCtx, ccHandler)

	watchdog.RegisterStopFunc(func() error {
		ccHandler.Stop()
		cFunc()
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
		util.Logger.Fatal(err)
	}

	modUpdateHandler := mod_update_hdl.New(modTransferHandler, modFileHandler)

	mApi := api.New(modHandler, modStagingHandler, modUpdateHandler, depHandler, depHealthHandler, jobHandler)

	gin.SetMode(gin.ReleaseMode)
	httpHandler := gin.New()
	staticHeader := map[string]string{
		model.HeaderApiVer:  version,
		model.HeaderSrvName: model.ServiceName,
	}
	httpHandler.Use(gin_mw.StaticHeaderHandler(staticHeader), requestid.New(requestid.WithCustomHeaderStrKey(model.HeaderRequestID)), gin_mw.LoggerHandler(util.Logger, http_hdl.GetPathFilter(), func(gc *gin.Context) string {
		return requestid.Get(gc)
	}), gin_mw.ErrorHandler(http_hdl.GetStatusCode, ", "), gin.Recovery())
	httpHandler.UseRawPath = true

	http_hdl.SetRoutes(httpHandler, mApi)
	util.Logger.Debugf("routes: %s", srv_base.ToJsonStr(http_hdl.GetRoutes(httpHandler)))

	listener, err := net.Listen("tcp", ":"+strconv.FormatInt(int64(config.ServerPort), 10))
	if err != nil {
		util.Logger.Fatal(err)
	}
	server := &http.Server{}
	srvCtx, srvCF := context.WithCancel(context.Background())
	watchdog.RegisterStopFunc(func() error {
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
	watchdog.RegisterHealthFunc(func() bool {
		if srvCtx.Err() == nil {
			return true
		}
		util.Logger.Error("http server closed unexpectedly")
		return false
	})

	watchdog.Start()

	dbCtx, dbCF := context.WithCancel(context.Background())
	watchdog.RegisterStopFunc(func() error {
		dbCF()
		return nil
	})
	if err = depStorageHandler.Init(dbCtx, config.Database.SchemaPath, time.Second*5); err != nil {
		util.Logger.Fatal(err)
	}
	dbCF()

	err = ccHandler.RunAsync(config.Jobs.MaxNumber, time.Duration(config.Jobs.JHInterval*1000))
	if err != nil {
		util.Logger.Fatal(err)
	}

	if err = mApi.StartDeployments(); err != nil {
		util.Logger.Fatal(err)
	}

	go func() {
		defer srvCF()
		util.Logger.Info("starting http server ...")
		if err = server.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
			util.Logger.Fatal(err)
		}
	}()

	watchdog.Join()
}
