package handler_repositories

import (
	"context"
	"io/fs"

	models_handler_repositories "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/repositories"
)

type repositoryHandler interface {
	Init() error
	Source() string
	Channels() []models_handler_repositories.Channel
	Refresh(ctx context.Context) error
	FileSystemsMap(ctx context.Context, channel string) (map[string]fs.FS, error)
	FileSystem(ctx context.Context, channel, fsRef string) (fs.FS, error)
}
