package mod_hdl

import (
	"context"
	"errors"
	"fmt"
	module_lib "github.com/SENERGY-Platform/mgw-module-lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/fs_util"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/modfile_util"
	models_error "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
	models_module "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/module"
	models_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/storage"
	"github.com/google/uuid"
	"io/fs"
	"os"
	"path"
	"sync"
	"time"
)

type Handler struct {
	storageHdl  StorageHandler
	cache       map[string]module_lib.Module
	workDirPath string
	dbTimeout   time.Duration
	cacheMU     sync.RWMutex
	mu          sync.RWMutex
}

func New(storageHdl StorageHandler, workDirPath string, dbTimeout time.Duration) *Handler {
	return &Handler{
		storageHdl:  storageHdl,
		cache:       make(map[string]module_lib.Module),
		workDirPath: workDirPath,
		dbTimeout:   dbTimeout,
	}
}

func (h *Handler) Init() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := os.MkdirAll(h.workDirPath, 0775); err != nil {
		return err
	}
	return nil
}

func (h *Handler) Modules(ctx context.Context, filter models_module.ModuleFilter) (map[string]models_module.ModuleAbbreviated, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	storedMods, err := h.storageHdl.ListMod(ctxWt, models_storage.ModuleFilter{IDs: filter.IDs})
	if err != nil {
		return nil, err
	}
	modulesMap := make(map[string]models_module.ModuleAbbreviated)
	var errs []error
	for _, storedMod := range storedMods {
		mod, ok := h.cacheGet(storedMod.ID)
		if !ok {
			fmt.Println("WARNING: module not in cache")
			modFS := os.DirFS(path.Join(h.workDirPath, storedMod.DirName))
			mod, err = modfile_util.GetModule(modFS)
			if err != nil {
				errs = append(errs, err)
				fmt.Println(err)
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
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	storedMod, err := h.storageHdl.ReadMod(ctxWt, id)
	if err != nil {
		return models_module.Module{}, err
	}
	mod, ok := h.cacheGet(id)
	if !ok {
		fmt.Println("WARNING: module not in cache")
		modFS := os.DirFS(path.Join(h.workDirPath, storedMod.DirName))
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
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	storedMod, err := h.storageHdl.ReadMod(ctxWt, id)
	if err != nil {
		return nil, err
	}
	return os.DirFS(path.Join(h.workDirPath, storedMod.DirName)), nil
}

func (h *Handler) Add(ctx context.Context, id, source, channel string, fSys fs.FS) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	if tmp, err := h.storageHdl.ReadMod(ctxWt, id); err != nil {
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
	dstPath := path.Join(h.workDirPath, mod.DirName)
	if err = fs_util.CopyAll(fSys, dstPath); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if e := os.RemoveAll(dstPath); e != nil {
				fmt.Println(e)
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
	ctxWt2, cf2 := context.WithTimeout(ctx, h.dbTimeout)
	defer cf2()
	if err = h.storageHdl.CreateMod(ctxWt2, mod); err != nil {
		return err
	}
	h.cacheSet(id, tmp)
	return nil
}

func (h *Handler) Update(ctx context.Context, id, source, channel string, fSys fs.FS) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	oldMod, err := h.storageHdl.ReadMod(ctxWt, id)
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
	dstPath := path.Join(h.workDirPath, newMod.DirName)
	if err = fs_util.CopyAll(fSys, dstPath); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			if e := os.RemoveAll(dstPath); e != nil {
				fmt.Println(e)
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
	ctxWt2, cf2 := context.WithTimeout(ctx, h.dbTimeout)
	defer cf2()
	if err = h.storageHdl.UpdateMod(ctxWt2, newMod); err != nil {
		return err
	}
	h.cacheSet(id, tmp)
	if e := os.RemoveAll(path.Join(h.workDirPath, oldMod.DirName)); e != nil {
		fmt.Println(e)
	}
	return nil
}

func (h *Handler) Remove(ctx context.Context, id string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	ctxWt, cf := context.WithTimeout(ctx, h.dbTimeout)
	defer cf()
	mod, err := h.storageHdl.ReadMod(ctxWt, id)
	if err != nil {
		return err
	}
	err = os.RemoveAll(path.Join(h.workDirPath, mod.DirName))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	ctxWt2, cf2 := context.WithTimeout(ctx, h.dbTimeout)
	defer cf2()
	if err = h.storageHdl.DeleteMod(ctxWt2, id); err != nil {
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
