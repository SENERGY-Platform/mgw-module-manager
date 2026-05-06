package service

import (
	"context"
	"io/fs"

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	models_configs "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/configs"
	models_deployments "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/deployments"
	models_module "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/modules"
	models_repositories "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/repositories"
)

type repositoriesHandler interface {
	RefreshRepositories(ctx context.Context) error
	Repositories(ctx context.Context) ([]models_repositories.Repository, error)
	Module(ctx context.Context, id, source, channel string) (models_repositories.Module, error)
	Modules(ctx context.Context, filter models_repositories.ModulesFilter) ([]models_repositories.Module, error)
	ModuleFS(ctx context.Context, id, source, channel string) (fs.FS, error)
}

type modulesHandler interface {
	Modules(ctx context.Context, filter models_module.ModulesFilterWithNameAndDep) (map[string]models_module.Module, error)
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
	) ([]lib_models.DeploymentResult, error)
	UpdateDeployments(
		ctx context.Context,
		selectedModules map[string]models_module.Module,
		userInputs map[string]models_deployments.UserInput,
	) ([]lib_models.DeploymentResult, error)
	RecreateDeployments(
		ctx context.Context,
		selectedModules map[string]models_module.Module,
	) ([]lib_models.DeploymentResult, error)
	DeleteDeployments(
		ctx context.Context,
		filter models_deployments.DeploymentsFilterWithState,
		allowAll bool,
	) ([]lib_models.DeploymentResult, error)
	EnableDeployments(ctx context.Context, moduleIds []string) ([]string, error)
	DisableDeployments(ctx context.Context, moduleIds []string) ([]string, error)
}

type auxiliaryDeploymentsHandler interface {
	GetDeployment(
		ctx context.Context,
		deploymentId string,
		auxDeploymentId string,
	) (lib_models.AuxiliaryDeployment, error)
	GetDeployments(
		ctx context.Context,
		deploymentId string,
		filter lib_models.AuxiliaryDeploymentsFilterWithState,
	) (map[string]lib_models.AuxiliaryDeployment, error)
	GetReducedDeployments(
		ctx context.Context,
		deploymentId string,
		filter lib_models.AuxiliaryDeploymentsFilterWithState,
	) (map[string]lib_models.AuxiliaryDeploymentReduced, error)
	CreateDeployment(
		ctx context.Context,
		module models_module.Module,
		activeDeployment models_deployments.Deployment,
		dependencies map[string]models_deployments.DeploymentReduced,
		serviceInput lib_models.AuxiliaryDeploymentInputBase,
	) (lib_models.AuxiliaryDeploymentResult, error)
	UpdateDeployment(
		ctx context.Context,
		module models_module.Module,
		activeDeployment models_deployments.Deployment,
		dependencies map[string]models_deployments.DeploymentReduced,
		auxDeploymentId string,
		serviceInput lib_models.AuxiliaryDeploymentUpdateInputBase,
	) error
	RecreateDeployments(
		ctx context.Context,
		module models_module.Module,
		activeDeployment models_deployments.Deployment,
		dependencies map[string]models_deployments.DeploymentReduced,
		filter lib_models.AuxiliaryDeploymentsFilterWithState,
	) ([]lib_models.AuxiliaryDeploymentBatchResult, error)
	DeleteDeployments(
		ctx context.Context,
		deploymentId string,
		filter lib_models.AuxiliaryDeploymentsFilterWithState,
		allowAll bool,
	) ([]lib_models.AuxiliaryDeploymentBatchResult, error)
	EnableDeployments(
		ctx context.Context,
		deploymentId string,
		filter lib_models.AuxiliaryDeploymentsFilterWithState,
	) ([]string, error)
	DisableDeployments(
		ctx context.Context,
		deploymentId string,
		filter lib_models.AuxiliaryDeploymentsFilterWithState,
	) ([]string, error)
	GetVolumes(
		ctx context.Context,
		deploymentId string,
		filterReferences []string,
	) (map[string]lib_models.AuxiliaryDeploymentVolume, error)
	GetVolumesWithMounts(
		ctx context.Context,
		deploymentId string,
		filterReferences []string,
	) (map[string]lib_models.AuxiliaryDeploymentVolumeWithMounts, error)
	DeleteVolumes(
		ctx context.Context,
		deploymentId string,
		filterReferences []string,
		allowAll bool,
	) ([]lib_models.AuxiliaryDeploymentVolumeResult, error)
	DeleteUnusedVolumes(
		ctx context.Context,
		deploymentId string,
		excludeReferences []string,
	) ([]lib_models.AuxiliaryDeploymentVolumeResult, error)
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
	) (lib_models.DeploymentAdvertisement, error)
	GetAdvertisementById(ctx context.Context, id string) (lib_models.DeploymentAdvertisement, error)
	GetAdvertisements(
		ctx context.Context,
		filter lib_models.DeploymentAdvertisementsFilter,
	) (map[string]lib_models.DeploymentAdvertisement, error)
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
		inputs []lib_models.DeploymentAdvertisementInput,
		incremental bool,
	) (map[string]string, error)
	DeleteAdvertisements(
		ctx context.Context,
		deploymentId string,
		filter lib_models.DeploymentAdvertisementsFilterReduced,
		allowAll bool,
	) error
}
