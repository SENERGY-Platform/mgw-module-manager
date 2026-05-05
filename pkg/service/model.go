package service

import (
	"io/fs"
	"time"

	lib_service "github.com/SENERGY-Platform/mgw-module-manager/lib/models/service"
	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

const (
	repositoryJobSlotNum = iota
	deploymentJobSlotNum
	moduleJobSlotNum
)

type modWrapper struct {
	Mod     models_external.ModuleLibModule
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
	Previous lib_service.ModuleAbbreviated
	Next     modWrapper
}
