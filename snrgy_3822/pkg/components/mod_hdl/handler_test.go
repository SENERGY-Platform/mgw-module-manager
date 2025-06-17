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
	"errors"
	"fmt"
	cew_model "github.com/SENERGY-Platform/mgw-container-engine-wrapper/lib/model"
	job_hdl_lib "github.com/SENERGY-Platform/mgw-go-service-base/job-hdl/lib"
	module_lib "github.com/SENERGY-Platform/mgw-module-lib/model"
	models_error "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/error"
	models_module "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/module"
	models_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/storage"
	"io"
	"net/url"
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
	h := New(stgHdlMock, nil, &loggerMock{Writer: os.Stdout}, Config{WorkDirPath: "./test"})
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
	h := New(stgHdlMock, nil, &loggerMock{Writer: os.Stdout}, Config{WorkDirPath: "./test"})
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
		t.Run("file", func(t *testing.T) {
			bk := h.cache
			h.config.WorkDirPath = ""
			h.cache = make(map[string]module_lib.Module)
			_, err = h.Module(context.Background(), "github.com/org/repo")
			if err == nil {
				t.Error("expected error")
			}
			h.cache = bk
		})
		t.Run("storage", func(t *testing.T) {
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
	cewCltMock := &cewClientMock{Images: make(map[string]cew_model.Image), Jobs: make(map[string]job_hdl_lib.Job), JobCompleteDelay: time.Second * 1}
	workDir := t.TempDir()
	h := New(stgHdlMock, cewCltMock, &loggerMock{Writer: os.Stdout}, Config{WorkDirPath: workDir, JobPollInterval: time.Millisecond * 250})
	err := h.Init()
	if err != nil {
		t.Fatal(err)
	}
	t.Run("success", func(t *testing.T) {
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
		_, ok = cewCltMock.Images["ghcr.io/org/repo:test"]
		if !ok {
			t.Error("expected image in map")
		}
	})
	t.Run("error", func(t *testing.T) {
		t.Run("source err", func(t *testing.T) {
			stgHdlMock.Mods = make(map[string]models_storage.Module)
			err = h.Add(context.Background(), "github.com/org/repo", "test_source", "test_channel", os.DirFS(""))
			if err == nil {
				t.Error("expected error")
			}
		})
		t.Run("storage", func(t *testing.T) {
			testErr := errors.New("test error")
			stgHdlMock.Err = testErr
			stgHdlMock.Mods = make(map[string]models_storage.Module)
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
		t.Run("get image", func(t *testing.T) {
			testErr := errors.New("test error")
			cewCltMock.GetImageErr = testErr
			stgHdlMock.Mods = make(map[string]models_storage.Module)
			err = h.Add(context.Background(), "github.com/org/repo", "test_source", "test_channel", os.DirFS("./test/test_mod"))
			if err == nil {
				t.Error("expected error")
			}
			if !errors.Is(err, testErr) {
				t.Errorf("expected %v, got %v", testErr, err)
			}
			cewCltMock.GetImageErr = nil
		})
		t.Run("add image", func(t *testing.T) {
			testErr := errors.New("test error")
			cewCltMock.AddImageErr = testErr
			cewCltMock.Images = make(map[string]cew_model.Image)
			stgHdlMock.Mods = make(map[string]models_storage.Module)
			err = h.Add(context.Background(), "github.com/org/repo", "test_source", "test_channel", os.DirFS("./test/test_mod"))
			if err == nil {
				t.Error("expected error")
			}
			if !errors.Is(err, testErr) {
				t.Errorf("expected %v, got %v", testErr, err)
			}
			cewCltMock.AddImageErr = nil
		})
		t.Run("add image await job", func(t *testing.T) {
			testErr := errors.New("test error")
			cewCltMock.GetJobErr = testErr
			cewCltMock.Images = make(map[string]cew_model.Image)
			stgHdlMock.Mods = make(map[string]models_storage.Module)
			err = h.Add(context.Background(), "github.com/org/repo", "test_source", "test_channel", os.DirFS("./test/test_mod"))
			if err == nil {
				t.Error("expected error")
			}
			if !errors.Is(err, testErr) {
				t.Errorf("expected %v, got %v", testErr, err)
			}
			cewCltMock.GetJobErr = nil
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
	cewCltMock := &cewClientMock{Images: make(map[string]cew_model.Image), Jobs: make(map[string]job_hdl_lib.Job), JobCompleteDelay: time.Second * 1}
	workDir := t.TempDir()
	err := os.MkdirAll(path.Join(workDir, "test_dir"), 0775)
	if err != nil {
		t.Fatal(err)
	}
	h := New(stgHdlMock, cewCltMock, &loggerMock{Writer: os.Stdout}, Config{WorkDirPath: workDir})
	err = h.Init()
	if err != nil {
		t.Fatal(err)
	}
	t.Run("success", func(t *testing.T) {
		populateTestDir(t, workDir)
		err = h.Update(context.Background(), "github.com/org/repo", "test_source2", "test_channel2", os.DirFS("./test/test_mod_2"))
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
		b, err := os.ReadFile(path.Join(workDir, mod.DirName, "test"))
		if err != nil {
			t.Error(err)
		}
		if string(b) != "1" {
			t.Error("expected test file to contain '1'")
		}
	})
	t.Run("error", func(t *testing.T) {
		t.Run("source err", func(t *testing.T) {
			stgHdlMock.Mods = map[string]models_storage.Module{
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
			}
			err = h.Update(context.Background(), "github.com/org/repo", "test_source2", "test_channel2", os.DirFS(""))
			if err == nil {
				t.Error("expected error")
			}
		})
		t.Run("storage", func(t *testing.T) {
			testErr := errors.New("test error")
			stgHdlMock.Err = testErr
			stgHdlMock.Mods = map[string]models_storage.Module{
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
			}
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
			stgHdlMock.Mods = map[string]models_storage.Module{
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
			}
			err = h.Update(context.Background(), "test", "test_source2", "test_channel2", os.DirFS("./test/test_mod"))
			if err == nil {
				t.Error("expected error")
			}
		})
		t.Run("get image", func(t *testing.T) {
			testErr := errors.New("test error")
			cewCltMock.GetImageErr = testErr
			stgHdlMock.Mods = map[string]models_storage.Module{
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
			}
			err = h.Update(context.Background(), "github.com/org/repo", "test_source2", "test_channel2", os.DirFS("./test/test_mod_2"))
			if err == nil {
				t.Error("expected error")
			}
			if !errors.Is(err, testErr) {
				t.Errorf("expected %v, got %v", testErr, err)
			}
			cewCltMock.GetImageErr = nil
		})
		t.Run("add image", func(t *testing.T) {
			testErr := errors.New("test error")
			cewCltMock.AddImageErr = testErr
			cewCltMock.Images = make(map[string]cew_model.Image)
			stgHdlMock.Mods = map[string]models_storage.Module{
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
			}
			err = h.Update(context.Background(), "github.com/org/repo", "test_source2", "test_channel2", os.DirFS("./test/test_mod_2"))
			if err == nil {
				t.Error("expected error")
			}
			if !errors.Is(err, testErr) {
				t.Errorf("expected %v, got %v", testErr, err)
			}
			cewCltMock.AddImageErr = nil
		})
		t.Run("add image await job error", func(t *testing.T) {
			testErr := errors.New("test error")
			cewCltMock.GetJobErr = testErr
			cewCltMock.Images = make(map[string]cew_model.Image)
			stgHdlMock.Mods = map[string]models_storage.Module{
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
			}
			err = h.Update(context.Background(), "github.com/org/repo", "test_source2", "test_channel2", os.DirFS("./test/test_mod_2"))
			if err == nil {
				t.Error("expected error")
			}
			if !errors.Is(err, testErr) {
				t.Errorf("expected %v, got %v", testErr, err)
			}
			cewCltMock.GetJobErr = nil
		})
	})
}

func TestHandler_Delete(t *testing.T) {
	stgHdlMock := &storageHandlerMock{}
	cewCltMock := &cewClientMock{Images: make(map[string]cew_model.Image), Jobs: make(map[string]job_hdl_lib.Job), JobCompleteDelay: time.Second * 1}
	workDir := t.TempDir()
	err := os.MkdirAll(path.Join(workDir, "test_dir"), 0775)
	if err != nil {
		t.Fatal(err)
	}
	h := New(stgHdlMock, cewCltMock, &loggerMock{Writer: os.Stdout}, Config{WorkDirPath: workDir})
	err = h.Init()
	if err != nil {
		t.Fatal(err)
	}
	t.Run("success", func(t *testing.T) {
		t.Run("not in cache", func(t *testing.T) {
			populateTestDir(t, workDir)
			stgHdlMock.Mods = map[string]models_storage.Module{
				"github.com/org/repo": {
					ModuleBase: models_storage.ModuleBase{
						ID:      "github.com/org/repo",
						DirName: "test_dir",
					},
				},
			}
			cewCltMock.Images["ghcr.io/org/repo:test"] = cew_model.Image{}
			err = h.Remove(context.Background(), "github.com/org/repo")
			if err != nil {
				t.Error(err)
			}
			_, ok := stgHdlMock.Mods["github.com/org/repo"]
			if ok {
				t.Error("expected module to not exist")
			}
			_, err = os.Stat(path.Join(workDir, "test_dir"))
			if err != nil {
				if !os.IsNotExist(err) {
					t.Fatal(err)
				}
			}
			_, ok = cewCltMock.Images["ghcr.io/org/repo:test"]
			if ok {
				t.Error("expected image to not exist")
			}
		})
		t.Run("in cache", func(t *testing.T) {
			populateTestDir(t, workDir)
			stgHdlMock.Mods = map[string]models_storage.Module{
				"github.com/org/repo": {
					ModuleBase: models_storage.ModuleBase{
						ID:      "github.com/org/repo",
						DirName: "test_dir",
					},
				},
			}
			cewCltMock.Images["ghcr.io/org/repo:test"] = cew_model.Image{}
			h.cache["github.com/org/repo"] = module_lib.Module{Services: map[string]*module_lib.Service{"test": {Image: "ghcr.io/org/repo:test"}}}
			err = h.Remove(context.Background(), "github.com/org/repo")
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
			_, ok = cewCltMock.Images["ghcr.io/org/repo:test"]
			if ok {
				t.Error("expected image to not exist")
			}
		})
		t.Run("image not found", func(t *testing.T) {
			populateTestDir(t, workDir)
			stgHdlMock.Mods = map[string]models_storage.Module{
				"github.com/org/repo": {
					ModuleBase: models_storage.ModuleBase{
						ID:      "github.com/org/repo",
						DirName: "test_dir",
					},
				},
			}
			cewCltMock.Images = make(map[string]cew_model.Image)
			err = h.Remove(context.Background(), "github.com/org/repo")
			if err != nil {
				t.Error(err)
			}
			_, ok := stgHdlMock.Mods["github.com/org/repo"]
			if ok {
				t.Error("expected module to not exist")
			}
			_, err = os.Stat(path.Join(workDir, "test_dir"))
			if err != nil {
				if !os.IsNotExist(err) {
					t.Fatal(err)
				}
			}
		})
	})
	t.Run("error", func(t *testing.T) {
		t.Run("storage", func(t *testing.T) {
			populateTestDir(t, workDir)
			stgHdlMock.Mods = map[string]models_storage.Module{
				"github.com/org/repo": {
					ModuleBase: models_storage.ModuleBase{
						ID:      "github.com/org/repo",
						DirName: "test_dir",
					},
				},
			}
			cewCltMock.Images["ghcr.io/org/repo:test"] = cew_model.Image{}
			testErr := errors.New("test error")
			stgHdlMock.Err = testErr
			err = h.Remove(context.Background(), "github.com/org/repo")
			if err == nil {
				t.Error("expected error")
			}
			if !errors.Is(err, testErr) {
				t.Errorf("expected %v, got %v", testErr, err)
			}
			stgHdlMock.Err = nil
		})
		t.Run("does not exist", func(t *testing.T) {
			populateTestDir(t, workDir)
			stgHdlMock.Mods = map[string]models_storage.Module{
				"github.com/org/repo": {
					ModuleBase: models_storage.ModuleBase{
						ID:      "github.com/org/repo",
						DirName: "test_dir",
					},
				},
			}
			cewCltMock.Images["ghcr.io/org/repo:test"] = cew_model.Image{}
			err = h.Remove(context.Background(), "test")
			if err == nil {
				t.Error("expected error")
			}
		})
		t.Run("remove image", func(t *testing.T) {
			populateTestDir(t, workDir)
			stgHdlMock.Mods = map[string]models_storage.Module{
				"github.com/org/repo": {
					ModuleBase: models_storage.ModuleBase{
						ID:      "github.com/org/repo",
						DirName: "test_dir",
					},
				},
			}
			testErr := errors.New("test error")
			cewCltMock.RemoveImageErr = testErr
			cewCltMock.Images["ghcr.io/org/repo:test"] = cew_model.Image{}
			err = h.Remove(context.Background(), "github.com/org/repo")
			if err == nil {
				t.Error("expected error")
			}
			if !errors.Is(err, testErr) {
				t.Errorf("expected %v, got %v", testErr, err)
			}
			stgHdlMock.Err = nil
		})
	})
}

func populateTestDir(t *testing.T, workDir string) {
	sf, err := os.Open("./test/test_mod/Modfile.yml")
	if err != nil {
		t.Fatal(err)
	}
	defer sf.Close()
	err = os.MkdirAll(path.Join(workDir, "test_dir"), 0775)
	if err != nil {
		t.Fatal(err)
	}
	df, err := os.Create(path.Join(workDir, "test_dir/Modfile.yml"))
	if err != nil {
		t.Fatal(err)
	}
	defer df.Close()
	_, err = io.Copy(df, sf)
	if err != nil {
		t.Fatal(err)
	}
}

type storageHandlerMock struct {
	Err  error
	Mods map[string]models_storage.Module
}

func (m *storageHandlerMock) ListMod(_ context.Context, _ models_storage.ModuleFilter) (map[string]models_storage.Module, error) {
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

func (m *storageHandlerMock) CreateMod(_ context.Context, mod models_storage.ModuleBase) error {
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

func (m *storageHandlerMock) UpdateMod(_ context.Context, mod models_storage.ModuleBase) error {
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

func (m *storageHandlerMock) DeleteMod(_ context.Context, id string) error {
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

type cewClientMock struct {
	Images           map[string]cew_model.Image
	Jobs             map[string]job_hdl_lib.Job
	JobCompleteDelay time.Duration
	GetImageErr      error
	AddImageErr      error
	RemoveImageErr   error
	GetJobErr        error
	CancelJobErr     error
}

func (m *cewClientMock) GetImage(_ context.Context, id string) (cew_model.Image, error) {
	if m.GetImageErr != nil {
		return cew_model.Image{}, m.GetImageErr
	}
	img, ok := m.Images[id]
	if !ok {
		return cew_model.Image{}, cew_model.NewNotFoundError(errors.New("image not found"))
	}
	return img, nil
}

func (m *cewClientMock) AddImage(_ context.Context, img string) (jobId string, err error) {
	if m.AddImageErr != nil {
		return "", m.AddImageErr
	}
	m.Images[img] = cew_model.Image{}
	jID := fmt.Sprintf("%d", len(m.Jobs))
	timestamp := time.Now().UTC()
	m.Jobs[jID] = job_hdl_lib.Job{
		ID:      jID,
		Created: timestamp,
		Started: &timestamp,
	}
	return jID, nil
}

func (m *cewClientMock) RemoveImage(_ context.Context, id string) error {
	if m.RemoveImageErr != nil {
		return m.RemoveImageErr
	}
	id, err := url.QueryUnescape(id)
	if err != nil {
		return err
	}
	id, err = url.QueryUnescape(id)
	if err != nil {
		return err
	}
	_, ok := m.Images[id]
	if !ok {
		return cew_model.NewNotFoundError(errors.New("image not found"))
	}
	delete(m.Images, id)
	return nil
}

func (m *cewClientMock) GetJob(_ context.Context, jID string) (job_hdl_lib.Job, error) {
	if m.GetJobErr != nil {
		return job_hdl_lib.Job{}, m.GetJobErr
	}
	job, ok := m.Jobs[jID]
	if !ok {
		return job_hdl_lib.Job{}, errors.New("not found")
	}
	if time.Since(*job.Started) >= m.JobCompleteDelay {
		timestamp := time.Now().UTC()
		job.Completed = &timestamp
	}
	return job, nil
}

func (m *cewClientMock) CancelJob(_ context.Context, jID string) error {
	if m.CancelJobErr != nil {
		return m.CancelJobErr
	}
	_, ok := m.Jobs[jID]
	if !ok {
		return errors.New("not found")
	}
	return nil
}

type loggerMock struct {
	Writer io.Writer
}

func (m *loggerMock) Errorf(format string, v ...any) {
	fmt.Fprintf(m.Writer, "ERROR "+format+"\n", v...)
}

func (m *loggerMock) Warningf(format string, v ...any) {
	fmt.Fprintf(m.Writer, "WARNING "+format+"\n", v...)
}
