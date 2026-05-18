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
	cew_client "github.com/SENERGY-Platform/mgw-container-engine-wrapper/client"
	cm_client "github.com/SENERGY-Platform/mgw-core-manager/client"
	hm_client "github.com/SENERGY-Platform/mgw-host-manager/client"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/api"
	handler_aux_deployments "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/aux_deployments"
	handler_database "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/database"
	migration_db_init "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/database/migrations/db_init"
	migration_db_restructure "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/database/migrations/restructure"
	handler_dep_advertisements "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/dep_advertisements"
	handler_deployments "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/deployments"
	handler_global_configs "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/global_configs"
	handler_jobs "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/jobs"
	handler_modules "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/modules"
	handler_repositories "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/repositories"
	handler_repositories_github "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/repositories/github"
	client_repositories_github "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/repositories/github/client"
	helper_http "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/http"
	helper_os_signal "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/os_signal"
	helper_slog "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slog"
	helper_sql_db "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/sql_db"
	helper_time "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/time"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/configuration"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants/slog_keys"
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

	helper_slog.ContextAttributeKeys = []string{api.ContextKeyRequestId, handler_jobs.ContextKeyJobId}
	logger := helper_slog.New(config.Logger, os.Stderr, "", serviceInfoHandler.Name())

	logger.Info(
		"start service",
		slog_keys.Version,
		serviceInfoHandler.Version(),
		slog_keys.ConfigValues,
		sb_config_hdl.StructToMap(config, true),
	)

	ctx, cf := context.WithCancel(context.Background())

	mySQLConnector, err := handler_database.NewConnector(config.Database.MySQL)
	if err != nil {
		logger.Error("create mysql connector", slog_keys.Error, err)
		ec = 1
		return
	}
	sqlDB := helper_sql_db.NewSQLDatabase(mySQLConnector, config.Database.SQL)
	defer sqlDB.Close()

	handler_database.InitLogger(logger)
	databaseHandler := handler_database.New(sqlDB)
	migration_db_restructure.InitLogger(logger)
	err = databaseHandler.Migrate(ctx, migration_db_restructure.Migration, migration_db_init.Migration)
	if err != nil {
		logger.Error("database migration", slog_keys.Error, err)
		ec = 1
		return
	}

	gitHubClient := client_repositories_github.New(
		helper_http.NewClient(config.GitHubModulesRepoHandler.Timeout),
		config.GitHubModulesRepoHandler.BaseUrl,
	)

	handler_repositories.InitLogger(logger)
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

	handler_aux_deployments.InitLogger(logger)
	auxDeploymentsHandler := handler_aux_deployments.New(
		databaseHandler,
		containerEngineWrapperClient,
		config.AuxDeploymentsHandler,
	)

	jobsHandler := handler_jobs.New(ctx, config.JobsHandler)

	handler_global_configs.InitLogger(logger)
	handler_dep_advertisements.InitLogger(logger)
	service.InitLogger(logger)
	srv := service.New(
		repositoriesHandler,
		modulesHdl,
		deploymentsHandler,
		auxDeploymentsHandler,
		handler_global_configs.New(databaseHandler),
		handler_dep_advertisements.New(databaseHandler),
		databaseHandler,
		jobsHandler,
	)

	jobsHandler.SetCleanupHandler(srv.DeleteJobResults)

	httpApi, err := api.New(
		srv,
		serviceInfoHandler,
		logger,
		config.HttpAccessLog,
	)
	if err != nil {
		logger.Error("create http engine", slog_keys.Error, err)
		ec = 1
		return
	}

	httpServer := &http.Server{Handler: httpApi.Handler()}
	serverListener, err := net.Listen("tcp", ":"+strconv.FormatInt(int64(config.ServerPort), 10))
	if err != nil {
		logger.Error("create server listener", slog_keys.Error, err)
		ec = 1
		return
	}

	err = repositoriesHandler.InitRepositories(ctx)
	if err != nil {
		logger.Error("initialize repositories", slog_keys.Error, err)
		ec = 1
		return
	}

	err = modulesHdl.Init()
	if err != nil {
		logger.Error("initialize modules handler", slog_keys.Error, err)
		ec = 1
		return
	}

	err = deploymentsHandler.Init()
	if err != nil {
		logger.Error("initialize deployments handler", slog_keys.Error, err)
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
		deploymentsHandler.RuntimeMonitor(ctx)
		cf()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		auxDeploymentsHandler.RuntimeMonitor(ctx)
		cf()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		jobsHandler.Cleanup(ctx)
		cf()
	}()

	go func() {
		logger.Info("start http server")
		if err := httpServer.Serve(serverListener); !errors.Is(err, http.ErrServerClosed) {
			logger.Error("start server", slog_keys.Error, err)
			ec = 1
		}
		cf()
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		logger.Info("stop http server")
		ctxWt, cf2 := context.WithTimeout(context.Background(), time.Second*5)
		defer cf2()
		if err := httpServer.Shutdown(ctxWt); err != nil {
			logger.Error("stop http server", slog_keys.Error, err)
			ec = 1
		} else {
			logger.Info("http server stopped")
		}
	}()

	wg.Wait()
}
