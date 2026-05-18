package service

import (
	"sync"

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	handler_jobs "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/jobs"
)

type Service struct {
	repositoriesHandler      repositoriesHandler
	modulesHandler           modulesHandler
	deploymentsHandler       deploymentsHandler
	auxDeploymentsHandler    auxiliaryDeploymentsHandler
	globalConfigsHandler     globalConfigsHandler
	depAdvertisementsHandler deploymentAdvertisementsHandler
	databaseHandler          databaseHandler
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
	databaseHandler databaseHandler,
	jobsHandler *handler_jobs.Handler,
) *Service {
	return &Service{
		repositoriesHandler:      repositoriesHandler,
		modulesHandler:           modulesHandler,
		deploymentsHandler:       deploymentsHandler,
		auxDeploymentsHandler:    auxDeploymentsHandler,
		globalConfigsHandler:     globalConfigsHandler,
		depAdvertisementsHandler: depAdvertisementsHandler,
		databaseHandler:          databaseHandler,
		jobsHandler:              jobsHandler,
		jobResults: jobResults{
			deployments:         make(map[string]lib_models.DeploymentJobResult),
			deploymentsUpdate:   make(map[string]lib_models.DeploymentUpdateJobResult),
			moduleChange:        make(map[string]lib_models.ModulesChangeJobResult),
			refreshRepositories: make(map[string]lib_models.JobResult),
			auxDeploymentCreate: make(map[string]lib_models.AuxiliaryDeploymentCreateJobResult),
			auxDeploymentUpdate: make(map[string]lib_models.JobResult),
			auxDeployment:       make(map[string]lib_models.AuxiliaryDeploymentJobResult),
		},
	}
}
