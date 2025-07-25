package main

import (
	"context"
	"errors"
	"fmt"
	sb_config_hdl "github.com/SENERGY-Platform/go-service-base/config-hdl"
	"github.com/SENERGY-Platform/go-service-base/srv-info-hdl"
	struct_logger "github.com/SENERGY-Platform/go-service-base/struct-logger"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/api"
	helper_os_signal "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/os_signal"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/config"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/slog_attr"
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

	config.ParseFlags()

	cfg, err := config.New(config.ConfPath)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		ec = 1
		return
	}

	logger := struct_logger.New(cfg.Logger, os.Stderr, "", srvInfoHdl.Name())

	logger.Info("starting service", slog_attr.VersionKey, srvInfoHdl.Version(), slog_attr.ConfigValuesKey, sb_config_hdl.StructToMap(cfg, true))

	httpApi, err := api.New(
		nil,
		srvInfoHdl,
		logger,
		cfg.HttpAccessLog,
	)
	if err != nil {
		logger.Error("creating http engine failed", slog_attr.ErrorKey, err)
		ec = 1
		return
	}

	httpServer := &http.Server{Handler: httpApi.Handler()}
	serverListener, err := net.Listen("tcp", ":"+strconv.FormatInt(int64(cfg.ServerPort), 10))
	if err != nil {
		logger.Error("creating server listener failed", slog_attr.ErrorKey, err)
		ec = 1
		return
	}

	ctx, cf := context.WithCancel(context.Background())

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
