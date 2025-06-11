package mod_repos_hdl

import (
	"context"
	"io/fs"
)

type RepoHandler interface {
	Init() error
	Source() string
	Channels() []string
	DefaultChannel() string
	Refresh(ctx context.Context) error
	FileSystemsMap(ctx context.Context, channel string) (map[string]fs.FS, error)
	FileSystem(ctx context.Context, channel, fsRef string) (fs.FS, error)
}
