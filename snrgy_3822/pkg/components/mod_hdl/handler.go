package mod_hdl

import (
	"context"
	"io/fs"
	"os"
	"sync"
)

type Handler struct {
	wrkPath string
	mu      sync.RWMutex
}

func New(wrkPath string) *Handler {
	return &Handler{
		wrkPath: wrkPath,
	}
}

func (h *Handler) Init() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if err := os.MkdirAll(h.wrkPath, 0775); err != nil {
		return err
	}
	return nil
}

func (h *Handler) Add(_ context.Context, modID string, modFS fs.FS) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	panic("not implemented")
}
