package mod_repos_hdl

import (
	"context"
	"errors"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/modfile_util"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	"io/fs"
	"sync"
)

type Handler struct {
	repoHandlers  map[string]RepoHandler
	defaultSource string
	variantsMap   map[string]map[string]map[string]moduleVariant // {moduleID:{source:{channel:variant}}}
	mu            sync.RWMutex
}

func New(defaultSource string, repoHandlers []RepoHandler) *Handler {
	handlerMap := make(map[string]RepoHandler)
	for _, handler := range repoHandlers {
		handlerMap[handler.Source()] = handler
	}
	return &Handler{
		defaultSource: defaultSource,
		repoHandlers:  handlerMap,
	}
}

func (h *Handler) InitRepositories(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	var errs []error
	for source, handler := range h.repoHandlers {
		if err := handler.Init(); err != nil {
			errs = append(errs, models.NewRepoErr(source, err))
		}
	}
	if len(errs) > 0 {
		return models.NewMultiError(errs)
	}
	return h.updateVariantsMap(ctx)
}

func (h *Handler) RefreshRepositories(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	var errs []error
	for source, handler := range h.repoHandlers {
		if err := handler.Refresh(ctx); err != nil {
			errs = append(errs, models.NewRepoErr(source, err))
		}
	}
	if len(errs) > 0 {
		return models.NewMultiError(errs)
	}
	return h.updateVariantsMap(ctx)
}

func (h *Handler) SetDefaultRepository(source string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.repoHandlers[source]; !ok {
		return errors.New("source does not exist")
	}
	h.defaultSource = source
	return nil
}

func (h *Handler) Repositories(_ context.Context) (map[string]models.Repository, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	reposMap := make(map[string]models.Repository)
	for source, handler := range h.repoHandlers {
		reposMap[source] = models.Repository{
			Source:         source,
			Default:        source == h.defaultSource,
			Channels:       handler.Channels(),
			DefaultChannel: handler.DefaultChannel(),
		}
	}
	return reposMap, nil
}

func (h *Handler) Modules(_ context.Context) ([]models.RepoModuleVariant, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	var variants []models.RepoModuleVariant
	for _, sources := range h.variantsMap {
		for _, channels := range sources {
			for _, variant := range channels {
				variants = append(variants, variant.RepoModuleVariant)
			}
		}
	}
	return variants, nil
}

func (h *Handler) Module(_ context.Context, id, source, channel string) (models.RepoModuleVariant, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	variant, err := h.getModuleVariant(id, source, channel)
	if err != nil {
		return models.RepoModuleVariant{}, err
	}
	return variant.RepoModuleVariant, nil
}

func (h *Handler) ModuleFS(ctx context.Context, id, source, channel string) (fs.FS, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	variant, err := h.getModuleVariant(id, source, channel)
	if err != nil {
		return nil, err
	}
	repoHandler, ok := h.repoHandlers[variant.Source]
	if !ok {
		return nil, errors.New("repo handler not found")
	}
	fSys, err := repoHandler.FileSystem(ctx, variant.Channel, variant.FSysRef)
	if err != nil {
		return nil, err
	}
	return fSys, nil
}

func (h *Handler) updateVariantsMap(ctx context.Context) error {
	variantsMap := make(map[string]map[string]map[string]moduleVariant)
	var errs []error
	for source, handler := range h.repoHandlers {
		for _, channel := range handler.Channels() {
			fsMap, err := handler.FileSystemsMap(ctx, channel)
			if err != nil {
				errs = append(errs, models.NewRepoModuleErr(source, channel, err))
				continue
			}
			for ref, fSys := range fsMap {
				if ctx.Err() != nil {
					return ctx.Err()
				}
				mod, err := modfile_util.GetModule(fSys)
				if err != nil {
					errs = append(errs, models.NewRepoModuleErr(source, channel, err))
					continue
				}
				sources, ok := variantsMap[mod.ID]
				if !ok {
					sources = make(map[string]map[string]moduleVariant)
					variantsMap[mod.ID] = sources
				}
				channels, ok := sources[source]
				if !ok {
					channels = make(map[string]moduleVariant)
					sources[source] = channels
				}
				channels[channel] = moduleVariant{
					RepoModuleVariant: models.RepoModuleVariant{
						RepoModuleVariantBase: models.RepoModuleVariantBase{
							ID:      mod.ID,
							Source:  source,
							Channel: channel,
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
		return models.NewMultiError(errs)
	}
	h.variantsMap = variantsMap
	return nil
}

func (h *Handler) getModuleVariant(id, source, channel string) (moduleVariant, error) {
	sources, ok := h.variantsMap[id]
	if !ok {
		return moduleVariant{}, errors.New("module not found")
	}
	channels, ok := sources[source]
	if !ok {
		return moduleVariant{}, errors.New("source not found")
	}
	variant, ok := channels[channel]
	if !ok {
		return moduleVariant{}, errors.New("channel not found")
	}
	return variant, nil
}
