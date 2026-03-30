package service

import (
	"sync"
)

type Service struct {
	repositoriesHandler repositoriesHandler
	modulesHandler      modulesHandler
	deploymentsHandler  deploymentsHandler
	changeReq           *modulesChangeRequest
	mu                  sync.RWMutex
}

func New(
	repositoriesHandler repositoriesHandler,
	modulesHandler modulesHandler,
	deploymentsHandler deploymentsHandler,
) *Service {
	return &Service{
		repositoriesHandler: repositoriesHandler,
		modulesHandler:      modulesHandler,
		deploymentsHandler:  deploymentsHandler,
	}
}

func equalMods(idA, sourceA, channelA, versionA, idB, sourceB, channelB, versionB string) bool {
	return idA == idB && sourceA == sourceB && channelA == channelB && versionA == versionB
}
