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
	helper_naming "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/naming"
	helper_os_signal "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/os_signal"
	helper_slog "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slog"
	helper_sql_db "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/sql_db"
	helper_time "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/time"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/configuration"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants/slog_keys"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/service"
	sm_client "github.com/SENERGY-Platform/mgw-secret-manager/pkg/client"
)

const runtimeIdKey = "runtime_id"

var version string

func main() {

	// set exit status code
	ec := 0
	defer func() {
		os.Exit(ec)
	}()

	// create info handler
	serviceInfoHandler := srv_info_hdl.New("module-manager", version)

	// load configuration
	configuration.ParseFlags()
	config, err := configuration.New(configuration.ConfPath)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "load configuration: %s", err)
		ec = 1
		return
	}

	// init naming helper
	helper_naming.CoreId = config.CoreId
	helper_naming.ModuleContainerNetwork = config.ModuleContainerNetwork
	err = helper_naming.SetManagerID(config.ManagerIdPath, configuration.ManagerId)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "set manager id: %s\n", err)
		ec = 1
		return
	}

	// init time helper
	helper_time.UTC = config.UseUTC

	// create database handler
	mySQLConnector, err := handler_database.NewConnector(config.Database.MySQL)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "create mysql connector: %s\n", err)
		ec = 1
		return
	}
	sqlDB := helper_sql_db.NewSQLDatabase(mySQLConnector, config.Database.SQL)
	defer sqlDB.Close()
	databaseHandler := handler_database.New(sqlDB)

	// create repositories handler
	gitHubClient := client_repositories_github.New(
		helper_http.NewClient(time.Duration(config.GitHubModulesRepoHandler.Timeout)),
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

	// create modules handler
	modulesHandler := handler_modules.New(
		databaseHandler,
		cew_client.New(helper_http.NewClient(time.Duration(config.MGW.Timeout)), config.MGW.CewBaseUrl),
		config.ModulesHandler,
	)

	// create deployments handler
	deploymentsHandler := handler_deployments.New(
		databaseHandler,
		cew_client.New(helper_http.NewClient(time.Duration(config.MGW.Timeout)), config.MGW.CewBaseUrl),
		hm_client.New(helper_http.NewClient(time.Duration(config.MGW.Timeout)), config.MGW.HmBaseUrl),
		sm_client.NewClient(config.MGW.SmBaseUrl, helper_http.NewClient(time.Duration(config.MGW.Timeout))),
		cm_client.New(helper_http.NewClient(time.Duration(config.MGW.Timeout)), config.MGW.CmBaseUrl),
		config.DeploymentsHandler,
	)

	// create auxiliary deployments handler
	auxiliaryDeploymentsHandler := handler_aux_deployments.New(
		databaseHandler,
		cew_client.New(helper_http.NewClient(time.Duration(config.MGW.Timeout)), config.MGW.CewBaseUrl),
		config.AuxDeploymentsHandler,
	)

	// create main context
	ctx, cf := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, runtimeIdKey, helper_naming.GetRuntimeId())

	// create jobs handler
	jobsHandler := handler_jobs.New(ctx, config.JobsHandler)

	// create service
	srv := service.New(
		repositoriesHandler,
		modulesHandler,
		deploymentsHandler,
		auxiliaryDeploymentsHandler,
		handler_global_configs.New(databaseHandler),
		handler_dep_advertisements.New(databaseHandler),
		databaseHandler,
		jobsHandler,
	)

	// set job results cleanup callback
	jobsHandler.SetCleanupHandler(srv.DeleteJobResults)

	// create handler work directories
	err = modulesHandler.CreateWorkDir()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "create modules handler work directory: %s\n", err)
		ec = 1
		return
	}
	err = deploymentsHandler.CreateWorkDir()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "create deployments handler work directory: %s\n", err)
		ec = 1
		return
	}

	// create http api
	httpApi, err := api.New(srv, serviceInfoHandler, config.HttpAccessLog)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "create http api engine: %s\n", err)
		ec = 1
		return
	}

	// create http server
	httpServer := &http.Server{Handler: httpApi.Handler()}
	serverListener, err := net.Listen("tcp", ":"+strconv.FormatInt(int64(config.ServerPort), 10))
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "create server listener: %s\n", err)
		ec = 1
		return
	}

	// create logger
	helper_slog.ContextAttributeKeys = []string{runtimeIdKey, api.ContextKeyRequestId, handler_jobs.ContextKeyJobId}
	logger := helper_slog.New(config.Logger, os.Stderr, "", serviceInfoHandler.Name())

	// init loggers
	handler_database.InitLogger(logger)
	handler_repositories.InitLogger(logger)
	handler_modules.InitLogger(logger)
	handler_deployments.InitLogger(logger)
	handler_aux_deployments.InitLogger(logger)
	handler_global_configs.InitLogger(logger)
	handler_dep_advertisements.InitLogger(logger)
	migration_db_restructure.InitLogger(logger)
	service.InitLogger(logger)
	api.InitLogger(logger)

	// start service ---------------------------------------------------------------------------------------------------

	logger.InfoContext(
		ctx,
		"start service",
		slog_keys.Version, serviceInfoHandler.Version(),
		slog_keys.ManagerId, helper_naming.ManagerId,
		slog_keys.CoreId, helper_naming.CoreId,
		slog_keys.Config, sb_config_hdl.StructToMap(config, true),
	)

	// run database migrations
	err = databaseHandler.Migrate(ctx, migration_db_restructure.Migration, migration_db_init.Migration)
	if err != nil {
		logger.ErrorContext(ctx, "database migration", slog_keys.Error, err)
		ec = 1
		return
	}

	// init repositories
	err = repositoriesHandler.InitRepositories(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "initialize repositories", slog_keys.Error, err)
		ec = 1
		return
	}

	// init http api
	err = httpApi.Init(ctx)
	if err != nil {
		logger.ErrorContext(ctx, "initialize http api", slog_keys.Error, err)
		ec = 1
		return
	}

	// start os signal listener
	go func() {
		sig := helper_os_signal.Wait(ctx, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		if sig != nil {
			logger.WarnContext(ctx, "caught os signal", slog_keys.Signal, sig.String())
		}
		cf()
	}()

	// create wait group for parallel tasks
	wg := &sync.WaitGroup{}

	// start deployments runtime monitor
	wg.Add(1)
	go func() {
		defer wg.Done()
		deploymentsHandler.RuntimeMonitor(ctx)
		cf()
	}()

	// start auxiliary deployments runtime monitor
	wg.Add(1)
	go func() {
		defer wg.Done()
		auxiliaryDeploymentsHandler.RuntimeMonitor(ctx)
		cf()
	}()

	// start jobs cleanup routine
	wg.Add(1)
	go func() {
		defer wg.Done()
		jobsHandler.Cleanup(ctx)
		cf()
	}()

	// start http server
	go func() {
		logger.InfoContext(ctx, "start http server")
		if err := httpServer.Serve(serverListener); !errors.Is(err, http.ErrServerClosed) {
			logger.ErrorContext(ctx, "start server", slog_keys.Error, err)
			ec = 1
		}
		cf()
	}()

	// start http server shutdown listener
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		logger.InfoContext(ctx, "stop http server")
		ctxWt, cf2 := context.WithTimeout(context.Background(), time.Second*5)
		defer cf2()
		if err := httpServer.Shutdown(ctxWt); err != nil {
			logger.ErrorContext(ctx, "stop http server", slog_keys.Error, err)
			ec = 1
		} else {
			logger.InfoContext(ctx, "http server stopped")
		}
	}()

	// wait for parallel tasks to finish
	wg.Wait()
}
