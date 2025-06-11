package mod_hdl

import (
	"context"
	"errors"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/fs_util"
	models_error "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
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
	workDirPath string
	dbTimeout   time.Duration
	mu          sync.RWMutex
}

func New(storageHdl StorageHandler, workDirPath string, dbTimeout time.Duration) *Handler {
	return &Handler{
		storageHdl:  storageHdl,
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

func (h *Handler) Add(ctx context.Context, id string, fSys fs.FS) error {
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
	}
	dstPath := path.Join(h.workDirPath, mod.DirName)
	if err = fs_util.CopyAll(fSys, dstPath); err != nil {
		return err
	}
	ctxWt2, cf2 := context.WithTimeout(ctx, h.dbTimeout)
	defer cf2()
	if err = h.storageHdl.CreateMod(ctxWt2, nil, mod); err != nil {
		os.RemoveAll(dstPath)
		return err
	}
	return nil
}

func (h *Handler) Update(ctx context.Context, id string, fSys fs.FS) error {
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
	}
	dstPath := path.Join(h.workDirPath, newMod.DirName)
	if err = fs_util.CopyAll(fSys, dstPath); err != nil {
		return err
	}
	ctxWt2, cf2 := context.WithTimeout(ctx, h.dbTimeout)
	defer cf2()
	if err = h.storageHdl.UpdateMod(ctxWt2, nil, newMod); err != nil {
		os.RemoveAll(dstPath)
		return err
	}
	os.RemoveAll(path.Join(h.workDirPath, oldMod.DirName))
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
	return nil
}
