package service

import (
	"sync"
)

type Service struct {
	repositoriesHandler repositoriesHandler
	modulesHandler      modulesHandler
	deploymentsHandler  deploymentsHandler
	changeRequest       *modulesChangeRequest
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
