package repositories

import (
	"context"
	"io/fs"

	models_repositories "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/repositories"
)

type repositoryHandler interface {
	Init() error
	Source() string
	Channels() []models_repositories.Channel
	Refresh(ctx context.Context) error
	FileSystemsMap(ctx context.Context, channel string) (map[string]fs.FS, error)
	FileSystem(ctx context.Context, channel, fsRef string) (fs.FS, error)
}
