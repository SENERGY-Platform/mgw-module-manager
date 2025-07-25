package main

import (
	"context"
	"errors"
	"fmt"
	sb_config_hdl "github.com/SENERGY-Platform/go-service-base/config-hdl"
	"github.com/SENERGY-Platform/go-service-base/srv-info-hdl"
	struct_logger "github.com/SENERGY-Platform/go-service-base/struct-logger"
	cew_client "github.com/SENERGY-Platform/mgw-container-engine-wrapper/client"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/api"
	handler_modules "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/modules"
	handler_repositories "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/repositories"
	handler_repositories_github "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/repositories/github"
	client_repositories_github "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/repositories/github/client"
	helper_http "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/http"
	helper_os_signal "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/os_signal"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/configuration"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/slog_attr"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/service"
	"net"
	"net/http"
	"os"
	"strconv"
	"sync"
	"syscall"
	"time"
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

	logger := struct_logger.New(config.Logger, os.Stderr, "", srvInfoHdl.Name())

	logger.Info("starting service", slog_attr.VersionKey, srvInfoHdl.Version(), slog_attr.ConfigValuesKey, sb_config_hdl.StructToMap(config, true))

	gitHubClt := client_repositories_github.New(helper_http.NewClient(config.GitHubModulesRepoHandler.Timeout), config.GitHubModulesRepoHandler.BaseUrl)

	repositoriesHdl := handler_repositories.New([]handler_repositories.Repository{
		{
			Handler: handler_repositories_github.New(gitHubClt, config.GitHubModulesRepoHandler.WorkDirPath, "SENERGY-Platform", "mgw-module-repository", []handler_repositories_github.Channel{
				{
					Name:      "main",
					Reference: "main-validated",
					Priority:  1,
				},
				{
					Name:      "testing",
					Reference: "testing-validated",
					Priority:  0,
				},
			}),
			Priority: 1,
		},
	})

	cewClient := cew_client.New(helper_http.NewClient(config.MGW.Timeout), config.MGW.CewBaseUrl)

	handler_modules.InitLogger(logger)
	modulesHdl := handler_modules.New(nil, cewClient, config.ModulesHandler)

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

	ctx, cf := context.WithCancel(context.Background())

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
