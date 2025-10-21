package repositories

import (
	"context"
	"errors"
	"io/fs"
	"sync"

	helper_modfile "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/modfile"
	models_error "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
	models_repo "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/repository"
)

type Handler struct {
	repositories map[string]Repository
	variantsMap  map[string]map[string]map[string]moduleWrapper // {moduleID:{source:{channel:variant}}}
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
		if err := repo.Handler.Init(); err != nil {
			errs = append(errs, models_error.NewRepoErr(source, err))
		}
	}
	if len(errs) > 0 {
		return models_error.NewMultiError(errs)
	}
	return h.updateVariantsMap(ctx)
}

func (h *Handler) RefreshRepositories(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	var errs []error
	for source, repo := range h.repositories {
		if err := repo.Handler.Refresh(ctx); err != nil {
			errs = append(errs, models_error.NewRepoErr(source, err))
		}
	}
	if len(errs) > 0 {
		return models_error.NewMultiError(errs)
	}
	return h.updateVariantsMap(ctx)
}

func (h *Handler) Repositories(_ context.Context) ([]models_repo.Repository, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	var repos []models_repo.Repository
	for source, repo := range h.repositories {
		repos = append(repos, models_repo.Repository{
			Source:   source,
			Priority: repo.Priority,
			Channels: repo.Handler.Channels(),
		})
	}
	return repos, nil
}

func (h *Handler) Modules(_ context.Context) ([]models_repo.Module, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	var variants []models_repo.Module
	for _, sources := range h.variantsMap {
		for _, channels := range sources {
			for _, variant := range channels {
				variants = append(variants, variant.Module)
			}
		}
	}
	return variants, nil
}

func (h *Handler) Module(_ context.Context, id, source, channel string) (models_repo.Module, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	variant, err := h.getModuleVariant(id, source, channel)
	if err != nil {
		return models_repo.Module{}, err
	}
	return variant.Module, nil
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
	var errs []error
	for source, repo := range h.repositories {
		for _, channel := range repo.Handler.Channels() {
			fsMap, err := repo.Handler.FileSystemsMap(ctx, channel.Name)
			if err != nil {
				errs = append(errs, models_error.NewRepoModuleErr(source, channel.Name, err))
				continue
			}
			for ref, fSys := range fsMap {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				mod, err := helper_modfile.GetModule(fSys)
				if err != nil {
					errs = append(errs, models_error.NewRepoModuleErr(source, channel.Name, err))
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
					Module: models_repo.Module{
						ModuleBase: models_repo.ModuleBase{
							ID:      mod.ID,
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
		return models_error.NewMultiError(errs)
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
