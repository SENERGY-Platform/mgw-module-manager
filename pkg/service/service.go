package service

import (
	"sync"
	"time"

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	handler_jobs "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/jobs"
	helper_time "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/time"
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
	now                      func() time.Time
	autoUpdateBackoff        autoUpdateBackoff
	infoHandler
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
	infoHandler infoHandler,
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
		infoHandler:              infoHandler,
		now:                      helper_time.Now,
		jobResults: jobResults{
			deployments:         make(map[string]lib_models.DeploymentJobResult),
			deploymentsUpdate:   make(map[string]lib_models.DeploymentUpdateJobResult),
			deploymentsDelete:   make(map[string]lib_models.DeploymentDeleteJobResult),
			moduleChange:        make(map[string]lib_models.ModulesChangeJobResult),
			refreshRepositories: make(map[string]lib_models.RepositoryJobResult),
			auxDeploymentCreate: make(map[string]lib_models.AuxiliaryDeploymentCreateJobResult),
			auxDeploymentUpdate: make(map[string]lib_models.JobResult),
			auxDeployment:       make(map[string]lib_models.AuxiliaryDeploymentJobResult),
		},
	}
}
