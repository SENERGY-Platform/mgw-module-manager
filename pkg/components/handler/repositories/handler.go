package repositories

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"sync"

	lib_errors "github.com/SENERGY-Platform/mgw-module-manager/lib/errors"
	helper_errors "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/errors"
	helper_modfile "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/modfile"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants/slog_keys"
)

type Handler struct {
	repositories map[string]Repository
	variantsMap  map[string]map[string]map[string]moduleWrapper // {moduleId:{source:{channel:variant}}}
	mu           sync.RWMutex
}

func New(repositories []Repository) *Handler {
	tmp := make(map[string]Repository)
	for _, repo := range repositories {
		tmp[repo.Handler.Source()] = repo
	}
	return &Handler{
		repositories: tmp,
	}
}

func (h *Handler) InitRepositories(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	var errs []error
	for source, repo := range h.repositories {
		err := repo.Handler.Init()
		if err != nil {
			logger.Error("initialize repository", slog_keys.Source, source, slog_keys.Error, err.Error())
			errs = append(errs, fmt.Errorf("'%s' %w", source, err))
			continue
		}
		logger.Info("initialize repository", slog_keys.Source, source)
	}
	if len(errs) > 0 {
		return helper_errors.Join(errs...)
	}
	return h.updateVariantsMap(ctx)
}

func (h *Handler) RefreshRepositories(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	var errs []error
	for source, repo := range h.repositories {
		err := repo.Handler.Refresh(ctx)
		if err != nil {
			logger.Error("refresh repository", slog_keys.Source, source, slog_keys.Error, err.Error())
			errs = append(errs, fmt.Errorf("'%s' %w", source, err))
			continue
		}
		logger.Info("refresh repository", slog_keys.Source, source)
	}
	if len(errs) > 0 {
		return helper_errors.Join(errs...)
	}
	return h.updateVariantsMap(ctx)
}

func (h *Handler) GetRepositories(_ context.Context) []pkg_models.Repository {
	h.mu.RLock()
	defer h.mu.RUnlock()
	var repos []pkg_models.Repository
	for source, repo := range h.repositories {
		repos = append(repos, pkg_models.Repository{
			Source:   source,
			Priority: repo.Priority,
			Channels: repo.Handler.Channels(),
		})
	}
	return repos
}

func (h *Handler) GetModules(_ context.Context, filter pkg_models.RepositoryModulesFilter) []pkg_models.RepositoryModule {
	h.mu.RLock()
	defer h.mu.RUnlock()
	filterById := len(filter.Ids) > 0
	filterBySource := len(filter.Sources) > 0
	filter.Name = strings.ToLower(filter.Name)
	sourceFilterMap := newSourceFilterMap(filter.Sources)
	var variants []pkg_models.RepositoryModule
	for modId, sources := range h.variantsMap {
		if filterById && !slices.Contains(filter.Ids, modId) {
			continue
		}
		for source, channels := range sources {
			for channel, variant := range channels {
				if filterBySource && !filterSources(source, channel, sourceFilterMap) {
					continue
				}
				if !strings.Contains(strings.ToLower(variant.RepositoryModule.Name), filter.Name) { // empty string = true
					continue
				}
				variants = append(variants, variant.RepositoryModule)
			}
		}
	}
	return variants
}

func (h *Handler) GetModule(_ context.Context, id, source, channel string) (pkg_models.RepositoryModule, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	variant, err := h.getModuleVariant(id, source, channel)
	if err != nil {
		return pkg_models.RepositoryModule{}, lib_errors.Wrap[lib_errors.ErrNotFound](err)
	}
	return variant.RepositoryModule, nil
}

func (h *Handler) GetModuleFS(ctx context.Context, id, source, channel string) (fs.FS, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	variant, err := h.getModuleVariant(id, source, channel)
	if err != nil {
		return nil, lib_errors.Wrap[lib_errors.ErrNotFound](err)
	}
	repo, ok := h.repositories[variant.Source]
	if !ok {
		return nil, errors.New("repository handler not found")
	}
	fSys, err := repo.Handler.FileSystem(ctx, variant.Channel, variant.FSysRef)
	if err != nil {
		return nil, err
	}
	return fSys, nil
}

func (h *Handler) updateVariantsMap(ctx context.Context) error {
	variantsMap := make(map[string]map[string]map[string]moduleWrapper)
	var errs []error
	for source, repo := range h.repositories {
		for _, channel := range repo.Handler.Channels() {
			fsMap, err := repo.Handler.FileSystemsMap(ctx, channel.Name)
			if err != nil {
				logger.Error(
					"update tree, get file systems",
					slog_keys.Source, source,
					slog_keys.Channel, channel.Name,
					slog_keys.Error, err.Error(),
				)
				errs = append(errs, fmt.Errorf("'%s' '%s' %w", source, channel.Name, err))
				continue
			}
			for ref, fSys := range fsMap {
				mod, err := helper_modfile.GetModule(fSys)
				if err != nil {
					logger.Error(
						"update tree, get module",
						slog_keys.Source, source,
						slog_keys.Channel, channel.Name,
						slog_keys.Reference, ref,
						slog_keys.Error, err.Error(),
					)
					errs = append(errs, fmt.Errorf("'%s' '%s' '%s' %w", source, channel.Name, ref, err))
					continue
				}
				sources, ok := variantsMap[mod.ID]
				if !ok {
					sources = make(map[string]map[string]moduleWrapper)
					variantsMap[mod.ID] = sources
				}
				channels, ok := sources[source]
				if !ok {
					channels = make(map[string]moduleWrapper)
					sources[source] = channels
				}
				channels[channel.Name] = moduleWrapper{
					RepositoryModule: pkg_models.RepositoryModule{
						RepositoryModuleBase: pkg_models.RepositoryModuleBase{
							Id:      mod.ID,
							Source:  source,
							Channel: channel.Name,
						},
						Name:    mod.Name,
						Desc:    mod.Description,
						Version: mod.Version,
					},
					FSysRef: ref,
				}
				logger.Debug(
					"update tree, add",
					slog_keys.Source, source,
					slog_keys.Channel, channel.Name,
					slog_keys.Reference, ref,
					slog_keys.ModuleId, mod.ID,
					slog_keys.Version, mod.Version,
				)
			}
		}
	}
	if len(errs) > 0 {
		return helper_errors.Join(errs...)
	}
	h.variantsMap = variantsMap
	return nil
}

func (h *Handler) getModuleVariant(id, source, channel string) (moduleWrapper, error) {
	sources, ok := h.variantsMap[id]
	if !ok {
		return moduleWrapper{}, errors.New("module not found")
	}
	channels, ok := sources[source]
	if !ok {
		return moduleWrapper{}, errors.New("source not found")
	}
	variant, ok := channels[channel]
	if !ok {
		return moduleWrapper{}, errors.New("channel not found")
	}
	return variant, nil
}

func newSourceFilterMap(sourceFilters []pkg_models.RepositorySourceFilter) map[string]map[string]struct{} {
	sourceFilterMap := make(map[string]map[string]struct{})
	for _, sourceFilter := range sourceFilters {
		channels, ok := sourceFilterMap[sourceFilter.Name]
		if !ok {
			channels = make(map[string]struct{})
			sourceFilterMap[sourceFilter.Name] = channels
		}
		for _, channel := range sourceFilter.Channels {
			channels[channel] = struct{}{}
		}
	}
	return sourceFilterMap
}

func filterSources(source, channel string, filter map[string]map[string]struct{}) bool {
	channels, ok := filter[source]
	if !ok {
		return false
	}
	if len(channels) > 0 {
		_, ok = channels[channel]
		return ok
	}
	return true
}
