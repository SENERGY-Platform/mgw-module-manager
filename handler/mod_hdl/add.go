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

package mod_hdl

import (
	"context"
	"errors"
	"fmt"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-lib/validation/sem_ver"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"os"
	"sort"
)

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

func (h *Handler) add(ctx context.Context, mID, ver, verRng string, indirect bool) error {
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
	f, name, err := h.modFileHandler.GetModFile(dir)
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
		dm, err := h.storageHandler.Get(ctx, dmID)
		if err != nil {
			var nfe *model.NotFoundError
			if !errors.As(err, &nfe) {
				return err
			}
			err = h.add(ctx, dmID, "", dmVerRng, true)
			if err != nil {
				return err
			}
			continue
		}
		if dm.DeploymentType == module.MultipleDeployment {
			return model.NewInternalError(fmt.Errorf("dependencies with deployment type '%s' not supported", module.MultipleDeployment))
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
	err = h.storageHandler.Add(ctx, dir, mID, name, indirect)
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
		return model.NewInternalError(err)
	}
	job, err := h.cewJobHandler.AwaitJob(ctx, jID)
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
