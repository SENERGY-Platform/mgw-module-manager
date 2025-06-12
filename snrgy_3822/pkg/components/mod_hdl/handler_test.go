/*
 * Copyright 2025 InfAI (CC SES)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package mod_hdl

import (
	"context"
	"database/sql/driver"
	"errors"
	module_lib "github.com/SENERGY-Platform/mgw-module-lib/model"
	models_error "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
	models_module "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/module"
	models_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/storage"
	"os"
	"path"
	"reflect"
	"testing"
	"time"
)

func TestHandler_Modules(t *testing.T) {
	timestamp := time.Now().UTC()
	stgHdlMock := &storageHandlerMock{Mods: map[string]models_storage.Module{
		"github.com/org/repo": {
			ModuleBase: models_storage.ModuleBase{
				ID:      "github.com/org/repo",
				DirName: "test_mod",
				Source:  "test_source",
				Channel: "test_channel",
			},
			Added:   timestamp,
			Updated: timestamp,
		},
	}}
	a := models_module.ModuleAbbreviated{
		ID:      "github.com/org/repo",
		Name:    "Test Module",
		Desc:    "Module for tests.",
		Version: "v1.0.0",
		ModuleBase: models_module.ModuleBase{
			Source:  "test_source",
			Channel: "test_channel",
			Added:   timestamp,
			Updated: timestamp,
		},
	}
	h := New(stgHdlMock, "./test", time.Second)
	err := h.Init()
	if err != nil {
		t.Fatal(err)
	}
	mods, err := h.Modules(context.Background(), models_module.ModuleFilter{})
	if err != nil {
		t.Error(err)
	}
	if len(mods) != 1 {
		t.Errorf("expected len 1 but got %d", len(mods))
	}
	b, ok := mods["github.com/org/repo"]
	if !ok {
		t.Error(errors.New("module not in map"))
	}
	if !reflect.DeepEqual(a, b) {
		t.Errorf("expected %v, got %v", a, b)
	}
}

func TestHandler_Module(t *testing.T) {
	timestamp := time.Now().UTC()
	stgHdlMock := &storageHandlerMock{Mods: map[string]models_storage.Module{
		"github.com/org/repo": {
			ModuleBase: models_storage.ModuleBase{
				ID:      "github.com/org/repo",
				DirName: "test_mod",
				Source:  "test_source",
				Channel: "test_channel",
			},
			Added:   timestamp,
			Updated: timestamp,
		},
	}}
	a := models_module.ModuleBase{
		Source:  "test_source",
		Channel: "test_channel",
		Added:   timestamp,
		Updated: timestamp,
	}
	h := New(stgHdlMock, "./test", time.Second)
	err := h.Init()
	if err != nil {
		t.Fatal(err)
	}
	mod, err := h.Module(context.Background(), "github.com/org/repo")
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(a, mod.ModuleBase) {
		t.Errorf("expected: %v, got: %v", a, mod.ModuleBase)
	}
	t.Run("error", func(t *testing.T) {
		t.Run("does not exist", func(t *testing.T) {
			_, err = h.Module(context.Background(), "test")
			if err == nil {
				t.Error("expected error")
			}
		})
		t.Run("file error", func(t *testing.T) {
			bk := h.cache
			h.workDirPath = ""
			h.cache = make(map[string]module_lib.Module)
			_, err = h.Module(context.Background(), "github.com/org/repo")
			if err == nil {
				t.Error("expected error")
			}
			h.cache = bk
		})
		t.Run("storage error", func(t *testing.T) {
			testErr := errors.New("test error")
			stgHdlMock.Err = testErr
			_, err = h.Module(context.Background(), "github.com/org/repo")
			if err == nil {
				t.Error("expected error")
			}
			if !errors.Is(err, testErr) {
				t.Errorf("expected: %v, got: %v", testErr, err)
			}
		})
	})

}

func TestHandler_Add(t *testing.T) {
	stgHdlMock := &storageHandlerMock{Mods: make(map[string]models_storage.Module)}
	workDir := t.TempDir()
	h := New(stgHdlMock, workDir, time.Second)
	err := h.Init()
	if err != nil {
		t.Fatal(err)
	}
	err = h.Add(context.Background(), "github.com/org/repo", "test_source", "test_channel", os.DirFS("./test/test_mod"))
	if err != nil {
		t.Error(err)
	}
	_, ok := h.cache["github.com/org/repo"]
	if !ok {
		t.Error(errors.New("module not in cache"))
	}
	mod, ok := stgHdlMock.Mods["github.com/org/repo"]
	if !ok {
		t.Error("expected module to exist")
	}
	if mod.Source != "test_source" {
		t.Error("expected module source to be test_source")
	}
	if mod.Channel != "test_channel" {
		t.Error("expected module channel to be test_channel")
	}
	_, err = os.Stat(path.Join(workDir, mod.DirName, "Modfile.yml"))
	if err != nil {
		t.Error(err)
	}
	t.Run("error", func(t *testing.T) {
		t.Run("source err", func(t *testing.T) {
			err = h.Add(context.Background(), "github.com/org/repo", "test_source", "test_channel", os.DirFS(""))
			if err == nil {
				t.Error("expected error")
			}
		})
		t.Run("storage error", func(t *testing.T) {
			testErr := errors.New("test error")
			stgHdlMock.Err = testErr
			err = h.Add(context.Background(), "github.com/org/repo", "test_source", "test_channel", os.DirFS("./test/test_mod"))
			if err == nil {
				t.Error("expected error")
			}
			if !errors.Is(err, testErr) {
				t.Errorf("expected %v, got %v", testErr, err)
			}
			stgHdlMock.Err = nil
		})
		t.Run("already exists", func(t *testing.T) {
			timestamp := time.Now().UTC()
			stgHdlMock.Mods = map[string]models_storage.Module{
				"github.com/org/repo": {
					ModuleBase: models_storage.ModuleBase{
						ID:      "github.com/org/repo",
						DirName: "test_mod",
						Source:  "test_source",
						Channel: "test_channel",
					},
					Added:   timestamp,
					Updated: timestamp,
				},
			}
			err = h.Add(context.Background(), "github.com/org/repo", "test_source", "test_channel", os.DirFS("./test/test_mod"))
			if err == nil {
				t.Error("expected error")
			}
		})
	})
}

func TestHandler_Update(t *testing.T) {
	timestamp := time.Now().UTC()
	stgHdlMock := &storageHandlerMock{Mods: map[string]models_storage.Module{
		"github.com/org/repo": {
			ModuleBase: models_storage.ModuleBase{
				ID:      "github.com/org/repo",
				DirName: "test_dir",
				Source:  "test_source",
				Channel: "test_channel",
			},
			Added:   timestamp,
			Updated: timestamp,
		},
	}}
	workDir := t.TempDir()
	err := os.MkdirAll(path.Join(workDir, "test_dir"), 0775)
	if err != nil {
		t.Fatal(err)
	}
	h := New(stgHdlMock, workDir, time.Second)
	err = h.Init()
	if err != nil {
		t.Fatal(err)
	}
	err = h.Update(context.Background(), "github.com/org/repo", "test_source2", "test_channel2", os.DirFS("./test/test_mod"))
	if err != nil {
		t.Error(err)
	}
	_, ok := h.cache["github.com/org/repo"]
	if !ok {
		t.Error(errors.New("module not in cache"))
	}
	mod, ok := stgHdlMock.Mods["github.com/org/repo"]
	if !ok {
		t.Error("expected module to exist")
	}
	if mod.Source != "test_source2" {
		t.Error("expected module source to be test_source2")
	}
	if mod.Channel != "test_channel2" {
		t.Error("expected module channel to be test_channel2")
	}
	if mod.DirName == "test_dir" {
		t.Error("expected different dir name")
	}
	if mod.Added == mod.Updated {
		t.Error("expected timestamp to be updated")
	}
	_, err = os.Stat(path.Join(workDir, mod.DirName, "Modfile.yml"))
	if err != nil {
		t.Error(err)
	}
	_, err = os.Stat(path.Join(workDir, "test_dir"))
	if err != nil {
		if !os.IsNotExist(err) {
			t.Fatal(err)
		}
	} else {
		t.Error("expected error")
	}
	t.Run("error", func(t *testing.T) {
		t.Run("source err", func(t *testing.T) {
			err = h.Update(context.Background(), "github.com/org/repo", "test_source2", "test_channel2", os.DirFS(""))
			if err == nil {
				t.Error("expected error")
			}
		})
		t.Run("storage error", func(t *testing.T) {
			testErr := errors.New("test error")
			stgHdlMock.Err = testErr
			err = h.Update(context.Background(), "github.com/org/repo", "test_source2", "test_channel2", os.DirFS("./test/test_mod"))
			if err == nil {
				t.Error("expected error")
			}
			if !errors.Is(err, testErr) {
				t.Errorf("expected %v, got %v", testErr, err)
			}
			stgHdlMock.Err = nil
		})
		t.Run("does not exist", func(t *testing.T) {
			err = h.Update(context.Background(), "test", "test_source2", "test_channel2", os.DirFS("./test/test_mod"))
			if err == nil {
				t.Error("expected error")
			}
		})
	})
}

func TestHandler_Delete(t *testing.T) {
	stgHdlMock := &storageHandlerMock{Mods: map[string]models_storage.Module{
		"github.com/org/repo": {
			ModuleBase: models_storage.ModuleBase{
				ID:      "github.com/org/repo",
				DirName: "test_dir",
			},
		},
	}}
	workDir := t.TempDir()
	err := os.MkdirAll(path.Join(workDir, "test_dir"), 0775)
	if err != nil {
		t.Fatal(err)
	}
	h := New(stgHdlMock, workDir, time.Second)
	err = h.Init()
	if err != nil {
		t.Fatal(err)
	}
	h.cache["github.com/org/repo"] = module_lib.Module{}
	err = h.Delete(context.Background(), "github.com/org/repo")
	if err != nil {
		t.Error(err)
	}
	_, ok := h.cache["github.com/org/repo"]
	if ok {
		t.Error(errors.New("module should not be in cache"))
	}
	_, ok = stgHdlMock.Mods["github.com/org/repo"]
	if ok {
		t.Error("expected module to not exist")
	}
	_, err = os.Stat(path.Join(workDir, "test_dir"))
	if err != nil {
		if !os.IsNotExist(err) {
			t.Fatal(err)
		}
	}
	t.Run("error", func(t *testing.T) {
		t.Run("storage error", func(t *testing.T) {
			testErr := errors.New("test error")
			stgHdlMock.Err = testErr
			err = h.Delete(context.Background(), "github.com/org/repo")
			if err == nil {
				t.Error("expected error")
			}
			if !errors.Is(err, testErr) {
				t.Errorf("expected %v, got %v", testErr, err)
			}
			stgHdlMock.Err = nil
		})
		t.Run("does not exist", func(t *testing.T) {
			err = h.Delete(context.Background(), "test")
			if err == nil {
				t.Error("expected error")
			}
		})
	})
}

type storageHandlerMock struct {
	Err  error
	Mods map[string]models_storage.Module
}

func (m *storageHandlerMock) ListMod(ctx context.Context, filter models_storage.ModuleFilter) (map[string]models_storage.Module, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Mods, nil
}

func (m *storageHandlerMock) ReadMod(_ context.Context, id string) (models_storage.Module, error) {
	if m.Err != nil {
		return models_storage.Module{}, m.Err
	}
	mod, ok := m.Mods[id]
	if !ok {
		return models_storage.Module{}, models_error.NewNotFoundError(errors.New("module not found"))
	}
	return mod, nil
}

func (m *storageHandlerMock) CreateMod(_ context.Context, _ driver.Tx, mod models_storage.ModuleBase) error {
	if m.Err != nil {
		return m.Err
	}
	_, ok := m.Mods[mod.ID]
	if ok {
		return errors.New("already exists")
	}
	timestamp := time.Now().UTC()
	m.Mods[mod.ID] = models_storage.Module{
		ModuleBase: mod,
		Added:      timestamp,
		Updated:    timestamp,
	}
	return nil
}

func (m *storageHandlerMock) UpdateMod(_ context.Context, _ driver.Tx, mod models_storage.ModuleBase) error {
	if m.Err != nil {
		return m.Err
	}
	tmp, ok := m.Mods[mod.ID]
	if !ok {
		return models_error.NewNotFoundError(errors.New("module not found"))
	}
	m.Mods[mod.ID] = models_storage.Module{
		ModuleBase: mod,
		Added:      tmp.Added,
		Updated:    time.Now().UTC(),
	}
	return nil
}

func (m *storageHandlerMock) DeleteMod(_ context.Context, _ driver.Tx, id string) error {
	if m.Err != nil {
		return m.Err
	}
	_, ok := m.Mods[id]
	if !ok {
		return models_error.NewNotFoundError(errors.New("module not found"))
	}
	delete(m.Mods, id)
	return nil
}
