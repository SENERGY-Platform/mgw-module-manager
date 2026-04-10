package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"syscall"
	"time"

	sb_config_hdl "github.com/SENERGY-Platform/go-service-base/config-hdl"
	"github.com/SENERGY-Platform/go-service-base/srv-info-hdl"
	struct_logger "github.com/SENERGY-Platform/go-service-base/struct-logger"
	cew_client "github.com/SENERGY-Platform/mgw-container-engine-wrapper/client"
	cm_client "github.com/SENERGY-Platform/mgw-core-manager/client"
	hm_client "github.com/SENERGY-Platform/mgw-host-manager/client"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/api"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/database"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/database/migrations/db_init"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/database/migrations/restructure"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/deployments"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/jobs"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/modules"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/repositories"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/repositories/github"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/repositories/github/client"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/http"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/os_signal"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/sql_db"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/time"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/configuration"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/slog_attr"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/service"
	sm_client "github.com/SENERGY-Platform/mgw-secret-manager/pkg/client"
)

var version string

func main() {
	ec := 0
	defer func() {
		os.Exit(ec)
	}()

	serviceInfoHandler := srv_info_hdl.New("module-manager", version)

	configuration.ParseFlags()

	config, err := configuration.New(configuration.ConfPath)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		ec = 1
		return
	}

	helper_time.UTC = config.UseUTC

	logger := struct_logger.New(config.Logger, os.Stderr, "", serviceInfoHandler.Name())

	logger.Info(
		"starting service",
		slog_attr.VersionKey,
		serviceInfoHandler.Version(),
		slog_attr.ConfigValuesKey,
		sb_config_hdl.StructToMap(config, true),
	)

	ctx, cf := context.WithCancel(context.Background())

	mySQLConnector, err := handler_database.NewConnector(config.Database.MySQL)
	if err != nil {
		logger.Error("creating mysql connector failed", slog_attr.ErrorKey, err)
		ec = 1
		return
	}
	sqlDB := helper_sql_db.NewSQLDatabase(mySQLConnector, config.Database.SQL)
	defer sqlDB.Close()

	databaseHandler := handler_database.New(sqlDB)
	migration_db_restructure.InitLogger(logger)
	err = databaseHandler.Migrate(ctx, migration_db_restructure.Migration, migration_db_init.Migration)
	if err != nil {
		logger.Error("database migration failed", slog_attr.ErrorKey, err)
		ec = 1
		return
	}

	gitHubClient := client_repositories_github.New(
		helper_http.NewClient(config.GitHubModulesRepoHandler.Timeout),
		config.GitHubModulesRepoHandler.BaseUrl,
	)

	repositoriesHandler := handler_repositories.New(
		[]handler_repositories.Repository{
			{
				Handler: handler_repositories_github.New(
					gitHubClient,
					config.GitHubModulesRepoHandler.WorkDirPath,
					"SENERGY-Platform",
					"mgw-module-repository",
					"main-validated",
					[]handler_repositories_github.Channel{
						{
							Name:     "main",
							Priority: 2,
						},
						{
							Name:     "testing",
							Priority: 1,
						},
						{
							Name:     "legacy",
							Priority: 0,
						},
					}),
				Priority: 1,
			},
		},
	)

	containerEngineWrapperClient := cew_client.New(helper_http.NewClient(config.MGW.Timeout), config.MGW.CewBaseUrl)

	handler_modules.InitLogger(logger)
	modulesHdl := handler_modules.New(databaseHandler, containerEngineWrapperClient, config.ModulesHandler)

	hostManagerClient := hm_client.New(helper_http.NewClient(config.MGW.Timeout), config.MGW.HmBaseUrl)

	secretManagerClient := sm_client.NewClient(config.MGW.SmBaseUrl, helper_http.NewClient(config.MGW.Timeout))

	coreManagerClient := cm_client.New(helper_http.NewClient(config.MGW.Timeout), config.MGW.CmBaseUrl)

	handler_deployments.InitLogger(logger)
	deploymentsHandler := handler_deployments.New(
		databaseHandler,
		containerEngineWrapperClient,
		hostManagerClient,
		secretManagerClient,
		coreManagerClient,
		config.DeploymentsHandler,
	)

	jobsHandler := handler_jobs.New(ctx, config.JobsHandler)

	service.InitLogger(logger)
	srv := service.New(repositoriesHandler, modulesHdl, deploymentsHandler, jobsHandler)

	httpApi, err := api.New(
		srv,
		serviceInfoHandler,
		logger,
		config.HttpAccessLog,
	)
	if err != nil {
		logger.Error("creating http engine failed", slog_attr.ErrorKey, err)
		ec = 1
		return
	}

	httpServer := &http.Server{Handler: httpApi.Handler()}
	serverListener, err := net.Listen("tcp", ":"+strconv.FormatInt(int64(config.ServerPort), 10))
	if err != nil {
		logger.Error("creating server listener failed", slog_attr.ErrorKey, err)
		ec = 1
		return
	}

	err = repositoriesHandler.InitRepositories(ctx)
	if err != nil {
		logger.Error("initializing module repositories failed", slog_attr.ErrorKey, err)
		ec = 1
		return
	}

	err = modulesHdl.Init()
	if err != nil {
		logger.Error("initializing modules handler failed", slog_attr.ErrorKey, err)
		ec = 1
		return
	}

	err = deploymentsHandler.Init()
	if err != nil {
		logger.Error("initializing deployments handler failed", slog_attr.ErrorKey, err)
		ec = 1
		return
	}

	go func() {
		helper_os_signal.Wait(ctx, logger, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		cf()
	}()

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		deploymentsHandler.DeploymentHealthMonitor(ctx)
		cf()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		jobsHandler.Cleanup(ctx)
		cf()
	}()

	go func() {
		logger.Info("starting http server")
		if err := httpServer.Serve(serverListener); !errors.Is(err, http.ErrServerClosed) {
			logger.Error("starting server failed", slog_attr.ErrorKey, err)
			ec = 1
		}
		cf()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		logger.Info("stopping http server")
		ctxWt, cf2 := context.WithTimeout(context.Background(), time.Second*5)
		defer cf2()
		if err := httpServer.Shutdown(ctxWt); err != nil {
			logger.Error("stopping server failed", slog_attr.ErrorKey, err)
			ec = 1
		} else {
			logger.Info("http server stopped")
		}
	}()

	wg.Wait()
}
