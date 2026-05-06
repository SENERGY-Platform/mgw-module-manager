package repositories

import (
	"context"
	"io/fs"

	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
)

type repositoryHandler interface {
	Init() error
	Source() string
	Channels() []pkg_models.RepositoryChannel
	Refresh(ctx context.Context) error
	FileSystemsMap(ctx context.Context, channel string) (map[string]fs.FS, error)
	FileSystem(ctx context.Context, channel, fsRef string) (fs.FS, error)
}
