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
	modules     map[string]module_lib.Module
	workDirPath string
	dbTimeout   time.Duration
	mu          sync.RWMutex
}

func New(storageHdl StorageHandler, workDirPath string, dbTimeout time.Duration) *Handler {
	return &Handler{
		storageHdl:  storageHdl,
		modules:     make(map[string]module_lib.Module),
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
	dirEntries, err := os.ReadDir(h.workDirPath)
	if err != nil {
		return err
	}
	for _, entry := range dirEntries {
		modFS := os.DirFS(path.Join(h.workDirPath, entry.Name()))
		mod, err := modfile_util.GetModule(modFS)
		if err != nil {
			fmt.Println(err)
			continue
		}
		h.modules[mod.ID] = mod
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
		mod, ok := h.modules[storedMod.ID]
		if !ok {
			fmt.Println("WARNING: module not in cache")
			modFS := os.DirFS(path.Join(h.workDirPath, storedMod.DirName))
			mod, err = modfile_util.GetModule(modFS)
			if err != nil {
				errs = append(errs, err)
				fmt.Println(err)
				continue
			}
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
	mod, ok := h.modules[id]
	if !ok {
		fmt.Println("WARNING: module not in cache")
		modFS := os.DirFS(path.Join(h.workDirPath, storedMod.DirName))
		mod, err = modfile_util.GetModule(modFS)
		if err != nil {
			return models_module.Module{}, err
		}
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
	if err = h.storageHdl.CreateMod(ctxWt2, nil, mod); err != nil {
		return err
	}
	h.modules[id] = tmp
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
	if err = h.storageHdl.UpdateMod(ctxWt2, nil, newMod); err != nil {
		return err
	}
	h.modules[id] = tmp
	if e := os.RemoveAll(path.Join(h.workDirPath, oldMod.DirName)); e != nil {
		fmt.Println(e)
	}
	return nil
}

func (h *Handler) Delete(ctx context.Context, id string) error {
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
	if err = h.storageHdl.DeleteMod(ctxWt2, nil, id); err != nil {
		return err
	}
	delete(h.modules, id)
	return nil
}
