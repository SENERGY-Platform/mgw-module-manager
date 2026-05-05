package service

import (
	"sync"

	models_service2 "github.com/SENERGY-Platform/mgw-module-manager/lib/models/service"
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
			deployments:         make(map[string]models_service2.JobResultDeployments),
			deploymentsUpdate:   make(map[string]models_service2.JobResultUpdateDeployments),
			moduleChange:        make(map[string]models_service2.JobResultModulesChange),
			refreshRepositories: make(map[string]models_service2.JobResult),
			auxDeploymentCreate: make(map[string]models_service2.JobResultCreateAuxiliaryDeployment),
			auxDeploymentUpdate: make(map[string]models_service2.JobResult),
			auxDeployment:       make(map[string]models_service2.JobResultAuxiliaryDeployments),
		},
	}
}
