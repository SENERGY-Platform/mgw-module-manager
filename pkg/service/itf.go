package service

import (
	"context"
	"io/fs"

	models_handler_module "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/module"
	models_handler_repo "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/repository"
)

type repositoriesHandler interface {
	RefreshRepositories(ctx context.Context) error
	Repositories(ctx context.Context) ([]models_handler_repo.Repository, error)
	Module(ctx context.Context, id, source, channel string) (models_handler_repo.Module, error)
	Modules(ctx context.Context, filter models_handler_repo.ModulesFilter) ([]models_handler_repo.Module, error)
	ModuleFS(ctx context.Context, id, source, channel string) (fs.FS, error)
}

type modulesHandler interface {
	Modules(ctx context.Context, filter models_handler_module.ModuleFilter) (map[string]models_handler_module.Module, error)
	Module(ctx context.Context, id string) (models_handler_module.Module, error)
	Add(ctx context.Context, id, source, channel string, fSys fs.FS) error
	Update(ctx context.Context, id, source, channel string, fSys fs.FS) error
	Remove(ctx context.Context, id string) error
}
