package modules

import (
	"context"
	"errors"
	"io/fs"
	"maps"
	"os"
	"path"
	"slices"
	"strings"
	"sync"

	lib_errors "github.com/SENERGY-Platform/mgw-module-manager/lib/errors"
	helper_errors "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/errors"
	helper_file_sys "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/file_sys"
	helper_job "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/job"
	helper_modfile "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/modfile"
	helper_time "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/time"
	helper_url "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/url"
	helper_uuid "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/uuid"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants/slog_keys"
	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

type Handler struct {
	databaseHandler              databaseHandler
	containerEngineWrapperClient containerEngineWrapperClient
	config                       Config
	cache                        map[string]external_models.ModuleLibModule
	cacheMU                      sync.RWMutex
	mu                           sync.RWMutex
}

func New(databaseHandler databaseHandler, containerEngineWrapperClient containerEngineWrapperClient, config Config) *Handler {
	return &Handler{
		databaseHandler:              databaseHandler,
		containerEngineWrapperClient: containerEngineWrapperClient,
		cache:                        make(map[string]external_models.ModuleLibModule),
		config:                       config,
	}
}

func (h *Handler) Init() error {
	return os.MkdirAll(h.config.WorkDirPath, 0775)
}

func (h *Handler) GetModules(
	ctx context.Context,
	filter pkg_models.ModulesFilterWithName,
	dependencies bool,
) (map[string]pkg_models.Module, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if dependencies && len(filter.Ids) > 0 {
		modulesWithDependencies := make(map[string]pkg_models.Module)
		err := h.getModulesWithDependencies(ctx, filter.Ids, modulesWithDependencies)
		if err != nil {
			return nil, err
		}
		return modulesWithDependencies, nil
	}
	return h.getModules(ctx, filter)
}

func (h *Handler) GetModule(ctx context.Context, id string) (pkg_models.Module, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	modules, err := h.getModules(
		ctx,
		pkg_models.ModulesFilterWithName{
			ModulesFilter: pkg_models.ModulesFilter{
				Ids: []string{id},
			},
		},
	)
	if err != nil {
		return pkg_models.Module{}, err
	}
	if len(modules) == 0 {
		return pkg_models.Module{}, lib_errors.New[lib_errors.ErrNotFound]("module not found")
	}
	return modules[id], nil
}

func (h *Handler) AddModule(ctx context.Context, id, source, channel string, fSys fs.FS) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.databaseHandler.Module(ctx, id)
	if err != nil {
		if !lib_errors.IsOf[lib_errors.ErrNotFound](err) {
			return err
		}
	} else {
		return lib_errors.New[lib_errors.ErrExists]("module already exists")
	}
	stgMod, err := newStgMod(id, source, channel)
	if err != nil {
		return err
	}
	timestamp := helper_time.Now()
	stgMod.Added = timestamp
	stgMod.Updated = timestamp
	dstPath := path.Join(h.config.WorkDirPath, stgMod.DirName)
	if err = helper_file_sys.CopyAll(fSys, dstPath); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if e := os.RemoveAll(dstPath); e != nil {
				logger.Error("removing dir failed", slog_keys.DirName, stgMod.DirName, slog_keys.Id, id, slog_keys.Error, e)
			}
		}
	}()
	mod, err := helper_modfile.GetModule(os.DirFS(dstPath))
	if err != nil {
		return err
	}
	if id != mod.ID {
		err = errors.New("id mismatch")
		return err
	}
	newImages, err := h.pullImages(ctx, getModuleServiceImages(mod.Services))
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if e := h.removeImages(ctx, newImages); e != nil {
				logger.Error("removing images failed", slog_keys.Id, id, slog_keys.Error, e)
			}
		}
	}()
	if err = h.databaseHandler.CreateModule(ctx, stgMod); err != nil {
		return err
	}
	h.cacheSet(id, mod)
	return nil
}

func (h *Handler) UpdateModule(ctx context.Context, id, source, channel string, fSys fs.FS) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	stgModOld, err := h.databaseHandler.Module(ctx, id)
	if err != nil {
		return err
	}
	oldMod, ok := h.cacheGet(id)
	if !ok {
		modFS := os.DirFS(path.Join(h.config.WorkDirPath, stgModOld.DirName))
		oldMod, err = helper_modfile.GetModule(modFS)
		if err != nil {
			return err
		}
	}
	stgModNew, err := newStgMod(id, source, channel)
	if err != nil {
		return err
	}
	stgModNew.Added = stgModOld.Added
	stgModNew.Updated = helper_time.Now()
	dstPath := path.Join(h.config.WorkDirPath, stgModNew.DirName)
	if err = helper_file_sys.CopyAll(fSys, dstPath); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if e := os.RemoveAll(dstPath); e != nil {
				logger.Error("removing dir failed", slog_keys.DirName, stgModNew.DirName, slog_keys.Id, id, slog_keys.Error, e)
			}
		}
	}()
	newMod, err := helper_modfile.GetModule(os.DirFS(dstPath))
	if err != nil {
		return err
	}
	if id != newMod.ID {
		err = errors.New("id mismatch")
		return err
	}
	newImages, err := h.pullImages(ctx, getModuleServiceImages(newMod.Services))
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if e := h.removeImages(ctx, newImages); e != nil {
				logger.Error("removing new images failed", slog_keys.Id, id, slog_keys.Error, e)
			}
		}
	}()
	if err = h.databaseHandler.UpdateModule(ctx, stgModNew); err != nil {
		return err
	}
	h.cacheSet(id, newMod)
	if e := os.RemoveAll(path.Join(h.config.WorkDirPath, stgModOld.DirName)); e != nil {
		logger.Error("removing dir failed", slog_keys.DirName, stgModOld.DirName, slog_keys.Id, id, slog_keys.Error, e)
	}
	if e := h.removeOldImages(ctx, getModuleServiceImages(oldMod.Services), getModuleServiceImages(newMod.Services)); e != nil {
		logger.Error("removing images failed", slog_keys.Id, id, slog_keys.Error, e)
	}
	return nil
}

func (h *Handler) RemoveModule(ctx context.Context, id string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	stgMod, err := h.databaseHandler.Module(ctx, id)
	if err != nil {
		return err
	}
	mod, ok := h.cacheGet(id)
	if !ok {
		modFS := os.DirFS(path.Join(h.config.WorkDirPath, stgMod.DirName))
		mod, err = helper_modfile.GetModule(modFS)
		if err != nil {
			return err
		}
	}
	err = os.RemoveAll(path.Join(h.config.WorkDirPath, stgMod.DirName))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err = h.removeImages(ctx, getModuleServiceImages(mod.Services)); err != nil {
		return err
	}
	if err = h.databaseHandler.DeleteModule(ctx, id); err != nil {
		return err
	}
	h.cacheDel(id)
	return nil
}

func (h *Handler) getModulesWithDependencies(ctx context.Context, filterIds []string, modulesWithDependencies map[string]pkg_models.Module) error {
	modules, err := h.getModules(
		ctx,
		pkg_models.ModulesFilterWithName{
			ModulesFilter: pkg_models.ModulesFilter{
				Ids: filterIds,
			},
		},
	)
	if err != nil {
		return err
	}
	for id, module := range modules {
		_, ok := modulesWithDependencies[id]
		if !ok {
			modulesWithDependencies[id] = module
		}
		if len(module.Dependencies) > 0 {
			var dependencyIds []string
			for dependencyId := range module.Dependencies {
				_, ok := modulesWithDependencies[dependencyId]
				if !ok {
					dependencyIds = append(dependencyIds, dependencyId)
				}
			}
			err = h.getModulesWithDependencies(ctx, dependencyIds, modulesWithDependencies)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *Handler) getModules(ctx context.Context, filter pkg_models.ModulesFilterWithName) (map[string]pkg_models.Module, error) {
	stgMods, err := h.databaseHandler.Modules(ctx, pkg_models.ModulesFilter{
		Ids:     filter.Ids,
		Source:  filter.Source,
		Channel: filter.Channel,
	})
	if err != nil {
		return nil, err
	}
	filter.Name = strings.ToLower(filter.Name)
	modules := make(map[string]pkg_models.Module)
	var errs []string
	for _, stgMod := range stgMods {
		modFS := os.DirFS(path.Join(h.config.WorkDirPath, stgMod.DirName))
		mod, ok := h.cacheGet(stgMod.Id)
		if !ok {
			mod, err = helper_modfile.GetModule(modFS)
			if err != nil {
				errs = append(errs, err.Error())
				logger.Error("getting module failed", slog_keys.Id, stgMod.Id, slog_keys.Error, err)
				continue
			}
			h.cacheSet(stgMod.Id, mod)
		}
		if !strings.Contains(strings.ToLower(mod.Name), filter.Name) { // empty string = true
			continue
		}
		modules[mod.ID] = pkg_models.Module{
			ModuleLibModule: mod,
			Source:          stgMod.Source,
			Channel:         stgMod.Channel,
			Added:           stgMod.Added,
			Updated:         stgMod.Updated,
			FileSystem:      modFS,
		}
	}
	lenErrs := len(errs)
	if lenErrs > 0 && lenErrs == len(stgMods) {
		return nil, errors.New(strings.Join(errs, "\n"))
	}
	return modules, nil
}

func (h *Handler) cacheGet(id string) (external_models.ModuleLibModule, bool) {
	h.cacheMU.RLock()
	defer h.cacheMU.RUnlock()
	mod, ok := h.cache[id]
	return mod, ok
}

func (h *Handler) cacheSet(id string, mod external_models.ModuleLibModule) {
	h.cacheMU.Lock()
	defer h.cacheMU.Unlock()
	h.cache[id] = mod
}

func (h *Handler) cacheDel(id string) {
	h.cacheMU.Lock()
	defer h.cacheMU.Unlock()
	delete(h.cache, id)
}

func (h *Handler) pullImages(ctx context.Context, images []string) ([]string, error) {
	var newImages []string
	for _, image := range images {
		_, err := h.containerEngineWrapperClient.GetImage(ctx, helper_url.EscapePath(image, h.config.PathEscapeDepth))
		if err != nil {
			var notFoundErr *external_models.CewNotFoundErr
			if !errors.As(err, &notFoundErr) {
				return newImages, err
			}
		} else {
			continue
		}
		err = h.pullImage(ctx, image)
		if err != nil {
			return newImages, err
		}
		newImages = append(newImages, image)
	}
	return newImages, nil
}

func (h *Handler) pullImage(ctx context.Context, image string) error {
	jobId, err := h.containerEngineWrapperClient.AddImage(ctx, image)
	if err != nil {
		return err
	}
	job, err := helper_job.Await(ctx, h.containerEngineWrapperClient, jobId, h.config.JobPollInterval)
	if err != nil {
		return err
	}
	if job.Error != nil {
		return errors.New(job.Error.Message)
	}
	return nil
}

func (h *Handler) removeImages(ctx context.Context, images []string) error {
	for _, image := range images {
		err := h.containerEngineWrapperClient.RemoveImage(ctx, helper_url.EscapePath(image, h.config.PathEscapeDepth))
		if err != nil {
			var notFoundErr *external_models.CewNotFoundErr
			if !errors.As(err, &notFoundErr) {
				return err
			}
		}
	}
	return nil
}

func (h *Handler) removeOldImages(ctx context.Context, oldImages, newImages []string) error {
	for _, image := range oldImages {
		if !slices.Contains(newImages, image) {
			err := h.containerEngineWrapperClient.RemoveImage(ctx, helper_url.EscapePath(image, h.config.PathEscapeDepth))
			if err != nil {
				var notFoundErr *external_models.CewNotFoundErr
				if !errors.As(err, &notFoundErr) {
					return err
				}
			}
		}
	}
	return nil
}

func getModuleServiceImages(services map[string]external_models.ModuleLibService) []string {
	images := make(map[string]struct{})
	for _, service := range services {
		images[service.Image] = struct{}{}
	}
	return slices.Collect(maps.Keys(images))
}

func newStgMod(id, source, channel string) (pkg_models.DatabaseModule, error) {
	dirName, err := helper_uuid.New()
	if err != nil {
		return pkg_models.DatabaseModule{}, err
	}
	return pkg_models.DatabaseModule{
		Id:      id,
		DirName: dirName,
		Source:  source,
		Channel: channel,
	}, nil
}
