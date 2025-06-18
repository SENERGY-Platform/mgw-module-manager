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

/*
models_repo.Module{
	"id": "",
	"name": "",
	"description": "",
	"version": "",
	"source": "",
	"channel": ""
}
models_module.ModuleAbbreviated{
	"id": "",
	"name": "",
	"description": "",
	"version": "",
	"source": "",
	"channel": "",
	"added": "0001-01-01T00:00:00Z",
	"updated": "0001-01-01T00:00:00Z"
}
models_repo.Repository{
	"Source": "",
	"Default": false,
	"Channels": null,
	"DefaultChannel": ""
}
*/

type repoModAbbreviated struct {
	Name    string
	Desc    string
	Version string
}
