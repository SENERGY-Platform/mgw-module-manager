package repositories

import (
	"context"
	"io/fs"

	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
)

type Repository interface {
	Type() string
	Priority() int
	Source() string
	Channels() []pkg_models.RepositoryChannel
	Init(ctx context.Context) error
	Refresh(ctx context.Context) error
	GetFileSystemsMap(ctx context.Context, channel string) (map[string]fs.FS, error)
	GetFileSystem(ctx context.Context, channel, fsRef string) (fs.FS, error)
}

type repositoryHandler interface {
	RepositoryType() string
	Init(ctx context.Context) error
	GetRepositories(ctx context.Context) (map[string]Repository, error)
	GetRepository(ctx context.Context, source string) (Repository, error)
	CreateRepository(ctx context.Context, data []byte) (Repository, error)
	DeleteRepository(ctx context.Context, source string) error
}
