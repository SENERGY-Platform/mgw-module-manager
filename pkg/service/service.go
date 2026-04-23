package service

import (
	"sync"

	handler_jobs "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/jobs"
	models_service "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/service"
)

type Service struct {
	repositoriesHandler   repositoriesHandler
	modulesHandler        modulesHandler
	deploymentsHandler    deploymentsHandler
	auxDeploymentsHandler auxiliaryDeploymentsHandler
	jobsHandler           *handler_jobs.Handler
	changeRequest         *modulesChangeRequest
	changeReport          *models_service.ModulesChangeReport
	jobResults            *jobResults
	mu                    sync.RWMutex
}

func New(
	repositoriesHandler repositoriesHandler,
	modulesHandler modulesHandler,
	deploymentsHandler deploymentsHandler,
	auxDeploymentsHandler auxiliaryDeploymentsHandler,
	jobsHandler *handler_jobs.Handler,
) *Service {
	return &Service{
		repositoriesHandler:   repositoriesHandler,
		modulesHandler:        modulesHandler,
		deploymentsHandler:    deploymentsHandler,
		auxDeploymentsHandler: auxDeploymentsHandler,
		jobsHandler:           jobsHandler,
		jobResults: &jobResults{
			deployments: make(map[string]models_service.DeploymentsResult),
		},
	}
}
