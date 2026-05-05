package service

import (
	"context"
	"io/fs"

	"github.com/SENERGY-Platform/mgw-module-manager/lib/models/aux_deployments"
	models_config "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/config"
	models_handler_aux_deployments "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/aux_deployments"
	models_handler_database "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/database"
	models_handler_dep_advertisements "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/dep_advertisements"
	models_handler_deployments "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/deployments"
	models_handler_global_configs "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/global_configs"
	models_handler_modules "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/modules"
	models_handler_repositories "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/repositories"
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
	GetDeployment(ctx context.Context, id string) (models_handler_deployments.Deployment, error)
	GetReducedDeploymentsByModuleIds(
		ctx context.Context,
		filter models_handler_deployments.DeploymentsFilter,
	) (map[string]models_handler_deployments.DeploymentReduced, error)
	GetDeploymentsByModuleIds(
		ctx context.Context,
		filter models_handler_deployments.DeploymentsFilter,
	) (map[string]models_handler_deployments.Deployment, error)
	GetDeploymentByModuleId(ctx context.Context, moduleId string) (models_handler_deployments.Deployment, error)
	GetDeploymentIds(ctx context.Context, filter models_handler_database.DeploymentsFilter) (map[string]string, error)
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
	DeleteDeployments(
		ctx context.Context,
		filter models_handler_deployments.DeploymentsFilter,
		allowAll bool,
	) ([]models_handler_deployments.Result, error)
	EnableDeployments(ctx context.Context, moduleIds []string) ([]string, error)
	DisableDeployments(ctx context.Context, moduleIds []string) ([]string, error)
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
		serviceInput aux_deployments.ServiceInput,
	) (aux_deployments.Result, error)
	UpdateDeployment(
		ctx context.Context,
		module models_handler_modules.Module,
		activeDeployment models_handler_deployments.Deployment,
		dependencies map[string]models_handler_deployments.DeploymentReduced,
		auxDeploymentId string,
		serviceInput aux_deployments.UpdateServiceInput,
	) error
	RecreateDeployments(
		ctx context.Context,
		module models_handler_modules.Module,
		activeDeployment models_handler_deployments.Deployment,
		dependencies map[string]models_handler_deployments.DeploymentReduced,
		filter models_handler_aux_deployments.AuxiliaryDeploymentsFilter,
	) ([]aux_deployments.BatchResult, error)
	DeleteDeployments(
		ctx context.Context,
		deploymentId string,
		filter models_handler_aux_deployments.AuxiliaryDeploymentsFilter,
		allowAll bool,
	) ([]aux_deployments.BatchResult, error)
	EnableDeployments(
		ctx context.Context,
		deploymentId string,
		filter models_handler_aux_deployments.AuxiliaryDeploymentsFilter,
	) ([]string, error)
	DisableDeployments(
		ctx context.Context,
		deploymentId string,
		filter models_handler_aux_deployments.AuxiliaryDeploymentsFilter,
	) ([]string, error)
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
	) ([]aux_deployments.VolumeResult, error)
	DeleteUnusedVolumes(
		ctx context.Context,
		deploymentId string,
		excludeReferences []string,
	) ([]aux_deployments.VolumeResult, error)
	DeleteMutex(deploymentId string)
}

type globalConfigsHandler interface {
	CreateGlobalConfig(ctx context.Context, name string, value models_config.Value) (string, error)
	ReadGlobalConfig(ctx context.Context, id string) (models_handler_global_configs.Config, error)
	ReadGlobalConfigs(ctx context.Context, ids []string) (map[string]models_handler_global_configs.Config, error)
	UpdateGlobalConfig(ctx context.Context, config models_handler_global_configs.Config) error
	DeleteGlobalConfig(ctx context.Context, id string) error
	DeleteGlobalConfigs(ctx context.Context, ids []string, allowAll bool) error
}

type deploymentAdvertisementsHandler interface {
	GetAdvertisement(
		ctx context.Context,
		deploymentId string,
		reference string,
	) (models_handler_database.DeploymentAdvertisement, error)
	GetAdvertisementById(ctx context.Context, id string) (models_handler_database.DeploymentAdvertisement, error)
	GetAdvertisements(
		ctx context.Context,
		filter models_handler_database.DeploymentAdvertisementsFilter,
	) (map[string]models_handler_database.DeploymentAdvertisement, error)
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
		inputs []models_handler_dep_advertisements.DeploymentAdvertisementInput,
		incremental bool,
	) (map[string]string, error)
	DeleteAdvertisements(
		ctx context.Context,
		deploymentId string,
		filter models_handler_database.DeploymentAdvertisementsFilterReduced,
		allowAll bool,
	) error
}
