package service

import (
	module_lib "github.com/SENERGY-Platform/mgw-module-lib/model"
	"io/fs"
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
