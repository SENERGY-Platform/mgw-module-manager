package service

import (
	module_lib "github.com/SENERGY-Platform/mgw-module-lib/model"
	models_service "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/service"
	"io/fs"
	"time"
)

type modWrapper struct {
	Mod     module_lib.Module
	FS      fs.FS
	Source  string
	Channel string
}

type repoAbbreviated struct {
	Priority int
	Channels map[string]int
}

type repoModAbbreviated struct {
	Name    string
	Desc    string
	Version string
}

type modulesInstallRequest struct {
	New     []modWrapper
	STC     []moduleSTC
	Created time.Time
}

type moduleSTC struct {
	Previous models_service.ModuleAbbreviated
	Next     modWrapper
}
