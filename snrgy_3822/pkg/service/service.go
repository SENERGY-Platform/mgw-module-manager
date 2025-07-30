package service

import (
	"sync"
)

type Service struct {
	reposHdl   RepositoriesHandler
	modsHdl    ModulesHandler
	installReq *modulesInstallRequest
	mu         sync.RWMutex
}

func New(reposHdl RepositoriesHandler, modsHdl ModulesHandler) *Service {
	return &Service{
		reposHdl: reposHdl,
		modsHdl:  modsHdl,
	}
}

func equalMods(idA, sourceA, channelA, versionA, idB, sourceB, channelB, versionB string) bool {
	return idA == idB && sourceA == sourceB && channelA == channelB && versionA == versionB
}
