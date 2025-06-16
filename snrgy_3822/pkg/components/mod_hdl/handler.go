package mod_hdl

import (
	"context"
	"errors"
	"fmt"
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	module_lib "github.com/SENERGY-Platform/mgw-module-lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/fs_util"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/job_util"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/modfile_util"
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

func (h *Handler) Modules(ctx context.Context, filter models_module.ModuleFilter) (map[string]models_module.ModuleAbbreviated, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	storedMods, err := h.storageHdl.ListMod(ctx, models_storage.ModuleFilter{IDs: filter.IDs})
	if err != nil {
		return nil, err
	}
	modulesMap := make(map[string]models_module.ModuleAbbreviated)
	var errs []error
	for _, storedMod := range storedMods {
		mod, ok := h.cacheGet(storedMod.ID)
		if !ok {
			h.logger.Warningf("module '%s' not in cache", storedMod.ID)
			modFS := os.DirFS(path.Join(h.config.WorkDirPath, storedMod.DirName))
			mod, err = modfile_util.GetModule(modFS)
			if err != nil {
				errs = append(errs, err)
				h.logger.Errorf("getting module '%s' failed: %s", storedMod.ID, err)
				continue
			}
			h.cacheSet(storedMod.ID, mod)
		}
		modulesMap[storedMod.ID] = models_module.ModuleAbbreviated{
			ID:      storedMod.ID,
			Name:    mod.Name,
			Desc:    mod.Description,
			Version: mod.Version,
			ModuleBase: models_module.ModuleBase{
				Source:  storedMod.Source,
				Channel: storedMod.Channel,
				Added:   storedMod.Added,
				Updated: storedMod.Updated,
			},
		}
	}
	if len(errs) == len(storedMods) {
		return nil, models_error.NewMultiError(errs)
	}
	return modulesMap, nil
}

func (h *Handler) Module(ctx context.Context, id string) (models_module.Module, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	storedMod, err := h.storageHdl.ReadMod(ctx, id)
	if err != nil {
		return models_module.Module{}, err
	}
	mod, ok := h.cacheGet(id)
	if !ok {
		h.logger.Warningf("module '%s' not in cache", storedMod.ID)
		modFS := os.DirFS(path.Join(h.config.WorkDirPath, storedMod.DirName))
		mod, err = modfile_util.GetModule(modFS)
		if err != nil {
			return models_module.Module{}, err
		}
		h.cacheSet(id, mod)
	}
	return models_module.Module{
		Module: mod,
		ModuleBase: models_module.ModuleBase{
			Source:  storedMod.Source,
			Channel: storedMod.Channel,
			Added:   storedMod.Added,
			Updated: storedMod.Updated,
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
	if tmp, err := h.storageHdl.ReadMod(ctx, id); err != nil {
		var notFoundErr *models_error.NotFoundError
		if !errors.As(err, &notFoundErr) {
			return err
		}
	} else {
		if tmp.ID != "" {
			return errors.New("already exists")
		}
	}
	newUUID, err := uuid.NewUUID()
	if err != nil {
		return err
	}
	mod := models_storage.ModuleBase{
		ID:      id,
		DirName: newUUID.String(),
		Source:  source,
		Channel: channel,
	}
	dstPath := path.Join(h.config.WorkDirPath, mod.DirName)
	if err = fs_util.CopyAll(fSys, dstPath); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if e := os.RemoveAll(dstPath); e != nil {
				h.logger.Errorf("removing new dir '%s' of '%s' failed: %s", dstPath, id, e)
			}
		}
	}()
	tmp, err := modfile_util.GetModule(os.DirFS(dstPath))
	if err != nil {
		return err
	}
	if id != tmp.ID {
		err = errors.New("id mismatch")
		return err
	}
	if err = h.addImages(ctx, tmp.Services); err != nil {
		return err
	}
	if err = h.storageHdl.CreateMod(ctx, mod); err != nil {
		return err
	}
	h.cacheSet(id, tmp)
	return nil
}

func (h *Handler) Update(ctx context.Context, id, source, channel string, fSys fs.FS) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	oldMod, err := h.storageHdl.ReadMod(ctx, id)
	if err != nil {
		return err
	}
	newUUID, err := uuid.NewUUID()
	if err != nil {
		return err
	}
	newMod := models_storage.ModuleBase{
		ID:      id,
		DirName: newUUID.String(),
		Source:  source,
		Channel: channel,
	}
	dstPath := path.Join(h.config.WorkDirPath, newMod.DirName)
	if err = fs_util.CopyAll(fSys, dstPath); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if e := os.RemoveAll(dstPath); e != nil {
				h.logger.Errorf("removing new dir '%s' of '%s' failed: %s", dstPath, id, e)
			}
		}
	}()
	tmp, err := modfile_util.GetModule(os.DirFS(dstPath))
	if err != nil {
		return err
	}
	if id != tmp.ID {
		err = errors.New("id mismatch")
		return err
	}
	if err = h.addImages(ctx, tmp.Services); err != nil {
		return err
	}
	if err = h.storageHdl.UpdateMod(ctx, newMod); err != nil {
		return err
	}
	h.cacheSet(id, tmp)
	if e := os.RemoveAll(path.Join(h.config.WorkDirPath, oldMod.DirName)); e != nil {
		h.logger.Errorf("removing old dir '%s' of '%s' failed: %s", path.Join(h.config.WorkDirPath, oldMod.DirName), id, e)
	}
	return nil
}

func (h *Handler) Remove(ctx context.Context, id string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	mod, err := h.storageHdl.ReadMod(ctx, id)
	if err != nil {
		return err
	}
	err = os.RemoveAll(path.Join(h.config.WorkDirPath, mod.DirName))
	if err != nil && !os.IsNotExist(err) {
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

func (h *Handler) addImages(ctx context.Context, services map[string]*module_lib.Service) error {
	images := make(map[string]struct{})
	for _, service := range services {
		images[service.Image] = struct{}{}
	}
	for image := range images {
		_, err := h.cewClient.GetImage(ctx, url.QueryEscape(url.QueryEscape(image)))
		if err != nil {
			var notFoundErr *cew_model.NotFoundError
			if !errors.As(err, &notFoundErr) {
				return err
			}
		} else {
			continue
		}
		jID, err := h.cewClient.AddImage(ctx, image)
		if err != nil {
			return err
		}
		job, err := job_util.Await(ctx, h.cewClient, jID, h.config.JobPollInterval)
		if err != nil {
			return err
		}
		if job.Error != nil {
			return fmt.Errorf("%v", job.Error)
		}
	}
	return nil
}

func (h *Handler) removeImages(ctx context.Context, services map[string]*module_lib.Service) error {
	images := make(map[string]struct{})
	for _, service := range services {
		images[service.Image] = struct{}{}
	}
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

func (h *Handler) removeOldImages(ctx context.Context, oldServices, newServices map[string]*module_lib.Service) error {
	newImages := make(map[string]struct{})
	for _, service := range newServices {
		newImages[service.Image] = struct{}{}
	}
	oldImages := make(map[string]struct{})
	for _, service := range oldServices {
		if _, ok := newImages[service.Image]; !ok {
			oldImages[service.Image] = struct{}{}
		}
	}
	for image := range oldImages {
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
