package service

import (
	"context"
	"io/fs"

	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/aux_deployments"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/database"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/deployments"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/modules"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/repositories"
)

type repositoriesHandler interface {
	RefreshRepositories(ctx context.Context) error
	Repositories(ctx context.Context) ([]models_handler_repositories.Repository, error)
	Module(ctx context.Context, id, source, channel string) (models_handler_repositories.Module, error)
	Modules(ctx context.Context, filter models_handler_repositories.ModulesFilter) ([]models_handler_repositories.Module, error)
	ModuleFS(ctx context.Context, id, source, channel string) (fs.FS, error)
}

type modulesHandler interface {
	Modules(ctx context.Context, filter models_handler_modules.ModuleFilter) (map[string]models_handler_modules.Module, error)
	Module(ctx context.Context, id string) (models_handler_modules.Module, error)
	Add(ctx context.Context, id, source, channel string, fSys fs.FS) error
	Update(ctx context.Context, id, source, channel string, fSys fs.FS) error
	Remove(ctx context.Context, id string) error
}

type deploymentsHandler interface {
	GetReducedDeploymentsByModuleIds(
		ctx context.Context,
		filter models_handler_deployments.DeploymentsFilter,
	) (map[string]models_handler_deployments.DeploymentReduced, error)
	GetDeploymentsByModuleIds(
		ctx context.Context,
		filter models_handler_deployments.DeploymentsFilter,
	) (map[string]models_handler_deployments.Deployment, error)
	GetDeploymentByModuleId(ctx context.Context, moduleId string) (models_handler_deployments.Deployment, error)
	CreateDeployments(
		ctx context.Context,
		selectedModules map[string]models_handler_modules.Module,
		userInputs map[string]models_handler_deployments.UserInput,
	) ([]models_handler_deployments.Result, error)
	UpdateDeployments(
		ctx context.Context,
		selectedModules map[string]models_handler_modules.Module,
		userInputs map[string]models_handler_deployments.UserInput,
	) ([]models_handler_deployments.Result, error)
	RecreateDeployments(
		ctx context.Context,
		selectedModules map[string]models_handler_modules.Module,
	) ([]models_handler_deployments.Result, error)
	DeleteDeployments(ctx context.Context, filter models_handler_deployments.DeploymentsFilter) error
	EnableDeployments(ctx context.Context, moduleIds []string) error
	DisableDeployments(ctx context.Context, moduleIds []string) error
}

type auxiliaryDeploymentsHandler interface {
	GetDeployment(
		ctx context.Context,
		deploymentId string,
		auxDeploymentId string,
	) (models_handler_aux_deployments.AuxiliaryDeployment, error)
	GetDeployments(
		ctx context.Context,
		deploymentId string,
		filter models_handler_aux_deployments.AuxiliaryDeploymentsFilter,
	) (map[string]models_handler_aux_deployments.AuxiliaryDeployment, error)
	GetReducedDeployments(
		ctx context.Context,
		deploymentId string,
		filter models_handler_aux_deployments.AuxiliaryDeploymentsFilter,
	) (map[string]models_handler_aux_deployments.AuxiliaryDeploymentReduced, error)
	CreateDeployment(
		ctx context.Context,
		module models_handler_modules.Module,
		activeDeployment models_handler_deployments.Deployment,
		dependencies map[string]models_handler_deployments.DeploymentReduced,
		serviceInput models_handler_aux_deployments.ServiceInput,
	) (models_handler_aux_deployments.Result, error)
	UpdateDeployment(
		ctx context.Context,
		module models_handler_modules.Module,
		activeDeployment models_handler_deployments.Deployment,
		dependencies map[string]models_handler_deployments.DeploymentReduced,
		auxDeploymentId string,
		serviceInput models_handler_aux_deployments.UpdateServiceInput,
	) error
	RecreateDeployments(
		ctx context.Context,
		module models_handler_modules.Module,
		activeDeployment models_handler_deployments.Deployment,
		dependencies map[string]models_handler_deployments.DeploymentReduced,
		filter models_handler_aux_deployments.AuxiliaryDeploymentsFilter,
	) ([]string, error)
	DeleteDeployments(
		ctx context.Context,
		deploymentId string,
		filter models_handler_aux_deployments.AuxiliaryDeploymentsFilter,
		allowAll bool,
	) ([]string, error)
	EnableDeployments(
		ctx context.Context,
		deploymentId string,
		filter models_handler_aux_deployments.AuxiliaryDeploymentsFilter,
	) error
	DisableDeployments(
		ctx context.Context,
		deploymentId string,
		filter models_handler_aux_deployments.AuxiliaryDeploymentsFilter,
	) error
	GetVolumes(
		ctx context.Context,
		deploymentId string,
		filterReferences []string,
	) (map[string]models_handler_database.AuxiliaryDeploymentVolume, error)
	GetVolumesWithMounts(
		ctx context.Context,
		deploymentId string,
		filterReferences []string,
	) (map[string]models_handler_database.AuxiliaryDeploymentVolumeWithMounts, error)
	DeleteVolumes(
		ctx context.Context,
		deploymentId string,
		filterReferences []string,
		allowAll bool,
	) ([]string, error)
	DeleteUnusedVolumes(ctx context.Context, deploymentId string, excludeReferences []string) ([]string, error)
	DeleteMutex(deploymentId string)
}
