package service

import (
	"sync"

	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/jobs"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/service"
)

type Service struct {
	repositoriesHandler      repositoriesHandler
	modulesHandler           modulesHandler
	deploymentsHandler       deploymentsHandler
	auxDeploymentsHandler    auxiliaryDeploymentsHandler
	globalConfigsHandler     globalConfigsHandler
	depAdvertisementsHandler deploymentAdvertisementsHandler
	jobsHandler              *handler_jobs.Handler
	changeRequest            *modulesChangeRequest
	jobResults               jobResults
	mu                       sync.RWMutex
}

func New(
	repositoriesHandler repositoriesHandler,
	modulesHandler modulesHandler,
	deploymentsHandler deploymentsHandler,
	auxDeploymentsHandler auxiliaryDeploymentsHandler,
	globalConfigsHandler globalConfigsHandler,
	depAdvertisementsHandler deploymentAdvertisementsHandler,
	jobsHandler *handler_jobs.Handler,
) *Service {
	return &Service{
		repositoriesHandler:      repositoriesHandler,
		modulesHandler:           modulesHandler,
		deploymentsHandler:       deploymentsHandler,
		auxDeploymentsHandler:    auxDeploymentsHandler,
		globalConfigsHandler:     globalConfigsHandler,
		depAdvertisementsHandler: depAdvertisementsHandler,
		jobsHandler:              jobsHandler,
		jobResults: jobResults{
			deployments:         make(map[string]models_service.JobResultDeployments),
			deploymentsUpdate:   make(map[string]models_service.JobResultUpdateDeployments),
			moduleChange:        make(map[string]models_service.JobResultModulesChange),
			refreshRepositories: make(map[string]models_service.JobResult),
			auxDeploymentCreate: make(map[string]models_service.JobResultCreateAuxiliaryDeployment),
			auxDeploymentUpdate: make(map[string]models_service.JobResult),
			auxDeployment:       make(map[string]models_service.JobResultAuxiliaryDeployments),
		},
	}
}
