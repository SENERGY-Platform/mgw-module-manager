package github

import (
	"context"
	"errors"
	"fmt"
	helper_archive "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/archive"
	models_repo "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/repository"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

const gitHubCom = "github.com"

var commonBlacklist = []string{
	".git",
	".github",
}

type Handler struct {
	gitHubClt   GitHubClient
	httpTimeout time.Duration
	owner       string
	repo        string
	channels    map[string]Channel
	wrkPath     string
	mu          sync.RWMutex
}

func New(gitHubClt GitHubClient, httpTimeout time.Duration, wrkPath, owner, repo string, channels []Channel) *Handler {
	channelsMap := make(map[string]Channel)
	for _, channel := range channels {
		channelsMap[channel.Name] = channel
	}
	return &Handler{
		gitHubClt:   gitHubClt,
		httpTimeout: httpTimeout,
		owner:       owner,
		repo:        repo,
		channels:    channelsMap,
		wrkPath:     path.Join(wrkPath, strings.Replace(strings.Replace(gitHubCom+"_"+owner+"_"+repo, "/", "_", -1), ".", "_", -1)),
	}
}

func (h *Handler) Init() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	err := os.MkdirAll(h.wrkPath, 0775)
	if err != nil {
		return err
	}
	for _, channel := range h.channels {
		if err = os.MkdirAll(path.Join(h.wrkPath, channel.Name), 0775); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) Source() string {
	return path.Join(gitHubCom, h.owner, h.repo)
}

func (h *Handler) Channels() []models_repo.Channel {
	var channels []models_repo.Channel
	for _, channel := range h.channels {
		channels = append(channels, models_repo.Channel{Name: channel.Name, Priority: channel.Priority})
	}
	return channels
}

func (h *Handler) FileSystemsMap(ctx context.Context, channelName string) (map[string]fs.FS, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	channel, ok := h.channels[channelName]
	if !ok {
		return nil, errors.New("channel does not exist")
	}
	basePath := path.Join(h.wrkPath, channel.Name)
	repo, err := readRepoFile(basePath)
	if err != nil {
		return nil, err
	}
	dirEntries, err := os.ReadDir(path.Join(basePath, repo.Path))
	if err != nil {
		return nil, err
	}
	fsMap := make(map[string]fs.FS)
	for _, entry := range dirEntries {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if entry.IsDir() && !inSlice(commonBlacklist, entry.Name()) && !inSlice(channel.Blacklist, entry.Name()) {
			fsMap[entry.Name()] = os.DirFS(path.Join(basePath, repo.Path, entry.Name()))
		}
	}
	return fsMap, nil
}

func (h *Handler) FileSystem(ctx context.Context, channelName, fsRef string) (fs.FS, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	channel, ok := h.channels[channelName]
	if !ok {
		return nil, errors.New("channel does not exist")
	}
	basePath := path.Join(h.wrkPath, channel.Name)
	repo, err := readRepoFile(basePath)
	if err != nil {
		return nil, err
	}
	dirEntries, err := os.ReadDir(path.Join(basePath, repo.Path))
	if err != nil {
		return nil, err
	}
	for _, entry := range dirEntries {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if entry.IsDir() && entry.Name() == fsRef {
			return os.DirFS(path.Join(basePath, repo.Path, entry.Name())), nil
		}
	}
	return nil, errors.New("reference does not exist")
}

func (h *Handler) Refresh(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	var errs []error
	for _, channel := range h.channels {
		if err := h.refresh(ctx, channel); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func (h *Handler) refresh(ctx context.Context, channel Channel) error {
	basePath := path.Join(h.wrkPath, channel.Name)
	oldRepo, err := readRepoFile(basePath)
	if err != nil {
		return err
	}
	var newRepo repoFile
	newRepo.GitCommit, err = h.gitHubClt.GetLastCommit(ctx, h.owner, h.repo, channel.Reference)
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
	if err = os.MkdirAll(path.Join(basePath, newRepo.GitCommit.Sha), 0775); err != nil {
		return err
	}
	rootDir, err := helper_archive.ExtractTarGz(repoArchive, path.Join(basePath, newRepo.GitCommit.Sha))
	if err != nil {
		_, _ = io.ReadAll(repoArchive)
		return err
	}
	newRepo.Path = path.Join(newRepo.GitCommit.Sha, rootDir)
	if err = writeRepoFile(basePath, newRepo); err != nil {
		if e := os.RemoveAll(path.Join(basePath, newRepo.GitCommit.Sha)); e != nil {
			return errors.Join(err, e)
		}
		return err
	}
	if oldRepo.Path != "" && oldRepo.Path != newRepo.Path {
		fmt.Println(path.Join(basePath, oldRepo.Path))
		if e := os.RemoveAll(path.Join(basePath, oldRepo.GitCommit.Sha)); e != nil {
			fmt.Println(e)
		}
	}
	return nil
}

func inSlice(sl []string, c string) bool {
	for _, v := range sl {
		if v == c {
			return true
		}
	}
	return false
}
