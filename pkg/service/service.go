package service

import (
	"sync"

	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/jobs"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/service"
)

type Service struct {
	repositoriesHandler   repositoriesHandler
	modulesHandler        modulesHandler
	deploymentsHandler    deploymentsHandler
	auxDeploymentsHandler auxiliaryDeploymentsHandler
	globalConfigsHandler  globalConfigsHandler
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
	globalConfigsHandler globalConfigsHandler,
	jobsHandler *handler_jobs.Handler,
) *Service {
	jResults := &jobResults{
		deploymentsResults:         make(map[string]models_service.JobResultDeployments),
		moduleChangeResults:        make(map[string]models_service.JobResultModulesChange),
		refreshRepositoriesResults: make(map[string]models_service.JobResult),
		auxDeploymentCreateResults: make(map[string]models_service.JobResultCreateAuxiliaryDeployment),
	}
	jobsHandler.SetCleanupHandler(jResults.deleteResults)
	return &Service{
		repositoriesHandler:   repositoriesHandler,
		modulesHandler:        modulesHandler,
		deploymentsHandler:    deploymentsHandler,
		auxDeploymentsHandler: auxDeploymentsHandler,
		globalConfigsHandler:  globalConfigsHandler,
		jobsHandler:           jobsHandler,
		jobResults:            jResults,
	}
}
