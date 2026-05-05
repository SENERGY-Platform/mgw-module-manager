package service

import (
	"context"
	"io/fs"

	lib_models_aux_deployments "github.com/SENERGY-Platform/mgw-module-manager/lib/models/aux_deployments"
	lib_models_dep_advertisements "github.com/SENERGY-Platform/mgw-module-manager/lib/models/dep_advertisements"
	lib_models_service "github.com/SENERGY-Platform/mgw-module-manager/lib/models/service"
	models_configs "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/configs"
	models_deployments "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/deployments"
	models_handler_repositories "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/repositories"
	models_module "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/modules"
)

type repositoriesHandler interface {
	RefreshRepositories(ctx context.Context) error
	Repositories(ctx context.Context) ([]models_handler_repositories.Repository, error)
	Module(ctx context.Context, id, source, channel string) (models_handler_repositories.Module, error)
	Modules(ctx context.Context, filter models_handler_repositories.ModulesFilter) ([]models_handler_repositories.Module, error)
	ModuleFS(ctx context.Context, id, source, channel string) (fs.FS, error)
}

type modulesHandler interface {
	Modules(ctx context.Context, filter models_module.ModuleFilter) (map[string]models_module.Module, error)
	Module(ctx context.Context, id string) (models_module.Module, error)
	Add(ctx context.Context, id, source, channel string, fSys fs.FS) error
	Update(ctx context.Context, id, source, channel string, fSys fs.FS) error
	Remove(ctx context.Context, id string) error
}

type deploymentsHandler interface {
	GetDeployment(ctx context.Context, id string) (models_deployments.Deployment, error)
	GetReducedDeploymentsByModuleIds(
		ctx context.Context,
		filter models_deployments.DeploymentsFilterWithState,
	) (map[string]models_deployments.DeploymentReduced, error)
	GetDeploymentsByModuleIds(
		ctx context.Context,
		filter models_deployments.DeploymentsFilterWithState,
	) (map[string]models_deployments.Deployment, error)
	GetDeploymentByModuleId(ctx context.Context, moduleId string) (models_deployments.Deployment, error)
	GetDeploymentIds(ctx context.Context, filter models_deployments.DeploymentsFilter) (map[string]string, error)
	CreateDeployments(
		ctx context.Context,
		selectedModules map[string]models_module.Module,
		userInputs map[string]models_deployments.UserInput,
	) ([]lib_models_service.DeploymentResult, error)
	UpdateDeployments(
		ctx context.Context,
		selectedModules map[string]models_module.Module,
		userInputs map[string]models_deployments.UserInput,
	) ([]lib_models_service.DeploymentResult, error)
	RecreateDeployments(
		ctx context.Context,
		selectedModules map[string]models_module.Module,
	) ([]lib_models_service.DeploymentResult, error)
	DeleteDeployments(
		ctx context.Context,
		filter models_deployments.DeploymentsFilterWithState,
		allowAll bool,
	) ([]lib_models_service.DeploymentResult, error)
	EnableDeployments(ctx context.Context, moduleIds []string) ([]string, error)
	DisableDeployments(ctx context.Context, moduleIds []string) ([]string, error)
}

type auxiliaryDeploymentsHandler interface {
	GetDeployment(
		ctx context.Context,
		deploymentId string,
		auxDeploymentId string,
	) (lib_models_aux_deployments.AuxiliaryDeployment, error)
	GetDeployments(
		ctx context.Context,
		deploymentId string,
		filter lib_models_aux_deployments.AuxiliaryDeploymentsFilterWithState,
	) (map[string]lib_models_aux_deployments.AuxiliaryDeployment, error)
	GetReducedDeployments(
		ctx context.Context,
		deploymentId string,
		filter lib_models_aux_deployments.AuxiliaryDeploymentsFilterWithState,
	) (map[string]lib_models_aux_deployments.AuxiliaryDeploymentReduced, error)
	CreateDeployment(
		ctx context.Context,
		module models_module.Module,
		activeDeployment models_deployments.Deployment,
		dependencies map[string]models_deployments.DeploymentReduced,
		serviceInput lib_models_aux_deployments.ServiceInput,
	) (lib_models_aux_deployments.Result, error)
	UpdateDeployment(
		ctx context.Context,
		module models_module.Module,
		activeDeployment models_deployments.Deployment,
		dependencies map[string]models_deployments.DeploymentReduced,
		auxDeploymentId string,
		serviceInput lib_models_aux_deployments.UpdateServiceInput,
	) error
	RecreateDeployments(
		ctx context.Context,
		module models_module.Module,
		activeDeployment models_deployments.Deployment,
		dependencies map[string]models_deployments.DeploymentReduced,
		filter lib_models_aux_deployments.AuxiliaryDeploymentsFilterWithState,
	) ([]lib_models_aux_deployments.BatchResult, error)
	DeleteDeployments(
		ctx context.Context,
		deploymentId string,
		filter lib_models_aux_deployments.AuxiliaryDeploymentsFilterWithState,
		allowAll bool,
	) ([]lib_models_aux_deployments.BatchResult, error)
	EnableDeployments(
		ctx context.Context,
		deploymentId string,
		filter lib_models_aux_deployments.AuxiliaryDeploymentsFilterWithState,
	) ([]string, error)
	DisableDeployments(
		ctx context.Context,
		deploymentId string,
		filter lib_models_aux_deployments.AuxiliaryDeploymentsFilterWithState,
	) ([]string, error)
	GetVolumes(
		ctx context.Context,
		deploymentId string,
		filterReferences []string,
	) (map[string]lib_models_aux_deployments.AuxiliaryDeploymentVolume, error)
	GetVolumesWithMounts(
		ctx context.Context,
		deploymentId string,
		filterReferences []string,
	) (map[string]lib_models_aux_deployments.AuxiliaryDeploymentVolumeWithMounts, error)
	DeleteVolumes(
		ctx context.Context,
		deploymentId string,
		filterReferences []string,
		allowAll bool,
	) ([]lib_models_aux_deployments.VolumeResult, error)
	DeleteUnusedVolumes(
		ctx context.Context,
		deploymentId string,
		excludeReferences []string,
	) ([]lib_models_aux_deployments.VolumeResult, error)
	DeleteMutex(deploymentId string)
}

type globalConfigsHandler interface {
	CreateGlobalConfig(ctx context.Context, name string, value models_configs.Value) (string, error)
	ReadGlobalConfig(ctx context.Context, id string) (models_configs.Config, error)
	ReadGlobalConfigs(ctx context.Context, ids []string) (map[string]models_configs.Config, error)
	UpdateGlobalConfig(ctx context.Context, config models_configs.Config) error
	DeleteGlobalConfig(ctx context.Context, id string) error
	DeleteGlobalConfigs(ctx context.Context, ids []string, allowAll bool) error
}

type deploymentAdvertisementsHandler interface {
	GetAdvertisement(
		ctx context.Context,
		deploymentId string,
		reference string,
	) (lib_models_dep_advertisements.DeploymentAdvertisement, error)
	GetAdvertisementById(ctx context.Context, id string) (lib_models_dep_advertisements.DeploymentAdvertisement, error)
	GetAdvertisements(
		ctx context.Context,
		filter lib_models_dep_advertisements.DeploymentAdvertisementsFilter,
	) (map[string]lib_models_dep_advertisements.DeploymentAdvertisement, error)
	PutAdvertisement(
		ctx context.Context,
		moduleId string,
		deploymentId string,
		reference string,
		items map[string]string,
	) (string, error)
	PutAdvertisements(
		ctx context.Context,
		moduleId string,
		deploymentId string,
		inputs []lib_models_dep_advertisements.DeploymentAdvertisementInput,
		incremental bool,
	) (map[string]string, error)
	DeleteAdvertisements(
		ctx context.Context,
		deploymentId string,
		filter lib_models_dep_advertisements.DeploymentAdvertisementsFilterReduced,
		allowAll bool,
	) error
}
