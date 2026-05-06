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
	helper_modfile "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/modfile"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants/attr_keys"
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
	var errs []string
	for source, repo := range h.repositories {
		if err := repo.Handler.Init(); err != nil {
			errs = append(errs, fmt.Sprintf(source+": "+err.Error()))
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return h.updateVariantsMap(ctx)
}

func (h *Handler) RefreshRepositories(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	var errs []string
	for source, repo := range h.repositories {
		if err := repo.Handler.Refresh(ctx); err != nil {
			errs = append(errs, fmt.Sprintf(source+": "+err.Error()))
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return h.updateVariantsMap(ctx)
}

func (h *Handler) Repositories(_ context.Context) ([]pkg_models.Repository, error) {
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
	return repos, nil
}

func (h *Handler) Modules(_ context.Context, filter pkg_models.RepositoryModulesFilter) ([]pkg_models.RepositoryModule, error) {
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
	return variants, nil
}

func (h *Handler) Module(_ context.Context, id, source, channel string) (pkg_models.RepositoryModule, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	variant, err := h.getModuleVariant(id, source, channel)
	if err != nil {
		return pkg_models.RepositoryModule{}, err
	}
	return variant.RepositoryModule, nil
}

func (h *Handler) ModuleFS(ctx context.Context, id, source, channel string) (fs.FS, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	variant, err := h.getModuleVariant(id, source, channel)
	if err != nil {
		return nil, err
	}
	repo, ok := h.repositories[variant.Source]
	if !ok {
		return nil, errors.New("repo handler not found")
	}
	fSys, err := repo.Handler.FileSystem(ctx, variant.Channel, variant.FSysRef)
	if err != nil {
		return nil, err
	}
	return fSys, nil
}

func (h *Handler) updateVariantsMap(ctx context.Context) error {
	variantsMap := make(map[string]map[string]map[string]moduleWrapper)
	var errs []string
	for source, repo := range h.repositories {
		for _, channel := range repo.Handler.Channels() {
			fsMap, err := repo.Handler.FileSystemsMap(ctx, channel.Name)
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s %s: %s", source, channel.Name, err.Error()))
				continue
			}
			for ref, fSys := range fsMap {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				mod, err := helper_modfile.GetModule(fSys)
				if err != nil {
					errs = append(errs, fmt.Sprintf("%s %s: %s", source, channel.Name, err.Error()))
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
			}
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	h.variantsMap = variantsMap
	return nil
}

func (h *Handler) getModuleVariant(id, source, channel string) (moduleWrapper, error) {
	sources, ok := h.variantsMap[id]
	if !ok {
		return moduleWrapper{}, lib_errors.New[lib_errors.ErrNotFound]("module not found", attr_keys.ModuleId, id)
	}
	channels, ok := sources[source]
	if !ok {
		return moduleWrapper{}, lib_errors.New[lib_errors.ErrNotFound](
			"source not found",
			attr_keys.ModuleId,
			id,
			attr_keys.Source,
			source,
		)
	}
	variant, ok := channels[channel]
	if !ok {
		return moduleWrapper{}, lib_errors.New[lib_errors.ErrNotFound](
			"channel not found",
			attr_keys.ModuleId,
			id,
			attr_keys.Source,
			source,
			attr_keys.Channel,
			channel,
		)
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
