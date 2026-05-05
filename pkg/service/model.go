package service

import (
	"io/fs"
	"time"

	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	models_service "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/service"
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
	Previous models_service.ModuleAbbreviated
	Next     modWrapper
}
