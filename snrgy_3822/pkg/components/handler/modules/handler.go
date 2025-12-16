package modules

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"
	"sync"

	helper_file_sys "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/file_sys"
	helper_job "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/job"
	helper_modfile "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/modfile"
	helper_time "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/time"
	helper_url "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/url"
	models_error "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	models_handler_module "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/module"
	models_handler_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/storage"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/slog_attr"
	"github.com/google/uuid"
)

type Handler struct {
	storageHdl storageHandler
	cewClient  containerEngineWrapperClient
	config     Config
	cache      map[string]models_external.Module
	cacheMU    sync.RWMutex
	mu         sync.RWMutex
}

func New(storageHdl storageHandler, cewClient containerEngineWrapperClient, config Config) *Handler {
	return &Handler{
		storageHdl: storageHdl,
		cewClient:  cewClient,
		cache:      make(map[string]models_external.Module),
		config:     config,
	}
}

func (h *Handler) Init() error {
	return os.MkdirAll(h.config.WorkDirPath, 0775)
}

func (h *Handler) Modules(ctx context.Context, filter models_handler_module.ModuleFilter) (map[string]models_handler_module.Module, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if filter.Dependencies {
		requiredModules := make(map[string]models_handler_module.Module)
		err := h.modulesWithDependencies(ctx, filter.Ids, requiredModules)
		if err != nil {
			return nil, err
		}
		return requiredModules, nil
	}
	return h.modules(ctx, filter)
}

func (h *Handler) Module(ctx context.Context, id string) (models_handler_module.Module, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	modules, err := h.modules(ctx, models_handler_module.ModuleFilter{Ids: []string{id}})
	if err != nil {
		return models_handler_module.Module{}, err
	}
	if len(modules) == 0 {
		return models_handler_module.Module{}, models_error.NotFoundErr
	}
	return modules[id], nil
}

func (h *Handler) Add(ctx context.Context, id, source, channel string, fSys fs.FS) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.storageHdl.Module(ctx, id)
	if err != nil {
		if !errors.Is(err, models_error.NotFoundErr) {
			return err
		}
	} else {
		return models_error.DuplicateErr
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
				logger.Error("removing dir failed", slog_attr.DirNameKey, stgMod.DirName, slog_attr.IdKey, id, slog_attr.ErrorKey, e)
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
	newImages, err := h.pullImages(ctx, imagesAsSet(mod.Services))
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if e := h.removeImages(ctx, newImages); e != nil {
				logger.Error("removing images failed", slog_attr.IdKey, id, slog_attr.ErrorKey, e)
			}
		}
	}()
	if err = h.storageHdl.CreateModule(ctx, stgMod); err != nil {
		return err
	}
	h.cacheSet(id, mod)
	return nil
}

func (h *Handler) Update(ctx context.Context, id, source, channel string, fSys fs.FS) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	stgModOld, err := h.storageHdl.Module(ctx, id)
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
				logger.Error("removing dir failed", slog_attr.DirNameKey, stgModNew.DirName, slog_attr.IdKey, id, slog_attr.ErrorKey, e)
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
	newImages, err := h.pullImages(ctx, imagesAsSet(newMod.Services))
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if e := h.removeImages(ctx, newImages); e != nil {
				logger.Error("removing new images failed", slog_attr.IdKey, id, slog_attr.ErrorKey, e)
			}
		}
	}()
	if err = h.storageHdl.UpdateModule(ctx, stgModNew); err != nil {
		return err
	}
	h.cacheSet(id, newMod)
	if e := os.RemoveAll(path.Join(h.config.WorkDirPath, stgModOld.DirName)); e != nil {
		logger.Error("removing dir failed", slog_attr.DirNameKey, stgModOld.DirName, slog_attr.IdKey, id, slog_attr.ErrorKey, e)
	}
	if e := h.removeOldImages(ctx, imagesAsSet(oldMod.Services), imagesAsSet(newMod.Services)); e != nil {
		logger.Error("removing images failed", slog_attr.IdKey, id, slog_attr.ErrorKey, e)
	}
	return nil
}

func (h *Handler) Remove(ctx context.Context, id string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	stgMod, err := h.storageHdl.Module(ctx, id)
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
	if err = h.removeImages(ctx, imagesAsSet(mod.Services)); err != nil {
		return err
	}
	if err = h.storageHdl.DeleteModule(ctx, id); err != nil {
		return err
	}
	h.cacheDel(id)
	return nil
}

func (h *Handler) modulesWithDependencies(ctx context.Context, ids []string, requiredModules map[string]models_handler_module.Module) error {
	modules, err := h.modules(ctx, models_handler_module.ModuleFilter{Ids: ids})
	if err != nil {
		return err
	}
	for _, id := range ids {
		module, ok := modules[id]
		if !ok {
			return fmt.Errorf("module %s not found", id) // TODO
		}
		if _, ok := requiredModules[id]; !ok {
			requiredModules[id] = module
		}
		if len(module.Dependencies) > 0 {
			var dependencyIds []string
			for dependencyId := range module.Dependencies {
				if _, ok := requiredModules[dependencyId]; !ok {
					dependencyIds = append(dependencyIds, dependencyId)
				}
			}
			err = h.modulesWithDependencies(ctx, dependencyIds, requiredModules)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *Handler) modules(ctx context.Context, filter models_handler_module.ModuleFilter) (map[string]models_handler_module.Module, error) {
	stgMods, err := h.storageHdl.Modules(ctx, models_handler_storage.ModulesFilter{
		Ids:     filter.Ids,
		Source:  filter.Source,
		Channel: filter.Channel,
	})
	if err != nil {
		return nil, err
	}
	filter.Name = strings.ToLower(filter.Name)
	modules := make(map[string]models_handler_module.Module)
	var errs []error
	for _, stgMod := range stgMods {
		modFS := os.DirFS(path.Join(h.config.WorkDirPath, stgMod.DirName))
		mod, ok := h.cacheGet(stgMod.Id)
		if !ok {
			mod, err = helper_modfile.GetModule(modFS)
			if err != nil {
				errs = append(errs, err)
				logger.Error("getting module failed", slog_attr.IdKey, stgMod.Id, slog_attr.ErrorKey, err)
				continue
			}
			h.cacheSet(stgMod.Id, mod)
		}
		if !strings.Contains(strings.ToLower(mod.Name), filter.Name) { // empty string = true
			continue
		}
		modules[mod.ID] = models_handler_module.Module{
			Module:     mod,
			Source:     stgMod.Source,
			Channel:    stgMod.Channel,
			Added:      stgMod.Added,
			Updated:    stgMod.Updated,
			FileSystem: modFS,
		}
	}
	lenErrs := len(errs)
	if lenErrs > 0 && lenErrs == len(stgMods) {
		return nil, models_error.NewMultiError(errs)
	}
	return modules, nil
}

func (h *Handler) cacheGet(id string) (models_external.Module, bool) {
	h.cacheMU.RLock()
	defer h.cacheMU.RUnlock()
	mod, ok := h.cache[id]
	return mod, ok
}

func (h *Handler) cacheSet(id string, mod models_external.Module) {
	h.cacheMU.Lock()
	defer h.cacheMU.Unlock()
	h.cache[id] = mod
}

func (h *Handler) cacheDel(id string) {
	h.cacheMU.Lock()
	defer h.cacheMU.Unlock()
	delete(h.cache, id)
}

func (h *Handler) pullImages(ctx context.Context, images map[string]struct{}) (map[string]struct{}, error) {
	newImages := make(map[string]struct{})
	for image := range images {
		_, err := h.cewClient.GetImage(ctx, helper_url.EscapePath(image, h.config.PathEscapeDepth))
		if err != nil {
			var notFoundErr *models_external.CEWNotFoundErr
			if !errors.As(err, &notFoundErr) {
				return newImages, err
			}
		} else {
			continue
		}
		jobId, err := h.cewClient.AddImage(ctx, image)
		if err != nil {
			return newImages, err
		}
		job, err := helper_job.Await(ctx, h.cewClient, jobId, h.config.JobPollInterval)
		if err != nil {
			return newImages, err
		}
		if job.Error != nil {
			return newImages, fmt.Errorf("%v", job.Error)
		}
		newImages[image] = struct{}{}
	}
	return newImages, nil
}

func (h *Handler) removeImages(ctx context.Context, images map[string]struct{}) error {
	for image := range images {
		err := h.cewClient.RemoveImage(ctx, helper_url.EscapePath(image, h.config.PathEscapeDepth))
		if err != nil {
			var notFoundErr *models_external.CEWNotFoundErr
			if !errors.As(err, &notFoundErr) {
				return err
			}
		}
	}
	return nil
}

func (h *Handler) removeOldImages(ctx context.Context, oldImages, newImages map[string]struct{}) error {
	for image := range oldImages {
		if _, ok := newImages[image]; !ok {
			err := h.cewClient.RemoveImage(ctx, helper_url.EscapePath(image, h.config.PathEscapeDepth))
			if err != nil {
				var notFoundErr *models_external.CEWNotFoundErr
				if !errors.As(err, &notFoundErr) {
					return err
				}
			}
		}
	}
	return nil
}

func imagesAsSet(services map[string]*models_external.ModuleService) map[string]struct{} {
	images := make(map[string]struct{})
	for _, service := range services {
		images[service.Image] = struct{}{}
	}
	return images
}

func newStgMod(id, source, channel string) (models_handler_storage.Module, error) {
	newUUID, err := uuid.NewUUID()
	if err != nil {
		return models_handler_storage.Module{}, err
	}
	return models_handler_storage.Module{
		Id:      id,
		DirName: newUUID.String(),
		Source:  source,
		Channel: channel,
	}, nil
}
