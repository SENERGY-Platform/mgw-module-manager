package service

import (
	"io/fs"
	"time"

	module_lib "github.com/SENERGY-Platform/mgw-module-lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/service"
)

const (
	repositoryJobSlotNum = iota
	deploymentJobSlotNum
	moduleJobSlotNum
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

type modulesChangeRequest struct {
	Install []modWrapper
	Change  []changeItem
	Remove  []string
	Created time.Time
}

type changeItem struct {
	Previous models_service.ModuleAbbreviated
	Next     modWrapper
}
