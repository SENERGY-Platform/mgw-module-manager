/*
 * Copyright 2023 InfAI (CC SES)
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

package mod_staging_hdl

import (
	"context"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/mgw-container-engine-wrapper/client"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-lib/tsort"
	"github.com/SENERGY-Platform/mgw-module-lib/validation"
	"github.com/SENERGY-Platform/mgw-module-lib/validation/sem_ver"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"github.com/SENERGY-Platform/mgw-module-manager/util/cew_job"
	"github.com/SENERGY-Platform/mgw-module-manager/util/dir_fs"
	"io/fs"
	"os"
	"path"
	"time"
)

type Handler struct {
	wrkSpcPath              string
	transferHandler         handler.ModTransferHandler
	modFileHandler          handler.ModFileHandler
	configValidationHandler handler.CfgValidationHandler
	cewClient               client.CewClient
	httpTimeout             time.Duration
}

func New(workspacePath string, transferHandler handler.ModTransferHandler, modFileHandler handler.ModFileHandler, configValidationHandler handler.CfgValidationHandler, cewClient client.CewClient, httpTimeout time.Duration) *Handler {
	return &Handler{
		wrkSpcPath:              workspacePath,
		transferHandler:         transferHandler,
		modFileHandler:          modFileHandler,
		configValidationHandler: configValidationHandler,
		cewClient:               cewClient,
		httpTimeout:             httpTimeout,
	}
}

func (h *Handler) InitWorkspace(perm fs.FileMode) error {
	if !path.IsAbs(h.wrkSpcPath) {
		return fmt.Errorf("workspace path must be absolute")
	}
	if err := os.MkdirAll(h.wrkSpcPath, perm); err != nil {
		return err
	}
	return nil
}

func (h *Handler) Prepare(ctx context.Context, modules map[string]*module.Module, mID, ver string, dependencies bool) (handler.Stage, error) {
	stgPth, err := os.MkdirTemp(h.wrkSpcPath, "stg_")
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	stg := &stage{
		modules:     make(map[string]*module.Module),
		items:       make(map[string]modExtra),
		path:        stgPth,
		cewClient:   h.cewClient,
		httpTimeout: h.httpTimeout,
	}
	err = h.getStageItems(ctx, stg, mID, ver, "", stgPth, "", false, dependencies)
	if err != nil {
		stg.Remove()
		return nil, err
	}
	nodes := make(tsort.Nodes)
	for _, mod := range stg.modules {
		if len(mod.Dependencies) > 0 {
			reqIDs := make(map[string]struct{})
			for i := range mod.Dependencies {
				reqIDs[i] = struct{}{}
			}
			nodes.Add(mod.ID, reqIDs, nil)
		}
	}
	_, err = tsort.GetTopOrder(nodes)
	if err != nil {
		stg.Remove()
		return nil, err
	}
	return stg, nil
}

func (h *Handler) getStageItems(ctx context.Context, stg *stage, mID, ver, verRng, stgPath, reqBy string, indirect bool, dependencies bool) error {
	i, ok := stg.items[mID]
	if !ok {
		modRepo, err := h.transferHandler.Get(ctx, mID)
		if err != nil {
			return err
		}
		defer modRepo.Remove()
		if ver == "" {
			var err error
			ver, err = getVersion(modRepo.Versions(), verRng)
			if err != nil {
				return err
			}
		}
		dir, err := modRepo.Get(ver)
		if err != nil {
			return err
		}
		modFile, modFileName, err := h.modFileHandler.GetModFile(dir)
		if err != nil {
			return err
		}
		mod, err := h.modFileHandler.GetModule(modFile)
		if err != nil {
			return err
		}
		if err = h.validateModule(mod, mID, ver); err != nil {
			return err
		}
		if indirect && mod.DeploymentType == module.MultipleDeployment {
			return fmt.Errorf("dependencies with deployment type '%s' not supported", module.MultipleDeployment)
		}
		for _, srv := range mod.Services {
			err = h.addImage(ctx, srv.Image)
			if err != nil {
				return err
			}
		}
		modPth, err := os.MkdirTemp(stgPath, "mod_")
		if err != nil {
			return err
		}
		err = util.CopyDir(dir.Path(), modPth)
		if err != nil {
			return err
		}
		modDir, err := dir_fs.New(modPth)
		if err != nil {
			return err
		}
		stg.items[mID] = &item{
			module:   mod,
			modFile:  modFileName,
			dir:      modDir,
			indirect: indirect,
		}
		if dependencies {
			for rmID, rmVerRng := range mod.Dependencies {
				err = h.getStageItems(ctx, stg, rmID, "", rmVerRng, stgPath, mID, true, dependencies)
				if err != nil {
					return err
				}
			}
		}
	} else {
		mod := i.Module()
		if verRng != "" {
			ok, err := sem_ver.InSemVerRange(verRng, mod.Version)
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("module '%s' at '%s' but '%s' requires '%s'", mID, mod.Version, reqBy, verRng)
			}
		}
	}
	return nil
}

func (h *Handler) addImage(ctx context.Context, img string) error {
	ctxWt, cf := context.WithTimeout(ctx, h.httpTimeout)
	defer cf()
	jID, err := h.cewClient.AddImage(ctxWt, img)
	if err != nil {
		return model.NewInternalError(err)
	}
	job, err := cew_job.Await(ctx, h.cewClient, jID, h.httpTimeout)
	if err != nil {
		return err
	}
	if job.Error != nil {
		return model.NewInternalError(fmt.Errorf("%v", job.Error))
	}
	return nil
}

func getVersion(versions []string, verRng string) (string, error) {
	var ver string
	for i := len(versions) - 1; i >= 0; i-- {
		v := versions[i]
		if verRng != "" {
			ok, _ := sem_ver.InSemVerRange(verRng, v)
			if ok {
				ver = v
				break
			}
		} else {
			if sem_ver.IsValidSemVer(v) {
				ver = v
				break
			}
		}
	}
	if ver == "" {
		return "", model.NewInternalError(errors.New("no versions available"))
	}
	return ver, nil
}

func (h *Handler) validateModule(m *module.Module, mID, ver string) error {
	if mID != m.ID {
		return fmt.Errorf("module ID mismatch: %s != %s", mID, m.ID)
	}
	if ver != m.Version {
		return fmt.Errorf("version mismatch: %s != %s", ver, m.Version)
	}
	err := validation.Validate(m)
	if err != nil {
		return err
	}
	for _, cv := range m.Configs {
		if err = h.configValidationHandler.ValidateBase(cv.Type, cv.TypeOpt, cv.DataType); err != nil {
			return err
		}
		if err = h.configValidationHandler.ValidateTypeOptions(cv.Type, cv.TypeOpt); err != nil {
			return err
		}
		if cv.Default != nil {
			if err = h.configValidationHandler.ValidateValue(cv.Type, cv.TypeOpt, cv.Default, cv.IsSlice, cv.DataType); err != nil {
				return err
			}
		}
	}
	return nil
}
