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
	"sort"
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

func (h *Handler) Prepare(ctx context.Context, modules map[string]*module.Module, mID, ver string, updateReq bool) (handler.Stage, error) {
	stgPth, err := os.MkdirTemp(h.wrkSpcPath, "stg_")
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	stg := &stage{
		items:       make(map[string]handler.StageItem),
		path:        stgPth,
		cewClient:   h.cewClient,
		httpTimeout: h.httpTimeout,
	}
	err = h.add(ctx, modules, stg, mID, ver, "", false, updateReq)
	if err != nil {
		stg.Remove()
		return nil, err
	}
	return stg, nil
}

func (h *Handler) add(ctx context.Context, modules map[string]*module.Module, stg *stage, mID, ver, verRng string, indirect, updateReq bool) error {
	if ver == "" {
		var err error
		ver, err = h.getVersion(ctx, mID, verRng)
		if err != nil {
			return err
		}
	} else {
		if !sem_ver.IsValidSemVer(ver) {
			return model.NewInvalidInputError(fmt.Errorf("version '%s' invalid", ver))
		}
	}
	trfDir, err := h.transferHandler.Get(ctx, mID, ver)
	if err != nil {
		return err
	}
	defer os.RemoveAll(trfDir.Path())
	f, name, err := h.modFileHandler.GetModFile(trfDir)
	if err != nil {
		return err
	}
	m, err := h.modFileHandler.GetModule(f)
	if err != nil {
		return err
	}
	if err = h.validateModule(m, mID, ver); err != nil {
		return err
	}
	if indirect && m.DeploymentType == module.MultipleDeployment {
		return model.NewInternalError(fmt.Errorf("dependencies with deployment type '%s' not supported", module.MultipleDeployment))
	}
	for dmID, dmVerRng := range m.Dependencies {
		dm, ok := modules[dmID]
		if !ok {
			err = h.add(ctx, modules, stg, dmID, "", dmVerRng, true, updateReq)
			if err != nil {
				return err
			}
			continue
		}
		if dm.DeploymentType == module.MultipleDeployment {
			return model.NewInternalError(fmt.Errorf("dependencies with deployment type '%s' not supported", module.MultipleDeployment))
		}
		if updateReq {

		}
		ok, err := sem_ver.InSemVerRange(dmVerRng, dm.Version)
		if err != nil {
			return model.NewInternalError(err)
		}
		if !ok {
			return fmt.Errorf("'%s' of '%s' does not satsify '%s'", dm.Version, dm.ID, dmVerRng)
		}
	}
	for _, srv := range m.Services {
		err = h.addImage(ctx, srv.Image)
		if err != nil {
			return err
		}
	}
	modPth, err := os.MkdirTemp(stg.path, "mod_")
	if err != nil {
		return err
	}
	err = util.CopyDir(trfDir.Path(), modPth)
	if err != nil {
		return err
	}
	modDir, err := dir_fs.New(modPth)
	if err != nil {
		return err
	}
	stg.items[m.ID] = item{
		module:   m,
		modFile:  name,
		dir:      modDir,
		indirect: indirect,
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

func (h *Handler) getVersion(ctx context.Context, mID, verRng string) (string, error) {
	verList, err := h.transferHandler.ListVersions(ctx, mID)
	if err != nil {
		return "", err
	}
	sort.Strings(verList)
	var ver string
	for i := len(verList) - 1; i >= 0; i-- {
		v := verList[i]
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
