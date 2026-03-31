package service

import (
	"context"
	"io/fs"

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
	) error
	UpdateDeployments(
		ctx context.Context,
		selectedModules map[string]models_handler_modules.Module,
		userInputs map[string]models_handler_deployments.UserInput,
	) error
	RecreateDeployments(ctx context.Context, selectedModules map[string]models_handler_modules.Module) error
	DeleteDeployments(ctx context.Context, filter models_handler_deployments.DeploymentsFilter) error
	EnableDeployments(ctx context.Context, moduleIds []string) error
	DisableDeployments(ctx context.Context, moduleIds []string) error
}
