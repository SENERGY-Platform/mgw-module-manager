package mod_hdl

import (
	"context"
	"errors"
	"fmt"
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	module_lib "github.com/SENERGY-Platform/mgw-module-lib/model"
	helper_file_sys "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/file_sys"
	helper_job "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/job"
	helper_modfile "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/modfile"
	models_error "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
	models_module "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/module"
	models_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/storage"
	"github.com/google/uuid"
	"io/fs"
	"net/url"
	"os"
	"path"
	"sync"
)

type Handler struct {
	storageHdl StorageHandler
	cewClient  ContainerEngineWrapperClient
	logger     Logger
	config     Config
	cache      map[string]module_lib.Module
	cacheMU    sync.RWMutex
	mu         sync.RWMutex
}

func New(storageHdl StorageHandler, cewClient ContainerEngineWrapperClient, logger Logger, config Config) *Handler {
	return &Handler{
		storageHdl: storageHdl,
		cewClient:  cewClient,
		logger:     logger,
		cache:      make(map[string]module_lib.Module),
		config:     config,
	}
}

func (h *Handler) Init() error {
	return os.MkdirAll(h.config.WorkDirPath, 0775)
}

func (h *Handler) Modules(ctx context.Context, filter models_module.ModuleFilter) ([]models_module.ModuleAbbreviated, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	stgMods, err := h.storageHdl.ListMod(ctx, models_storage.ModuleFilter{IDs: filter.IDs})
	if err != nil {
		return nil, err
	}
	var modules []models_module.ModuleAbbreviated
	var errs []error
	for _, stgMod := range stgMods {
		mod, ok := h.cacheGet(stgMod.ID)
		if !ok {
			h.logger.Warningf("module '%s' not in cache", stgMod.ID)
			modFS := os.DirFS(path.Join(h.config.WorkDirPath, stgMod.DirName))
			mod, err = helper_modfile.GetModule(modFS)
			if err != nil {
				errs = append(errs, err)
				h.logger.Errorf("getting module '%s' failed: %s", stgMod.ID, err)
				continue
			}
			h.cacheSet(stgMod.ID, mod)
		}
		modules = append(modules, models_module.ModuleAbbreviated{
			ID:      stgMod.ID,
			Name:    mod.Name,
			Desc:    mod.Description,
			Version: mod.Version,
			ModuleBase: models_module.ModuleBase{
				Source:  stgMod.Source,
				Channel: stgMod.Channel,
				Added:   stgMod.Added,
				Updated: stgMod.Updated,
			},
		})
	}
	if len(errs) == len(stgMods) {
		return nil, models_error.NewMultiError(errs)
	}
	return modules, nil
}

func (h *Handler) Module(ctx context.Context, id string) (models_module.Module, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	stgMod, err := h.storageHdl.ReadMod(ctx, id)
	if err != nil {
		return models_module.Module{}, err
	}
	mod, ok := h.cacheGet(id)
	if !ok {
		h.logger.Warningf("module '%s' not in cache", stgMod.ID)
		modFS := os.DirFS(path.Join(h.config.WorkDirPath, stgMod.DirName))
		mod, err = helper_modfile.GetModule(modFS)
		if err != nil {
			return models_module.Module{}, err
		}
		h.cacheSet(id, mod)
	}
	return models_module.Module{
		Module: mod,
		ModuleBase: models_module.ModuleBase{
			Source:  stgMod.Source,
			Channel: stgMod.Channel,
			Added:   stgMod.Added,
			Updated: stgMod.Updated,
		},
	}, nil
}

func (h *Handler) ModuleFS(ctx context.Context, id string) (fs.FS, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	storedMod, err := h.storageHdl.ReadMod(ctx, id)
	if err != nil {
		return nil, err
	}
	return os.DirFS(path.Join(h.config.WorkDirPath, storedMod.DirName)), nil
}

func (h *Handler) Add(ctx context.Context, id, source, channel string, fSys fs.FS) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	stgMod, err := h.storageHdl.ReadMod(ctx, id)
	if err != nil {
		var notFoundErr *models_error.NotFoundError
		if !errors.As(err, &notFoundErr) {
			return err
		}
	}
	if stgMod.ID != "" {
		return errors.New("already exists")
	}
	stgModBase, err := newStgModBase(id, source, channel)
	if err != nil {
		return err
	}
	dstPath := path.Join(h.config.WorkDirPath, stgModBase.DirName)
	if err = helper_file_sys.CopyAll(fSys, dstPath); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if e := os.RemoveAll(dstPath); e != nil {
				h.logger.Errorf("removing dir '%s' of '%s' failed: %s", stgModBase.DirName, id, e)
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
	newImages, err := h.addImages(ctx, imagesAsSet(mod.Services))
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if e := h.removeImages(ctx, newImages); e != nil {
				h.logger.Errorf("removing images of '%s' failed: %s", id, e)
			}
		}
	}()
	if err = h.storageHdl.CreateMod(ctx, stgModBase); err != nil {
		return err
	}
	h.cacheSet(id, mod)
	return nil
}

func (h *Handler) Update(ctx context.Context, id, source, channel string, fSys fs.FS) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	stgMod, err := h.storageHdl.ReadMod(ctx, id)
	if err != nil {
		return err
	}
	oldMod, ok := h.cacheGet(id)
	if !ok {
		h.logger.Warningf("module '%s' not in cache", stgMod.ID)
		modFS := os.DirFS(path.Join(h.config.WorkDirPath, stgMod.DirName))
		oldMod, err = helper_modfile.GetModule(modFS)
		if err != nil {
			return err
		}
	}
	stgModBase, err := newStgModBase(id, source, channel)
	if err != nil {
		return err
	}
	dstPath := path.Join(h.config.WorkDirPath, stgModBase.DirName)
	if err = helper_file_sys.CopyAll(fSys, dstPath); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if e := os.RemoveAll(dstPath); e != nil {
				h.logger.Errorf("removing dir '%s' of '%s' failed: %s", stgModBase.DirName, id, e)
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
	newImages, err := h.addImages(ctx, imagesAsSet(newMod.Services))
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if e := h.removeImages(ctx, newImages); e != nil {
				h.logger.Errorf("removing new images of '%s' failed: %s", id, e)
			}
		}
	}()
	if err = h.storageHdl.UpdateMod(ctx, stgModBase); err != nil {
		return err
	}
	h.cacheSet(id, newMod)
	if e := os.RemoveAll(path.Join(h.config.WorkDirPath, stgMod.DirName)); e != nil {
		h.logger.Errorf("removing dir '%s' of '%s' failed: %s", stgMod.DirName, id, e)
	}
	if e := h.removeOldImages(ctx, imagesAsSet(oldMod.Services), imagesAsSet(newMod.Services)); e != nil {
		h.logger.Errorf("removing images of '%s' failed: %s", id, e)
	}
	return nil
}

func (h *Handler) Remove(ctx context.Context, id string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	stgMod, err := h.storageHdl.ReadMod(ctx, id)
	if err != nil {
		return err
	}
	mod, ok := h.cacheGet(id)
	if !ok {
		h.logger.Warningf("module '%s' not in cache", stgMod.ID)
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
	if err = h.storageHdl.DeleteMod(ctx, id); err != nil {
		return err
	}
	h.cacheDel(id)
	return nil
}

func (h *Handler) cacheGet(id string) (module_lib.Module, bool) {
	h.cacheMU.RLock()
	defer h.cacheMU.RUnlock()
	mod, ok := h.cache[id]
	return mod, ok
}

func (h *Handler) cacheSet(id string, mod module_lib.Module) {
	h.cacheMU.Lock()
	defer h.cacheMU.Unlock()
	h.cache[id] = mod
}

func (h *Handler) cacheDel(id string) {
	h.cacheMU.Lock()
	defer h.cacheMU.Unlock()
	delete(h.cache, id)
}

func (h *Handler) addImages(ctx context.Context, images map[string]struct{}) (map[string]struct{}, error) {
	newImages := make(map[string]struct{})
	for image := range images {
		_, err := h.cewClient.GetImage(ctx, url.QueryEscape(url.QueryEscape(image)))
		if err != nil {
			var notFoundErr *cew_model.NotFoundError
			if !errors.As(err, &notFoundErr) {
				return newImages, err
			}
		} else {
			continue
		}
		jID, err := h.cewClient.AddImage(ctx, image)
		if err != nil {
			return newImages, err
		}
		job, err := helper_job.Await(ctx, h.cewClient, jID, h.config.JobPollInterval)
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
		err := h.cewClient.RemoveImage(ctx, url.QueryEscape(url.QueryEscape(image)))
		if err != nil {
			var notFoundErr *cew_model.NotFoundError
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
			err := h.cewClient.RemoveImage(ctx, url.QueryEscape(url.QueryEscape(image)))
			if err != nil {
				var notFoundErr *cew_model.NotFoundError
				if !errors.As(err, &notFoundErr) {
					return err
				}
			}
		}
	}
	return nil
}

func imagesAsSet(services map[string]*module_lib.Service) map[string]struct{} {
	images := make(map[string]struct{})
	for _, service := range services {
		images[service.Image] = struct{}{}
	}
	return images
}

func newStgModBase(id, source, channel string) (models_storage.ModuleBase, error) {
	newUUID, err := uuid.NewUUID()
	if err != nil {
		return models_storage.ModuleBase{}, err
	}
	return models_storage.ModuleBase{
		ID:      id,
		DirName: newUUID.String(),
		Source:  source,
		Channel: channel,
	}, nil
}
