package github

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"slices"
	"sync"

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	helper_archive "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/archive"
	helper_errors "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/errors"
)

const gitHubCom = "github.com"

var commonBlacklist = []string{
	".git",
	".github",
}

type Repository struct {
	gitHubClt   gitHubClient
	source      Source
	workdirPath string
	mu          sync.RWMutex
}

func newRepository(gitHubClt gitHubClient, source Source, workdirPath string) *Repository {
	return &Repository{
		gitHubClt:   gitHubClt,
		source:      source,
		workdirPath: workdirPath,
	}
}

func (r *Repository) Type() string {
	return gitHubCom
}

func (r *Repository) Priority() int {
	return r.source.Priority
}

func (r *Repository) Source() string {
	return getSourceString(r.source)
}

func (r *Repository) Channels() []lib_models.RepositoryChannel {
	var channels []lib_models.RepositoryChannel
	for _, channel := range r.source.Channels {
		channels = append(channels, lib_models.RepositoryChannel{Name: channel.Name, Priority: channel.Priority})
	}
	return channels
}

func (r *Repository) GetFileSystemsMap(_ context.Context, channelName string) (map[string]fs.FS, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	i := slices.IndexFunc(r.source.Channels, func(channel Channel) bool {
		return channel.Name == channelName
	})
	if i < 0 {
		return nil, errors.New("channel not found")
	}
	channel := r.source.Channels[i]
	repo, err := readRepoFile(r.workdirPath)
	if err != nil {
		return nil, err
	}
	dirEntries, err := os.ReadDir(path.Join(r.workdirPath, repo.Path, channel.Name))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	fsMap := make(map[string]fs.FS)
	for _, entry := range dirEntries {
		if entry.IsDir() && !slices.Contains(commonBlacklist, entry.Name()) && !slices.Contains(channel.Blacklist, entry.Name()) {
			fsMap[entry.Name()] = os.DirFS(path.Join(r.workdirPath, repo.Path, channel.Name, entry.Name()))
		}
	}
	return fsMap, nil
}

func (r *Repository) GetFileSystem(_ context.Context, channelName, fsRef string) (fs.FS, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	i := slices.IndexFunc(r.source.Channels, func(channel Channel) bool {
		return channel.Name == channelName
	})
	if i < 0 {
		return nil, errors.New("channel not found")
	}
	channel := r.source.Channels[i]
	repo, err := readRepoFile(r.workdirPath)
	if err != nil {
		return nil, err
	}
	dirEntries, err := os.ReadDir(path.Join(r.workdirPath, repo.Path, channel.Name))
	if err != nil {
		return nil, err
	}
	for _, entry := range dirEntries {
		if entry.IsDir() && entry.Name() == fsRef {
			return os.DirFS(path.Join(r.workdirPath, repo.Path, channel.Name, entry.Name())), nil
		}
	}
	return nil, errors.New("reference not found")
}

func (r *Repository) Refresh(ctx context.Context) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	err := os.MkdirAll(r.workdirPath, 0775)
	if err != nil {
		return err
	}
	oldRepo, err := readRepoFile(r.workdirPath)
	if err != nil {
		return err
	}
	var newRepo repoFile
	newRepo.GitCommit, err = r.gitHubClt.GetLastCommit(ctx, r.source.Owner, r.source.Repository, r.source.Reference)
	if err != nil {
		return err
	}
	if newRepo.GitCommit.Sha == oldRepo.GitCommit.Sha {
		return nil
	}
	repoArchive, err := r.gitHubClt.GetRepoTarGzArchive(ctx, r.source.Owner, r.source.Repository, newRepo.GitCommit.Sha)
	if err != nil {
		return err
	}
	defer repoArchive.Close()
	if err = os.MkdirAll(path.Join(r.workdirPath, newRepo.GitCommit.Sha), 0775); err != nil {
		return err
	}
	rootDir, err := helper_archive.ExtractTarGz(repoArchive, path.Join(r.workdirPath, newRepo.GitCommit.Sha))
	if err != nil {
		_, _ = io.ReadAll(repoArchive)
		return err
	}
	newRepo.Path = path.Join(newRepo.GitCommit.Sha, rootDir)
	if err = writeRepoFile(r.workdirPath, newRepo); err != nil {
		if e := os.RemoveAll(path.Join(r.workdirPath, newRepo.GitCommit.Sha)); e != nil {
			return helper_errors.Join(err, e)
		}
		return err
	}
	if oldRepo.Path != "" && oldRepo.Path != newRepo.Path {
		fmt.Println(path.Join(r.workdirPath, oldRepo.Path))
		if e := os.RemoveAll(path.Join(r.workdirPath, oldRepo.GitCommit.Sha)); e != nil {
			fmt.Println(e)
		}
	}
	return nil
}
