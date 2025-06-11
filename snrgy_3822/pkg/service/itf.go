package service

import (
	"context"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	"io/fs"
)

type ModuleReposHandler interface {
	RefreshRepositories(ctx context.Context) error
	SetDefaultRepository(source string) error
	Repositories(ctx context.Context) (map[string]models.Repository, error)
	Modules(ctx context.Context) ([]models.RepoModuleVariant, error)
	ModuleFS(ctx context.Context, id, source, channel string) (fs.FS, error)
}
