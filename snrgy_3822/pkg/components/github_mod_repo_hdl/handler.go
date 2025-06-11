package github_mod_repo_hdl

import (
	"context"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/archive_util"
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
	for _, reference := range h.channels {
		if err = os.MkdirAll(path.Join(h.wrkPath, reference.Name), 0775); err != nil {
			return err
		}
	}
	return nil
}

func (h *Handler) Source() string {
	return path.Join(gitHubCom, h.owner, h.repo)
}

func (h *Handler) Channels() []string {
	var channels []string
	for _, channel := range h.channels {
		channels = append(channels, channel.Name)
	}
	return channels
}

func (h *Handler) DefaultChannel() string {
	for _, channel := range h.channels {
		if channel.Default {
			return channel.Name
		}
	}
	return ""
}

func (h *Handler) FileSystemsMap(ctx context.Context, channelID string) (map[string]fs.FS, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	channel, ok := h.channels[channelID]
	if !ok {
		return nil, errors.New("channel does not exist")
	}
	repo, err := readRepoFile(path.Join(h.wrkPath, channel.Name))
	if err != nil {
		return nil, err
	}
	dirEntries, err := os.ReadDir(path.Join(h.wrkPath, channel.Name, repo.Source))
	if err != nil {
		return nil, err
	}
	fsMap := make(map[string]fs.FS)
	for _, entry := range dirEntries {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if entry.IsDir() && !inSlice(commonBlacklist, entry.Name()) && !inSlice(channel.Blacklist, entry.Name()) {
			fsMap[entry.Name()] = os.DirFS(path.Join(h.wrkPath, channel.Name, repo.Source, entry.Name()))
		}
	}
	return fsMap, nil
}

func (h *Handler) FileSystem(ctx context.Context, channelID, fsRef string) (fs.FS, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	channel, ok := h.channels[channelID]
	if !ok {
		return nil, errors.New("channel does not exist")
	}
	repo, err := readRepoFile(path.Join(h.wrkPath, channel.Name))
	if err != nil {
		return nil, err
	}
	dirEntries, err := os.ReadDir(path.Join(h.wrkPath, channel.Name, repo.Source))
	if err != nil {
		return nil, err
	}
	for _, entry := range dirEntries {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if entry.IsDir() && entry.Name() == fsRef {
			return os.DirFS(path.Join(h.wrkPath, channel.Name, repo.Source, entry.Name())), nil
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
	oldRepo, err := readRepoFile(path.Join(h.wrkPath, channel.Name))
	if err != nil {
		return err
	}
	var newRepo repoFile
	ctxWt1, cf1 := context.WithTimeout(ctx, h.httpTimeout)
	defer cf1()
	newRepo.GitCommit, err = h.gitHubClt.GetLastCommit(ctxWt1, h.owner, h.repo, channel.Reference)
	if err != nil {
		return err
	}
	if newRepo.GitCommit.Sha == oldRepo.GitCommit.Sha {
		return nil
	}
	ctxWt2, cf2 := context.WithTimeout(ctx, h.httpTimeout)
	defer cf2()
	repoArchive, err := h.gitHubClt.GetRepoTarGzArchive(ctxWt2, h.owner, h.repo, newRepo.GitCommit.Sha)
	if err != nil {
		return err
	}
	defer repoArchive.Close()
	newRepo.Source, err = archive_util.ExtractTarGz(repoArchive, path.Join(h.wrkPath, channel.Name))
	if err != nil {
		_, _ = io.ReadAll(repoArchive)
		return err
	}
	if err = writeRepoFile(path.Join(h.wrkPath, channel.Name), newRepo); err != nil {
		if e := os.RemoveAll(path.Join(path.Join(h.wrkPath, channel.Name), newRepo.Source)); e != nil {
			return errors.Join(err, e)
		}
		return err
	}
	if oldRepo.Source != "" && oldRepo.Source != newRepo.Source {
		if e := os.RemoveAll(path.Join(path.Join(h.wrkPath, channel.Name), oldRepo.Source)); e != nil {
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
