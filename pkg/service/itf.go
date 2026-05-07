package service

import (
	"context"
	"io/fs"

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
)

type repositoriesHandler interface {
	RefreshRepositories(ctx context.Context) error
	Repositories(ctx context.Context) []pkg_models.Repository
	Module(ctx context.Context, id, source, channel string) (pkg_models.RepositoryModule, error)
	Modules(ctx context.Context, filter pkg_models.RepositoryModulesFilter) []pkg_models.RepositoryModule
	ModuleFS(ctx context.Context, id, source, channel string) (fs.FS, error)
}

type modulesHandler interface {
	Modules(ctx context.Context, filter pkg_models.ModulesFilterWithNameAndDep) (map[string]pkg_models.Module, error)
	Module(ctx context.Context, id string) (pkg_models.Module, error)
	Add(ctx context.Context, id, source, channel string, fSys fs.FS) error
	Update(ctx context.Context, id, source, channel string, fSys fs.FS) error
	Remove(ctx context.Context, id string) error
}

type deploymentsHandler interface {
	GetDeployment(ctx context.Context, id string) (pkg_models.Deployment, error)
	GetReducedDeploymentsByModuleIds(
		ctx context.Context,
		filter pkg_models.DeploymentsFilterWithState,
	) (map[string]pkg_models.DeploymentReduced, error)
	GetDeploymentsByModuleIds(
		ctx context.Context,
		filter pkg_models.DeploymentsFilterWithState,
	) (map[string]pkg_models.Deployment, error)
	GetDeploymentByModuleId(ctx context.Context, moduleId string) (pkg_models.Deployment, error)
	GetDeploymentIds(ctx context.Context, filter pkg_models.DeploymentsFilter) (map[string]string, error)
	CreateDeployments(
		ctx context.Context,
		selectedModules map[string]pkg_models.Module,
		userInputs map[string]pkg_models.DeploymentUserInput,
	) ([]lib_models.DeploymentResult, error)
	UpdateDeployments(
		ctx context.Context,
		selectedModules map[string]pkg_models.Module,
		userInputs map[string]pkg_models.DeploymentUserInput,
	) ([]lib_models.DeploymentResult, error)
	RecreateDeployments(
		ctx context.Context,
		selectedModules map[string]pkg_models.Module,
	) ([]lib_models.DeploymentResult, error)
	DeleteDeployments(
		ctx context.Context,
		filter pkg_models.DeploymentsFilterWithState,
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
		module pkg_models.Module,
		activeDeployment pkg_models.Deployment,
		dependencies map[string]pkg_models.DeploymentReduced,
		serviceInput lib_models.AuxiliaryDeploymentInputBase,
	) (lib_models.AuxiliaryDeploymentResult, error)
	UpdateDeployment(
		ctx context.Context,
		module pkg_models.Module,
		activeDeployment pkg_models.Deployment,
		dependencies map[string]pkg_models.DeploymentReduced,
		auxDeploymentId string,
		serviceInput lib_models.AuxiliaryDeploymentUpdateInputBase,
	) error
	RecreateDeployments(
		ctx context.Context,
		module pkg_models.Module,
		activeDeployment pkg_models.Deployment,
		dependencies map[string]pkg_models.DeploymentReduced,
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
	CreateGlobalConfig(ctx context.Context, name string, value pkg_models.Value) (string, error)
	ReadGlobalConfig(ctx context.Context, id string) (pkg_models.Config, error)
	ReadGlobalConfigs(ctx context.Context, ids []string) (map[string]pkg_models.Config, error)
	UpdateGlobalConfig(ctx context.Context, config pkg_models.Config) error
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
