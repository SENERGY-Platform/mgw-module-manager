package service

import (
	"io/fs"
	"time"

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

const (
	repositoryJobSlotNum = iota
	deploymentJobSlotNum
	moduleJobSlotNum
)

type modWrapper struct {
	Mod     external_models.ModuleLibModule
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
	Previous lib_models.ModuleAbbreviated
	Next     modWrapper
}
