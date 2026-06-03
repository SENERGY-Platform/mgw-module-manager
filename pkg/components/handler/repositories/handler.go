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
	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	helper_errors "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/errors"
	helper_modfile "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/modfile"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants/slog_keys"
)

type Handler struct {
	repositoryHandlers map[string]repositoryHandler
	lookUpMap          map[string]map[string]map[string]moduleWrapper // {moduleId:{source:{channel:variant}}}
	mu                 sync.RWMutex
}

func New(handlers ...repositoryHandler) *Handler {
	h := Handler{
		repositoryHandlers: make(map[string]repositoryHandler),
	}
	for _, handler := range handlers {
		h.repositoryHandlers[handler.RepositoryType()] = handler
	}
	return &h
}

func (h *Handler) Init(ctx context.Context) error {
	var errs []error
	repositories := make(map[string]Repository)
	priorities := make(map[int]struct{})
	for repoType, handler := range h.repositoryHandlers {
		repos, err := handler.GetRepositories(ctx)
		if err != nil {
			errs = append(errs, fmt.Errorf("get repositories: %s %w", repoType, err))
			continue
		}
		for source, repo := range repos {
			if _, ok := repositories[source]; ok {
				errs = append(errs, errors.New(fmt.Sprintf("source collision: %s", source)))
				continue
			}
			if _, ok := priorities[repo.Priority()]; ok {
				errs = append(errs, errors.New(fmt.Sprintf("priority collision: %d", repo.Priority())))
				continue
			}
			repositories[source] = repo
		}
	}
	h.updateLookupMap(ctx, repositories)
	if len(errs) > 0 {
		return helper_errors.Join(errs...)
	}
	return nil
}

func (h *Handler) RefreshRepositories(ctx context.Context) ([]lib_models.RepositoryResult, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	repositories := make(map[string]Repository)
	priorities := make(map[int]struct{})
	var results []lib_models.RepositoryResult
	for repoType, handler := range h.repositoryHandlers {
		repos, err := handler.GetRepositories(ctx)
		if err != nil {
			logger.ErrorContext(ctx, "refresh repositories", slog_keys.RepositoryType, repoType, slog_keys.Error, err.Error())
			continue
		}
		for source, repo := range repos {
			result := lib_models.RepositoryResult{
				Type:   repoType,
				Source: source,
			}
			if _, ok := repositories[source]; ok {
				logger.ErrorContext(
					ctx,
					"refresh repositories",
					slog_keys.RepositoryType, repoType,
					slog_keys.Source, source,
					slog_keys.Error, "source collision",
				)
				result.ErrorResult = lib_models.NewErrorResult(fmt.Sprintf("source collision: %s", source))
				results = append(results, result)
				continue
			}
			if _, ok := priorities[repo.Priority()]; ok {
				logger.ErrorContext(
					ctx,
					"refresh repositories",
					slog_keys.RepositoryType, repoType,
					slog_keys.Source, source,
					slog_keys.Priority, repo.Priority(),
					slog_keys.Error, "priority collision",
				)
				result.ErrorResult = lib_models.NewErrorResult(fmt.Sprintf("priority collision: %d", repo.Priority()))
				results = append(results, result)
				continue
			}
			err = repo.Refresh(ctx)
			if err != nil {
				logger.ErrorContext(
					ctx,
					"refresh repositories",
					slog_keys.RepositoryType, repoType,
					slog_keys.Source, source,
					slog_keys.Error, err.Error(),
				)
				result.ErrorResult = lib_models.NewErrorResult(err.Error())
				results = append(results, result)
				continue
			}
			repositories[source] = repo
			priorities[repo.Priority()] = struct{}{}
			results = append(results, result)
		}
	}
	lookupErrResults := h.updateLookupMap(ctx, repositories)
	for i, result := range results {
		chanErrs, ok := lookupErrResults[result.Source]
		if ok {
			result.ChannelErrors = chanErrs
			result.ErrorResult = lib_models.NewErrorResult(fmt.Sprintf("%d channel errors", len(chanErrs)))
			results[i] = result
		}
	}
	return results, nil
}

func (h *Handler) GetRepositories(ctx context.Context) ([]lib_models.Repository, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	var repos []lib_models.Repository
	for repoType, handler := range h.repositoryHandlers {
		repositories, err := handler.GetRepositories(ctx)
		if err != nil {
			logger.ErrorContext(ctx, "get repositories", slog_keys.RepositoryType, repoType, slog_keys.Error, err.Error())
			continue
		}
		for source, repo := range repositories {
			repos = append(repos, lib_models.Repository{
				Type:     repoType,
				Source:   source,
				Priority: repo.Priority(),
				Channels: repo.Channels(),
			})
		}
	}
	return repos, nil
}

func (h *Handler) CreateRepository(ctx context.Context, repositoryType string, data []byte) error {
	h.mu.RLock()
	defer h.mu.RUnlock()
	handler, ok := h.repositoryHandlers[repositoryType]
	if !ok {
		err := lib_errors.New[lib_errors.ErrNotFound]("repository handler not found")
		logger.ErrorContext(ctx, "create repository", slog_keys.RepositoryType, repositoryType, slog_keys.Error, err.Error())
		return err
	}
	err := handler.CreateRepository(ctx, data)
	if err != nil {
		logger.ErrorContext(ctx, "create repository", slog_keys.RepositoryType, repositoryType, slog_keys.Error, err.Error())
		return err
	}
	return nil
}

func (h *Handler) DeleteRepository(ctx context.Context, source string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	for repoType, handler := range h.repositoryHandlers {
		err := handler.DeleteRepository(ctx, source)
		if err != nil {
			logger.ErrorContext(
				ctx,
				"delete repository",
				slog_keys.RepositoryType, repoType,
				slog_keys.Source, source,
				slog_keys.Error, err.Error(),
			)
			return err
		}
	}
	return nil
}

func (h *Handler) GetModules(_ context.Context, filter pkg_models.RepositoryModulesFilter) ([]pkg_models.RepositoryModule, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	filterById := len(filter.Ids) > 0
	filterBySource := len(filter.Sources) > 0
	filter.Name = strings.ToLower(filter.Name)
	sourceFilterMap := newSourceFilterMap(filter.Sources)
	var variants []pkg_models.RepositoryModule
	for modId, sources := range h.lookUpMap {
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

func (h *Handler) GetModule(ctx context.Context, id, source, channel string) (pkg_models.RepositoryModule, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	variant, err := h.getModuleVariant(id, source, channel)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"get module",
			slog_keys.Source, source,
			slog_keys.Channel, channel,
			slog_keys.ModuleId, id,
			slog_keys.Error, err,
		)
		return pkg_models.RepositoryModule{}, lib_errors.Wrap[lib_errors.ErrNotFound](err)
	}
	return variant.RepositoryModule, nil
}

func (h *Handler) GetModuleFS(ctx context.Context, id, source, channel string) (fs.FS, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	variant, err := h.getModuleVariant(id, source, channel)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"get module file system",
			slog_keys.Source, source,
			slog_keys.Channel, channel,
			slog_keys.ModuleId, id,
			slog_keys.Error, err,
		)
		return nil, lib_errors.Wrap[lib_errors.ErrNotFound](err)
	}
	handler, ok := h.repositoryHandlers[variant.RepoType]
	if !ok {
		err = errors.New("repository handler not found")
		logger.ErrorContext(
			ctx,
			"get module file system",
			slog_keys.RepositoryType, variant.RepoType,
			slog_keys.Source, source,
			slog_keys.Channel, channel,
			slog_keys.ModuleId, id,
			slog_keys.Error, err,
		)
		return nil, err
	}
	repo, err := handler.GetRepository(ctx, variant.Source)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"get module file system",
			slog_keys.RepositoryType, variant.RepoType,
			slog_keys.Source, source,
			slog_keys.Channel, channel,
			slog_keys.ModuleId, id,
			slog_keys.Error, err,
		)
		return nil, err
	}
	fSys, err := repo.GetFileSystem(ctx, variant.Channel, variant.FSysRef)
	if err != nil {
		logger.ErrorContext(
			ctx,
			"get module file system",
			slog_keys.RepositoryType, variant.RepoType,
			slog_keys.Source, source,
			slog_keys.Channel, channel,
			slog_keys.ModuleId, id,
			slog_keys.Error, err,
		)
		return nil, err
	}
	return fSys, nil
}

func (h *Handler) updateLookupMap(ctx context.Context, repositories map[string]Repository) map[string][]lib_models.RepositoryChannelErrorResult {
	lookupMap := make(map[string]map[string]map[string]moduleWrapper)
	errResults := make(map[string][]lib_models.RepositoryChannelErrorResult)
	for source, repo := range repositories {
		for _, channel := range repo.Channels() {
			fsMap, err := repo.GetFileSystemsMap(ctx, channel.Name)
			if err != nil {
				logger.ErrorContext(
					ctx,
					"update module variants, get file systems",
					slog_keys.RepositoryType, repo.Type(),
					slog_keys.Source, source,
					slog_keys.Channel, channel.Name,
					slog_keys.Error, err.Error(),
				)

				errResults[source] = append(errResults[source], lib_models.RepositoryChannelErrorResult{
					Channel:     channel.Name,
					ErrorResult: lib_models.NewErrorResult(err.Error()),
				})
				continue
			}
			for ref, fSys := range fsMap {
				mod, err := helper_modfile.GetModule(fSys)
				if err != nil {
					logger.ErrorContext(
						ctx,
						"update module variants, get module",
						slog_keys.RepositoryType, repo.Type(),
						slog_keys.Source, source,
						slog_keys.Channel, channel.Name,
						slog_keys.Reference, ref,
						slog_keys.Error, err.Error(),
					)
					errResults[source] = append(errResults[source], lib_models.RepositoryChannelErrorResult{
						Channel:     channel.Name,
						ErrorResult: lib_models.NewErrorResult(err.Error()),
					})
					continue
				}
				sources, ok := lookupMap[mod.ID]
				if !ok {
					sources = make(map[string]map[string]moduleWrapper)
					lookupMap[mod.ID] = sources
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
					RepoType: repo.Type(),
					FSysRef:  ref,
				}
				logger.DebugContext(
					ctx,
					"update module variants, add",
					slog_keys.RepositoryType, repo.Type(),
					slog_keys.Source, source,
					slog_keys.Channel, channel.Name,
					slog_keys.Reference, ref,
					slog_keys.ModuleId, mod.ID,
					slog_keys.Version, mod.Version,
				)
			}
		}
	}
	h.lookUpMap = lookupMap
	return errResults
}

func (h *Handler) getModuleVariant(id, source, channel string) (moduleWrapper, error) {
	sources, ok := h.lookUpMap[id]
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
