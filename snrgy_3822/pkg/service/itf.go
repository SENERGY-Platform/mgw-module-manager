package service

import (
	"context"
	"io/fs"

	models_module "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/module"
	models_repo "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/repository"
)

type RepositoriesHandler interface {
	RefreshRepositories(ctx context.Context) error
	Repositories(ctx context.Context) ([]models_repo.Repository, error)
	Module(ctx context.Context, id, source, channel string) (models_repo.Module, error)
	Modules(ctx context.Context) ([]models_repo.Module, error)
	ModuleFS(ctx context.Context, id, source, channel string) (fs.FS, error)
}

type ModulesHandler interface {
	Modules(ctx context.Context, filter models_module.ModuleFilter) ([]models_module.ModuleAbbreviated, error)
	Module(ctx context.Context, id string) (models_module.Module, error)
	ModuleFS(ctx context.Context, id string) (fs.FS, error)
	Add(ctx context.Context, id, source, channel string, fSys fs.FS) error
	Update(ctx context.Context, id, source, channel string, fSys fs.FS) error
	Remove(ctx context.Context, id string) error
}

type Logger interface {
	Error(v ...any)
}
