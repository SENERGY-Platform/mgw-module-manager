package service

import (
	"context"
	models_repo "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/repository"
	"io/fs"
)

type ModuleReposHandler interface {
	RefreshRepositories(ctx context.Context) error
	SetDefaultRepository(source string) error
	Repositories(ctx context.Context) (map[string]models_repo.Repository, error)
	Modules(ctx context.Context) ([]models_repo.Module, error)
	ModuleFS(ctx context.Context, id, source, channel string) (fs.FS, error)
}
