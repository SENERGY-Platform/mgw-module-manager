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
	"strings"
	"sync"

	helper_archive "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/archive"
	helper_errors "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/errors"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
)

const gitHubCom = "github.com"

var commonBlacklist = []string{
	".git",
	".github",
}

type Handler struct {
	gitHubClt gitHubClient
	owner     string
	repo      string
	reference string
	channels  map[string]Channel
	wrkPath   string
	mu        sync.RWMutex
}

func New(gitHubClt gitHubClient, wrkPath, owner, repo, reference string, channels []Channel) *Handler {
	channelsMap := make(map[string]Channel)
	for _, channel := range channels {
		channelsMap[channel.Name] = channel
	}
	return &Handler{
		gitHubClt: gitHubClt,
		owner:     owner,
		repo:      repo,
		reference: reference,
		channels:  channelsMap,
		wrkPath:   path.Join(wrkPath, strings.Replace(strings.Replace(gitHubCom+"_"+owner+"_"+repo+"_"+reference, "/", "_", -1), ".", "_", -1)),
	}
}

func (h *Handler) Init() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	err := os.MkdirAll(h.wrkPath, 0775)
	if err != nil {
		return err
	}
	return nil
}

func (h *Handler) Source() string {
	return path.Join(gitHubCom, h.owner, h.repo)
}

func (h *Handler) Channels() []pkg_models.RepositoryChannel {
	var channels []pkg_models.RepositoryChannel
	for _, channel := range h.channels {
		channels = append(channels, pkg_models.RepositoryChannel{Name: channel.Name, Priority: channel.Priority})
	}
	return channels
}

func (h *Handler) FileSystemsMap(_ context.Context, channelName string) (map[string]fs.FS, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	channel, ok := h.channels[channelName]
	if !ok {
		return nil, errors.New("channel not found")
	}
	repo, err := readRepoFile(h.wrkPath)
	if err != nil {
		return nil, err
	}
	dirEntries, err := os.ReadDir(path.Join(h.wrkPath, repo.Path, channel.Name))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	fsMap := make(map[string]fs.FS)
	for _, entry := range dirEntries {
		if entry.IsDir() && !slices.Contains(commonBlacklist, entry.Name()) && !slices.Contains(channel.Blacklist, entry.Name()) {
			fsMap[entry.Name()] = os.DirFS(path.Join(h.wrkPath, repo.Path, channel.Name, entry.Name()))
		}
	}
	return fsMap, nil
}

func (h *Handler) FileSystem(_ context.Context, channelName, fsRef string) (fs.FS, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	channel, ok := h.channels[channelName]
	if !ok {
		return nil, errors.New("channel not found")
	}
	repo, err := readRepoFile(h.wrkPath)
	if err != nil {
		return nil, err
	}
	dirEntries, err := os.ReadDir(path.Join(h.wrkPath, repo.Path, channel.Name))
	if err != nil {
		return nil, err
	}
	for _, entry := range dirEntries {
		if entry.IsDir() && entry.Name() == fsRef {
			return os.DirFS(path.Join(h.wrkPath, repo.Path, channel.Name, entry.Name())), nil
		}
	}
	return nil, errors.New("reference not found")
}

func (h *Handler) Refresh(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	oldRepo, err := readRepoFile(h.wrkPath)
	if err != nil {
		return err
	}
	var newRepo repoFile
	newRepo.GitCommit, err = h.gitHubClt.GetLastCommit(ctx, h.owner, h.repo, h.reference)
	if err != nil {
		return err
	}
	if newRepo.GitCommit.Sha == oldRepo.GitCommit.Sha {
		return nil
	}
	repoArchive, err := h.gitHubClt.GetRepoTarGzArchive(ctx, h.owner, h.repo, newRepo.GitCommit.Sha)
	if err != nil {
		return err
	}
	defer repoArchive.Close()
	if err = os.MkdirAll(path.Join(h.wrkPath, newRepo.GitCommit.Sha), 0775); err != nil {
		return err
	}
	rootDir, err := helper_archive.ExtractTarGz(repoArchive, path.Join(h.wrkPath, newRepo.GitCommit.Sha))
	if err != nil {
		_, _ = io.ReadAll(repoArchive)
		return err
	}
	newRepo.Path = path.Join(newRepo.GitCommit.Sha, rootDir)
	if err = writeRepoFile(h.wrkPath, newRepo); err != nil {
		if e := os.RemoveAll(path.Join(h.wrkPath, newRepo.GitCommit.Sha)); e != nil {
			return helper_errors.Join(err, e)
		}
		return err
	}
	if oldRepo.Path != "" && oldRepo.Path != newRepo.Path {
		fmt.Println(path.Join(h.wrkPath, oldRepo.Path))
		if e := os.RemoveAll(path.Join(h.wrkPath, oldRepo.GitCommit.Sha)); e != nil {
			fmt.Println(e)
		}
	}
	return nil
}
