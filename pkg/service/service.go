package service

import (
	"sync"

	lib_models_service "github.com/SENERGY-Platform/mgw-module-manager/lib/models/service"
	handler_jobs "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/jobs"
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
			deployments:         make(map[string]lib_models_service.JobResultDeployments),
			deploymentsUpdate:   make(map[string]lib_models_service.JobResultUpdateDeployments),
			moduleChange:        make(map[string]lib_models_service.JobResultModulesChange),
			refreshRepositories: make(map[string]lib_models_service.JobResult),
			auxDeploymentCreate: make(map[string]lib_models_service.JobResultCreateAuxiliaryDeployment),
			auxDeploymentUpdate: make(map[string]lib_models_service.JobResult),
			auxDeployment:       make(map[string]lib_models_service.JobResultAuxiliaryDeployments),
		},
	}
}
