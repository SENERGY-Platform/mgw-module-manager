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

package modules

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"reflect"
	"testing"
	"time"

	module_lib "github.com/SENERGY-Platform/mgw-module-lib/model"
	lib_errors "github.com/SENERGY-Platform/mgw-module-manager/lib/errors"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

func TestHandler_Modules(t *testing.T) {
	timestamp := time.Now().UTC()
	stgHdlMock := &storageHandlerMock{Mods: map[string]pkg_models.DatabaseModule{
		"github.com/org/repo": {
			Id:      "github.com/org/repo",
			DirName: "test_mod",
			Source:  "test_source",
			Channel: "test_channel",
			Added:   timestamp,
			Updated: timestamp,
		},
	}}
	a := pkg_models.Module{
		ModuleLibModule: external_models.ModuleLibModule{
			ID:          "github.com/org/repo",
			Name:        "Test Module",
			Description: "Module for tests.",
			Version:     "v1.0.0",
		},
		Source:     "test_source",
		Channel:    "test_channel",
		Added:      timestamp,
		Updated:    timestamp,
		FileSystem: os.DirFS("test/test_mod"),
	}
	h := New(stgHdlMock, nil, Config{WorkdirPath: "./test"})
	err := h.CreateWorkDir()
	if err != nil {
		t.Fatal(err)
	}
	mods, err := h.GetModules(context.Background(), pkg_models.ModulesFilterWithName{}, false)
	if err != nil {
		t.Error(err)
	}
	if len(mods) != 1 {
		t.Errorf("expected len 1 but got %d", len(mods))
	}
	b, ok := mods["github.com/org/repo"]
	if !ok {
		t.Error(errors.New("module not in slice"))
	}
	if a.Version != b.Version {
		t.Errorf("expected %v, got %v", a.Version, b.Version)
	}
	a.ModuleLibModule = external_models.ModuleLibModule{}
	b.ModuleLibModule = external_models.ModuleLibModule{}
	if !reflect.DeepEqual(a, b) {
		t.Errorf("expected %v, got %v", a, b)
	}
}

func TestHandler_Module(t *testing.T) {
	timestamp := time.Now().UTC()
	stgHdlMock := &storageHandlerMock{Mods: map[string]pkg_models.DatabaseModule{
		"github.com/org/repo": {
			Id:      "github.com/org/repo",
			DirName: "test_mod",
			Source:  "test_source",
			Channel: "test_channel",
			Added:   timestamp,
			Updated: timestamp,
		},
	}}
	a := pkg_models.Module{
		ModuleLibModule: external_models.ModuleLibModule{
			ID:          "github.com/org/repo",
			Name:        "Test Module",
			Description: "Module for tests.",
			Version:     "v1.0.0",
		},
		Source:     "test_source",
		Channel:    "test_channel",
		Added:      timestamp,
		Updated:    timestamp,
		FileSystem: os.DirFS("test/test_mod"),
	}
	h := New(stgHdlMock, nil, Config{WorkdirPath: "./test"})
	err := h.CreateWorkDir()
	if err != nil {
		t.Fatal(err)
	}
	mod, err := h.GetModule(context.Background(), "github.com/org/repo")
	if err != nil {
		t.Error(err)
	}
	if a.Version != mod.Version {
		t.Errorf("expected %v, got %v", a.Version, mod.Version)
	}
	a.ModuleLibModule = external_models.ModuleLibModule{}
	mod.ModuleLibModule = external_models.ModuleLibModule{}
	if !reflect.DeepEqual(a, mod) {
		t.Errorf("expected: %v, got: %v", a, mod)
	}
	t.Run("error", func(t *testing.T) {
		t.Run("does not exist", func(t *testing.T) {
			_, err = h.GetModule(context.Background(), "test")
			if err == nil {
				t.Error("expected error")
			}
		})
		t.Run("file", func(t *testing.T) {
			bk := h.cache
			h.config.WorkdirPath = ""
			h.cache = make(map[string]module_lib.Module)
			_, err = h.GetModule(context.Background(), "github.com/org/repo")
			if err == nil {
				t.Error("expected error")
			}
			h.cache = bk
		})
		t.Run("storage", func(t *testing.T) {
			testErr := errors.New("test error")
			stgHdlMock.Err = testErr
			_, err = h.GetModule(context.Background(), "github.com/org/repo")
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
	stgHdlMock := &storageHandlerMock{Mods: make(map[string]pkg_models.DatabaseModule)}
	cewCltMock := &cewClientMock{Images: make(map[string]external_models.CewImage), Jobs: make(map[string]external_models.JobLibJob), JobCompleteDelay: time.Second * 1}
	workDir := t.TempDir()
	h := New(stgHdlMock, cewCltMock, Config{WorkdirPath: workDir, JobPollInterval: time.Millisecond * 250})
	err := h.CreateWorkDir()
	if err != nil {
		t.Fatal(err)
	}
	t.Run("success", func(t *testing.T) {
		err = h.AddModule(context.Background(), "github.com/org/repo", "test_source", "test_channel", os.DirFS("./test/test_mod"))
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
			stgHdlMock.Mods = make(map[string]pkg_models.DatabaseModule)
			err = h.AddModule(context.Background(), "github.com/org/repo", "test_source", "test_channel", os.DirFS(""))
			if err == nil {
				t.Error("expected error")
			}
		})
		t.Run("storage", func(t *testing.T) {
			testErr := errors.New("test error")
			stgHdlMock.Err = testErr
			stgHdlMock.Mods = make(map[string]pkg_models.DatabaseModule)
			err = h.AddModule(context.Background(), "github.com/org/repo", "test_source", "test_channel", os.DirFS("./test/test_mod"))
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
			stgHdlMock.Mods = map[string]pkg_models.DatabaseModule{
				"github.com/org/repo": {
					Id:      "github.com/org/repo",
					DirName: "test_mod",
					Source:  "test_source",
					Channel: "test_channel",
					Added:   timestamp,
					Updated: timestamp,
				},
			}
			err = h.AddModule(context.Background(), "github.com/org/repo", "test_source", "test_channel", os.DirFS("./test/test_mod"))
			if err == nil {
				t.Error("expected error")
			}
		})
		t.Run("get image", func(t *testing.T) {
			testErr := errors.New("test error")
			cewCltMock.GetImageErr = testErr
			stgHdlMock.Mods = make(map[string]pkg_models.DatabaseModule)
			err = h.AddModule(context.Background(), "github.com/org/repo", "test_source", "test_channel", os.DirFS("./test/test_mod"))
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
			cewCltMock.Images = make(map[string]external_models.CewImage)
			stgHdlMock.Mods = make(map[string]pkg_models.DatabaseModule)
			err = h.AddModule(context.Background(), "github.com/org/repo", "test_source", "test_channel", os.DirFS("./test/test_mod"))
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
			cewCltMock.Images = make(map[string]external_models.CewImage)
			stgHdlMock.Mods = make(map[string]pkg_models.DatabaseModule)
			err = h.AddModule(context.Background(), "github.com/org/repo", "test_source", "test_channel", os.DirFS("./test/test_mod"))
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
	stgHdlMock := &storageHandlerMock{Mods: map[string]pkg_models.DatabaseModule{
		"github.com/org/repo": {
			Id:      "github.com/org/repo",
			DirName: "test_dir",
			Source:  "test_source",
			Channel: "test_channel",
			Added:   timestamp,
			Updated: timestamp,
		},
	}}
	cewCltMock := &cewClientMock{Images: make(map[string]external_models.CewImage), Jobs: make(map[string]external_models.JobLibJob), JobCompleteDelay: time.Second * 1}
	workDir := t.TempDir()
	err := os.MkdirAll(path.Join(workDir, "test_dir"), 0775)
	if err != nil {
		t.Fatal(err)
	}
	h := New(stgHdlMock, cewCltMock, Config{WorkdirPath: workDir})
	err = h.CreateWorkDir()
	if err != nil {
		t.Fatal(err)
	}
	t.Run("success", func(t *testing.T) {
		populateTestDir(t, workDir)
		err = h.UpdateModule(context.Background(), "github.com/org/repo", "test_source2", "test_channel2", os.DirFS("./test/test_mod_2"))
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
			stgHdlMock.Mods = map[string]pkg_models.DatabaseModule{
				"github.com/org/repo": {
					Id:      "github.com/org/repo",
					DirName: "test_dir",
					Source:  "test_source",
					Channel: "test_channel",
					Added:   timestamp,
					Updated: timestamp,
				},
			}
			err = h.UpdateModule(context.Background(), "github.com/org/repo", "test_source2", "test_channel2", os.DirFS(""))
			if err == nil {
				t.Error("expected error")
			}
		})
		t.Run("storage", func(t *testing.T) {
			testErr := errors.New("test error")
			stgHdlMock.Err = testErr
			stgHdlMock.Mods = map[string]pkg_models.DatabaseModule{
				"github.com/org/repo": {
					Id:      "github.com/org/repo",
					DirName: "test_dir",
					Source:  "test_source",
					Channel: "test_channel",
					Added:   timestamp,
					Updated: timestamp,
				},
			}
			err = h.UpdateModule(context.Background(), "github.com/org/repo", "test_source2", "test_channel2", os.DirFS("./test/test_mod"))
			if err == nil {
				t.Error("expected error")
			}
			if !errors.Is(err, testErr) {
				t.Errorf("expected %v, got %v", testErr, err)
			}
			stgHdlMock.Err = nil
		})
		t.Run("does not exist", func(t *testing.T) {
			stgHdlMock.Mods = map[string]pkg_models.DatabaseModule{
				"github.com/org/repo": {
					Id:      "github.com/org/repo",
					DirName: "test_dir",
					Source:  "test_source",
					Channel: "test_channel",
					Added:   timestamp,
					Updated: timestamp,
				},
			}
			err = h.UpdateModule(context.Background(), "test", "test_source2", "test_channel2", os.DirFS("./test/test_mod"))
			if err == nil {
				t.Error("expected error")
			}
		})
		t.Run("get image", func(t *testing.T) {
			testErr := errors.New("test error")
			cewCltMock.GetImageErr = testErr
			stgHdlMock.Mods = map[string]pkg_models.DatabaseModule{
				"github.com/org/repo": {
					Id:      "github.com/org/repo",
					DirName: "test_dir",
					Source:  "test_source",
					Channel: "test_channel",
					Added:   timestamp,
					Updated: timestamp,
				},
			}
			err = h.UpdateModule(context.Background(), "github.com/org/repo", "test_source2", "test_channel2", os.DirFS("./test/test_mod_2"))
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
			cewCltMock.Images = make(map[string]external_models.CewImage)
			stgHdlMock.Mods = map[string]pkg_models.DatabaseModule{
				"github.com/org/repo": {
					Id:      "github.com/org/repo",
					DirName: "test_dir",
					Source:  "test_source",
					Channel: "test_channel",
					Added:   timestamp,
					Updated: timestamp,
				},
			}
			err = h.UpdateModule(context.Background(), "github.com/org/repo", "test_source2", "test_channel2", os.DirFS("./test/test_mod_2"))
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
			cewCltMock.Images = make(map[string]external_models.CewImage)
			stgHdlMock.Mods = map[string]pkg_models.DatabaseModule{
				"github.com/org/repo": {
					Id:      "github.com/org/repo",
					DirName: "test_dir",
					Source:  "test_source",
					Channel: "test_channel",
					Added:   timestamp,
					Updated: timestamp,
				},
			}
			err = h.UpdateModule(context.Background(), "github.com/org/repo", "test_source2", "test_channel2", os.DirFS("./test/test_mod_2"))
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
	cewCltMock := &cewClientMock{Images: make(map[string]external_models.CewImage), Jobs: make(map[string]external_models.JobLibJob), JobCompleteDelay: time.Second * 1}
	workDir := t.TempDir()
	err := os.MkdirAll(path.Join(workDir, "test_dir"), 0775)
	if err != nil {
		t.Fatal(err)
	}
	h := New(stgHdlMock, cewCltMock, Config{WorkdirPath: workDir})
	err = h.CreateWorkDir()
	if err != nil {
		t.Fatal(err)
	}
	t.Run("success", func(t *testing.T) {
		t.Run("not in cache", func(t *testing.T) {
			populateTestDir(t, workDir)
			stgHdlMock.Mods = map[string]pkg_models.DatabaseModule{
				"github.com/org/repo": {
					Id:      "github.com/org/repo",
					DirName: "test_dir",
				},
			}
			cewCltMock.Images["ghcr.io/org/repo:test"] = external_models.CewImage{}
			err = h.DeleteModule(context.Background(), "github.com/org/repo")
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
			stgHdlMock.Mods = map[string]pkg_models.DatabaseModule{
				"github.com/org/repo": {
					Id:      "github.com/org/repo",
					DirName: "test_dir",
				},
			}
			cewCltMock.Images["ghcr.io/org/repo:test"] = external_models.CewImage{}
			h.cache["github.com/org/repo"] = module_lib.Module{Services: map[string]module_lib.Service{"test": {Image: "ghcr.io/org/repo:test"}}}
			err = h.DeleteModule(context.Background(), "github.com/org/repo")
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
			stgHdlMock.Mods = map[string]pkg_models.DatabaseModule{
				"github.com/org/repo": {
					Id:      "github.com/org/repo",
					DirName: "test_dir",
				},
			}
			cewCltMock.Images = make(map[string]external_models.CewImage)
			err = h.DeleteModule(context.Background(), "github.com/org/repo")
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
			stgHdlMock.Mods = map[string]pkg_models.DatabaseModule{
				"github.com/org/repo": {
					Id:      "github.com/org/repo",
					DirName: "test_dir",
				},
			}
			cewCltMock.Images["ghcr.io/org/repo:test"] = external_models.CewImage{}
			testErr := errors.New("test error")
			stgHdlMock.Err = testErr
			err = h.DeleteModule(context.Background(), "github.com/org/repo")
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
			stgHdlMock.Mods = map[string]pkg_models.DatabaseModule{
				"github.com/org/repo": {
					Id:      "github.com/org/repo",
					DirName: "test_dir",
				},
			}
			cewCltMock.Images["ghcr.io/org/repo:test"] = external_models.CewImage{}
			err = h.DeleteModule(context.Background(), "test")
			if err != nil {
				t.Error("unexpected error")
			}
		})
		t.Run("remove image", func(t *testing.T) {
			populateTestDir(t, workDir)
			stgHdlMock.Mods = map[string]pkg_models.DatabaseModule{
				"github.com/org/repo": {
					Id:      "github.com/org/repo",
					DirName: "test_dir",
				},
			}
			testErr := errors.New("test error")
			cewCltMock.RemoveImageErr = testErr
			cewCltMock.Images["ghcr.io/org/repo:test"] = external_models.CewImage{}
			err = h.DeleteModule(context.Background(), "github.com/org/repo")
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
	Mods map[string]pkg_models.DatabaseModule
}

func (m *storageHandlerMock) ReadModules(_ context.Context, filter pkg_models.ModulesFilter) (map[string]pkg_models.DatabaseModule, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	if len(filter.Ids) > 0 {
		mods := make(map[string]pkg_models.DatabaseModule)
		for _, id := range filter.Ids {
			mod, ok := m.Mods[id]
			if ok {
				mods[id] = mod
			}
		}
		return mods, nil
	}
	return m.Mods, nil
}

func (m *storageHandlerMock) ReadModule(_ context.Context, id string) (pkg_models.DatabaseModule, error) {
	if m.Err != nil {
		return pkg_models.DatabaseModule{}, m.Err
	}
	mod, ok := m.Mods[id]
	if !ok {
		return pkg_models.DatabaseModule{}, lib_errors.New[lib_errors.ErrNotFound]("not found")
	}
	return mod, nil
}

func (m *storageHandlerMock) CreateModule(_ context.Context, mod pkg_models.DatabaseModule) error {
	if m.Err != nil {
		return m.Err
	}
	_, ok := m.Mods[mod.Id]
	if ok {
		return errors.New("already exists")
	}
	m.Mods[mod.Id] = mod
	return nil
}

func (m *storageHandlerMock) UpdateModule(_ context.Context, mod pkg_models.DatabaseModule) error {
	if m.Err != nil {
		return m.Err
	}
	_, ok := m.Mods[mod.Id]
	if !ok {
		return lib_errors.New[lib_errors.ErrNotFound]("not found")
	}
	m.Mods[mod.Id] = mod
	return nil
}

func (m *storageHandlerMock) DeleteModule(_ context.Context, id string) error {
	if m.Err != nil {
		return m.Err
	}
	_, ok := m.Mods[id]
	if !ok {
		return nil
	}
	delete(m.Mods, id)
	return nil
}

type cewClientMock struct {
	Images           map[string]external_models.CewImage
	Jobs             map[string]external_models.JobLibJob
	JobCompleteDelay time.Duration
	GetImageErr      error
	AddImageErr      error
	RemoveImageErr   error
	GetJobErr        error
	CancelJobErr     error
}

func (m *cewClientMock) GetImage(_ context.Context, id string) (external_models.CewImage, error) {
	if m.GetImageErr != nil {
		return external_models.CewImage{}, m.GetImageErr
	}
	img, ok := m.Images[id]
	if !ok {
		return external_models.CewImage{}, &external_models.CewNotFoundErr{}
	}
	return img, nil
}

func (m *cewClientMock) AddImage(_ context.Context, img string) (jobId string, err error) {
	if m.AddImageErr != nil {
		return "", m.AddImageErr
	}
	m.Images[img] = external_models.CewImage{}
	jID := fmt.Sprintf("%d", len(m.Jobs))
	timestamp := time.Now().UTC()
	m.Jobs[jID] = external_models.JobLibJob{
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
		return &external_models.CewNotFoundErr{}
	}
	delete(m.Images, id)
	return nil
}

func (m *cewClientMock) GetJob(_ context.Context, jID string) (external_models.JobLibJob, error) {
	if m.GetJobErr != nil {
		return external_models.JobLibJob{}, m.GetJobErr
	}
	job, ok := m.Jobs[jID]
	if !ok {
		return external_models.JobLibJob{}, errors.New("not found")
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
