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
	"github.com/SENERGY-Platform/go-cc-job-handler/ccjh"
	sb_logger "github.com/SENERGY-Platform/go-service-base/logger"
	cew_client "github.com/SENERGY-Platform/mgw-container-engine-wrapper/client"
	cm_client "github.com/SENERGY-Platform/mgw-core-manager/client"
	"github.com/SENERGY-Platform/mgw-go-service-base/job-hdl"
	"github.com/SENERGY-Platform/mgw-go-service-base/sql-db-hdl"
	srv_info_hdl "github.com/SENERGY-Platform/mgw-go-service-base/srv-info-hdl"
	sb_util "github.com/SENERGY-Platform/mgw-go-service-base/util"
	"github.com/SENERGY-Platform/mgw-go-service-base/watchdog"
	hm_client "github.com/SENERGY-Platform/mgw-host-manager/client"
	"github.com/SENERGY-Platform/mgw-modfile-lib/modfile"
	"github.com/SENERGY-Platform/mgw-modfile-lib/v1/v1dec"
	"github.com/SENERGY-Platform/mgw-modfile-lib/v1/v1gen"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/aux_dep_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/aux_job_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/cfg_valid_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/cfg_valid_hdl/validators"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/dep_adv_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/dep_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/http_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/mod_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/mod_staging_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/mod_transfer_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/mod_update_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/modfile_hdl"
	"github.com/SENERGY-Platform/mgw-module-manager/handler/storage_hdl"
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/manager"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"github.com/SENERGY-Platform/mgw-module-manager/util/db"
	"github.com/SENERGY-Platform/mgw-module-manager/util/db/instances_migr"
	"github.com/SENERGY-Platform/mgw-module-manager/util/db/modules_migr"
	"github.com/SENERGY-Platform/mgw-module-manager/util/naming_hdl"
	sm_client "github.com/SENERGY-Platform/mgw-secret-manager/pkg/client"
	"net"
	"net/http"
	"os"
	"strconv"
	"syscall"
	"time"
)

var version string

var inputValidators = map[string]cfg_valid_hdl.Validator{
	"regex":            validators.Regex,
	"number_compare":   validators.NumberCompare,
	"text_len_compare": validators.TextLenCompare,
}

func main() {
	srvInfoHdl := srv_info_hdl.New("module-manager", version)

	ec := 0
	defer func() {
		os.Exit(ec)
	}()

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
		var logFileError *sb_logger.LogFileError
		if errors.As(err, &logFileError) {
			ec = 1
			return
		}
	}
	if logFile != nil {
		defer logFile.Close()
	}

	util.Logger.Printf("%s %s", srvInfoHdl.GetName(), srvInfoHdl.GetVersion())

	util.Logger.Debugf("config: %s", sb_util.ToJsonStr(config))

	watchdog.Logger = util.Logger
	wtchdg := watchdog.New(syscall.SIGINT, syscall.SIGTERM)

	managerID, err := util.GetManagerID(config.ManagerIDPath, util.Flags.ManagerID)
	if err != nil {
		util.Logger.Error(err)
		ec = 1
		return
	}

	util.Logger.Debugf("manager ID: %s", managerID)

	naming_hdl.Init(config.CoreID, "mgw")

	sql_db_hdl.Logger = util.Logger
	db, err := db.New(config.Database.Host, config.Database.Port, config.Database.User, config.Database.Passwd.Value(), config.Database.Name)
	if err != nil {
		util.Logger.Error(err)
		ec = 1
		return
	}
	defer db.Close()

	storageHandler := storage_hdl.New(db)

	mfDecoders := make(modfile.Decoders)
	mfDecoders.Add(v1dec.GetDecoder)
	mfGenerators := make(modfile.Generators)
	mfGenerators.Add(v1gen.GetGenerator)

	modFileHandler := modfile_hdl.New(mfDecoders, mfGenerators)

	modTransferHandler := mod_transfer_hdl.New(config.ModTransferHandler.WorkdirPath, time.Duration(config.ModTransferHandler.Timeout))
	if err = modTransferHandler.InitWorkspace(0770); err != nil {
		util.Logger.Error(err)
		ec = 1
		return
	}

	cewClient := cew_client.New(http.DefaultClient, config.HttpClient.CewBaseUrl)

	modHandler := mod_hdl.New(storageHandler, modFileHandler, cewClient, time.Duration(config.Database.Timeout), time.Duration(config.HttpClient.Timeout), config.ModHandler.WorkdirPath)
	if err = modHandler.Init(0770); err != nil {
		util.Logger.Error(err)
		ec = 1
		return
	}

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

	cmClient := cm_client.New(http.DefaultClient, config.HttpClient.CmBaseUrl)

	hmClient := hm_client.New(http.DefaultClient, config.HttpClient.HmBaseUrl)

	smClient := sm_client.NewClient(config.HttpClient.SmBaseUrl, http.DefaultClient)

	depHandler := dep_hdl.New(storageHandler, cfgValidHandler, cewClient, cmClient, hmClient, smClient, time.Duration(config.Database.Timeout), time.Duration(config.HttpClient.Timeout), config.DepHandler.WorkdirPath, config.DepHandler.HostDepPath, config.DepHandler.HostSecPath, managerID, config.DepHandler.ModuleNet, config.CoreID)
	if err = depHandler.InitWorkspace(0770); err != nil {
		util.Logger.Error(err)
		ec = 1
		return
	}

	auxDepHandler := aux_dep_hdl.New(storageHandler, cewClient, time.Duration(config.Database.Timeout), time.Duration(config.HttpClient.Timeout), managerID, config.DepHandler.ModuleNet, config.CoreID, config.DepHandler.HostDepPath)

	ccHandler := ccjh.New(config.Jobs.BufferSize)

	job_hdl.Logger = util.Logger
	job_hdl.ErrCodeMapper = util.GetErrCode
	job_hdl.NewNotFoundErr = lib_model.NewNotFoundError
	job_hdl.NewInvalidInputError = lib_model.NewInvalidInputError
	job_hdl.NewInternalErr = lib_model.NewInternalError
	jobCtx, jobCF := context.WithCancel(context.Background())
	jobHandler := job_hdl.New(jobCtx, ccHandler)
	purgeJobsHdl := job_hdl.NewPurgeJobsHandler(jobHandler, time.Duration(config.Jobs.PJHInterval), time.Duration(config.Jobs.MaxAge))

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

	mm := manager.New(modHandler, modStagingHandler, modUpdateHandler, depHandler, auxDepHandler, jobHandler, aux_job_hdl.New(), dep_adv_hdl.New(storageHandler, time.Duration(config.Database.Timeout)), srvInfoHdl)

	httpHandler, err := http_hdl.New(mm, map[string]string{
		lib_model.HeaderApiVer:  srvInfoHdl.GetVersion(),
		lib_model.HeaderSrvName: srvInfoHdl.GetName(),
	})
	if err != nil {
		util.Logger.Error(err)
		ec = 1
		return
	}

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

	purgeJobsHdl.Start(jobCtx)

	go func() {
		defer dbCF()
		if err = sql_db_hdl.InitDB(dbCtx, db, config.Database.SchemaPath, time.Second*5, time.Duration(config.Database.Timeout), &instances_migr.Migration{
			Addr: config.Database.Host,
			Port: config.Database.Port,
			User: config.Database.User,
			PW:   config.Database.Passwd.Value(),
		}, &modules_migr.Migration{
			ModFileHandler: modFileHandler,
			WrkSpcPath:     config.ModHandler.WorkdirPath,
		}); err != nil {
			util.Logger.Error(err)
			ec = 1
			wtchdg.Trigger()
			return
		}
		if err = mm.StartEnabledDeployments(smClient, time.Second*5, 3); err != nil {
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
