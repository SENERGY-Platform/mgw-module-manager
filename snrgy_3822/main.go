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
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/api"
	handler_database "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/database"
	handler_database_restructure "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/database/migrations/restructure"
	handler_database_schema "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/database/schema"
	handler_modules "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/modules"
	handler_repositories "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/repositories"
	handler_repositories_github "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/repositories/github"
	client_repositories_github "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/repositories/github/client"
	helper_http "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/http"
	helper_os_signal "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/os_signal"
	helper_sql_db "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/sql_db"
	helper_time "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/time"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/configuration"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/slog_attr"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/service"
)

var version string

func main() {
	ec := 0
	defer func() {
		os.Exit(ec)
	}()

	srvInfoHdl := srv_info_hdl.New("module-manager", version)

	configuration.ParseFlags()

	config, err := configuration.New(configuration.ConfPath)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		ec = 1
		return
	}

	helper_time.UTC = config.UseUTC

	logger := struct_logger.New(config.Logger, os.Stderr, "", srvInfoHdl.Name())

	logger.Info("starting service", slog_attr.VersionKey, srvInfoHdl.Version(), slog_attr.ConfigValuesKey, sb_config_hdl.StructToMap(config, true))

	ctx, cf := context.WithCancel(context.Background())

	mySQLConnector, err := handler_database.NewConnector(config.Database.MySQL)
	if err != nil {
		logger.Error("creating mysql connector failed", slog_attr.ErrorKey, err)
		ec = 1
		return
	}
	sqlDB := helper_sql_db.NewSQLDatabase(mySQLConnector, config.Database.SQL)
	defer sqlDB.Close()

	databaseHdl := handler_database.New(sqlDB)
	handler_database_restructure.InitLogger(logger)
	err = databaseHdl.Migrate(ctx, handler_database_schema.Init, handler_database_restructure.Migration)
	if err != nil {
		logger.Error("database migration failed", slog_attr.ErrorKey, err)
		ec = 1
		return
	}

	gitHubClt := client_repositories_github.New(helper_http.NewClient(config.GitHubModulesRepoHandler.Timeout), config.GitHubModulesRepoHandler.BaseUrl)

	repositoriesHdl := handler_repositories.New([]handler_repositories.Repository{
		{
			Handler: handler_repositories_github.New(gitHubClt, config.GitHubModulesRepoHandler.WorkDirPath, "SENERGY-Platform", "mgw-module-repository", "main-validated", []handler_repositories_github.Channel{
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
	})

	cewClient := cew_client.New(helper_http.NewClient(config.MGW.Timeout), config.MGW.CewBaseUrl)

	handler_modules.InitLogger(logger)
	modulesHdl := handler_modules.New(databaseHdl, cewClient, config.ModulesHandler)

	service.InitLogger(logger)
	srv := service.New(repositoriesHdl, modulesHdl)

	httpApi, err := api.New(
		srv,
		srvInfoHdl,
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

	err = repositoriesHdl.InitRepositories(ctx)
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

	go func() {
		helper_os_signal.Wait(ctx, logger, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		cf()
	}()

	wg := &sync.WaitGroup{}

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
