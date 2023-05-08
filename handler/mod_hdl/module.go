/*
 * Copyright 2022 InfAI (CC SES)
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
	"github.com/SENERGY-Platform/mgw-container-engine-wrapper/client"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-lib/validation"
	"github.com/SENERGY-Platform/mgw-module-lib/validation/sem_ver"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"io/fs"
	"os"
	"sort"
	"time"
)

type Handler struct {
	storageHandler          handler.ModStorageHandler
	transferHandler         handler.ModTransferHandler
	modFileHandler          handler.ModFileHandler
	configValidationHandler handler.CfgValidationHandler
	cewJobHandler           handler.CewJobHandler
	cewClient               client.CewClient
	httpTimeout             time.Duration
}

func New(storageHandler handler.ModStorageHandler, transferHandler handler.ModTransferHandler, modFileHandler handler.ModFileHandler, configValidationHandler handler.CfgValidationHandler, cewJobHandler handler.CewJobHandler, cewClient client.CewClient, httpTimeout time.Duration) *Handler {
	return &Handler{
		storageHandler:          storageHandler,
		transferHandler:         transferHandler,
		modFileHandler:          modFileHandler,
		configValidationHandler: configValidationHandler,
		cewJobHandler:           cewJobHandler,
		cewClient:               cewClient,
		httpTimeout:             httpTimeout,
	}
}

func (h *Handler) List(ctx context.Context, filter model.ModFilter) ([]model.ModuleMeta, error) {
	return h.storageHandler.List(ctx, filter)
}

func (h *Handler) Get(ctx context.Context, mID string) (model.Module, error) {
	return h.storageHandler.Get(ctx, mID)
}

func (h *Handler) GetReq(ctx context.Context, mID string) (model.Module, map[string]*module.Module, error) {
	m, err := h.storageHandler.Get(ctx, mID)
	if err != nil {
		return model.Module{}, nil, err
	}
	dep := make(map[string]*module.Module)
	if err := h.getReqMod(ctx, m.Module, dep); err != nil {
		return model.Module{}, nil, err
	}
	return m, dep, nil
}

func (h *Handler) GetIncl(ctx context.Context, mID string) (util.DirFS, error) {
	mod, dir, err := h.storageHandler.GetDir(ctx, mID)
	if err != nil {
		return "", err
	}
	for _, srv := range mod.Services {
		for _, bindMount := range srv.BindMounts {
			_, err = fs.Stat(dir, bindMount.Source)
			if err != nil {
				return "", err
			}
		}
	}
	return dir, nil
}

func (h *Handler) Add(ctx context.Context, mr model.ModRequest) error {
	m, err := h.storageHandler.Get(ctx, mr.ID)
	if err != nil {
		var nfe *model.NotFoundError
		if !errors.As(err, &nfe) {
			return err
		}
	}
	if m.Module != nil {
		return model.NewInternalError(errors.New("already installed"))
	}
	return h.add(ctx, mr.ID, mr.Version, "", false)
}

func (h *Handler) add(ctx context.Context, mID, ver, verRng string) error {
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
	dir, err := h.transferHandler.Get(ctx, mID, ver)
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir.Path())
	m, err := h.modFileHandler.GetModule(dir)
	if err != nil {
		return err
	}
	if err = h.validateModule(m, mID); err != nil {
		return err
	}
	for dmID, dmVerRng := range m.Dependencies {
		dm, err := h.storageHandler.Get(ctx, dmID)
		if err != nil {
			var nfe *model.NotFoundError
			if !errors.As(err, &nfe) {
				return err
			}
			err = h.add(ctx, dmID, "", dmVerRng)
			if err != nil {
				return err
			}
			continue
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
	err = h.storageHandler.Add(ctx, dir, mID)
	if err != nil {
		return err
	}
	return nil
}

func (h *Handler) addImage(ctx context.Context, img string) error {
	ctxWt, cf := context.WithTimeout(ctx, h.httpTimeout)
	defer cf()
	jID, err := h.cewClient.AddImage(ctxWt, img)
	if err != nil {
		return err
	}
	job, err := h.cewJobHandler.AwaitJob(ctx, jID)
	if err != nil {
		return err
	}
	if job.Error != nil {
		return fmt.Errorf("%v", job.Error)
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

func (h *Handler) Delete(ctx context.Context, mID string) error {
	return h.storageHandler.Delete(ctx, mID)
}

func (h *Handler) Update(ctx context.Context, mID string) error {
	panic("not implemented")
}

func (h *Handler) getReqMod(ctx context.Context, mod *module.Module, reqMod map[string]*module.Module) error {
	for mID := range mod.Dependencies {
		if _, ok := reqMod[mID]; !ok {
			m, err := h.storageHandler.Get(ctx, mID)
			if err != nil {
				return err
			}
			reqMod[mID] = m.Module
			if err = h.getReqMod(ctx, m.Module, reqMod); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *Handler) validateModule(m *module.Module, mID string) error {
	if mID != m.ID {
		return fmt.Errorf("module ID missmatch: %s != %s", mID, m.ID)
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
