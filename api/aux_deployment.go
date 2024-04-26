/*
 * Copyright 2024 InfAI (CC SES)
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

package api

import (
	"context"
	"errors"
	"fmt"
	job_hdl_lib "github.com/SENERGY-Platform/go-service-base/job-hdl/lib"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
)

func (a *Api) GetAuxDeployments(ctx context.Context, dID string, filter lib_model.AuxDepFilter, assets, containerInfo bool) (map[string]lib_model.AuxDeployment, error) {
	auxDeps, err := a.auxDeploymentHandler.List(ctx, dID, filter, assets, containerInfo)
	if err != nil {
		return nil, newApiErr(fmt.Sprintf("get aux deployments (%s assets=%v container_info=%v)", getAuxDepFilterValues(filter), assets, containerInfo), err)
	}
	return auxDeps, nil
}

func (a *Api) GetAuxDeployment(ctx context.Context, dID, aID string, assets, containerInfo bool) (lib_model.AuxDeployment, error) {
	metaStr := fmt.Sprintf("get aux deployment (assets=%v container_info=%v)", assets, containerInfo)
	auxDep, err := a.auxDeploymentHandler.Get(ctx, dID, aID, assets, containerInfo)
	if err != nil {
		return lib_model.AuxDeployment{}, newApiErr(metaStr, err)
	}
	return auxDep, nil
}

func (a *Api) CreateAuxDeployment(ctx context.Context, dID string, auxDepInput lib_model.AuxDepReq, forcePullImg bool) (string, error) {
	metaStr := fmt.Sprintf("create aux deployment (deployment_id=%s ref=%s image=%s force_pull_image=%v)", dID, auxDepInput.Ref, auxDepInput.Image, forcePullImg)
	dep, err := a.deploymentHandler.Get(ctx, dID, true, true, true, false)
	if err != nil {
		return "", newApiErr(metaStr, err)
	}
	mod, err := a.moduleHandler.Get(ctx, dep.Module.ID, false)
	if err != nil {
		return "", newApiErr(metaStr, err)
	}
	var requiredDep map[string]lib_model.Deployment
	if len(dep.RequiredDep) > 0 {
		requiredDep, err = a.deploymentHandler.List(ctx, lib_model.DepFilter{IDs: dep.RequiredDep}, false, false, true, false)
		if err != nil {
			return "", newApiErr(metaStr, err)
		}
	}
	jID, err := a.jobHandler.Create(ctx, metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer cf()
		aID, err := a.auxDeploymentHandler.Create(ctx, mod.Module.Module, dep, requiredDep, auxDepInput, forcePullImg)
		if err == nil {
			err = ctx.Err()
		}
		if err != nil {
			return nil, err
		}
		return aID, err
	})
	if err != nil {
		return "", newApiErr(metaStr, err)
	}
	a.auxJobHandler.Add(dID, jID)
	return jID, nil
}

func (a *Api) UpdateAuxDeployment(ctx context.Context, dID, aID string, auxDepInput lib_model.AuxDepReq, incremental, forcePullImg bool) (string, error) {
	metaStr := fmt.Sprintf("update aux deployment (deployment_id=%s aux_deployment_id=%s ref=%s, image=%s force_pull_image=%v)", dID, aID, auxDepInput.Ref, auxDepInput.Image, forcePullImg)
	dep, err := a.deploymentHandler.Get(ctx, dID, true, true, true, false)
	if err != nil {
		return "", newApiErr(metaStr, err)
	}
	mod, err := a.moduleHandler.Get(ctx, dep.Module.ID, false)
	if err != nil {
		return "", newApiErr(metaStr, err)
	}
	var requiredDep map[string]lib_model.Deployment
	if len(dep.RequiredDep) > 0 {
		requiredDep, err = a.deploymentHandler.List(ctx, lib_model.DepFilter{IDs: dep.RequiredDep}, false, false, true, false)
		if err != nil {
			return "", newApiErr(metaStr, err)
		}
	}
	jID, err := a.jobHandler.Create(ctx, metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer cf()
		err := a.auxDeploymentHandler.Update(ctx, aID, mod.Module.Module, dep, requiredDep, auxDepInput, forcePullImg, incremental)
		if err == nil {
			err = ctx.Err()
		}
		return nil, err
	})
	if err != nil {
		return "", newApiErr(metaStr, err)
	}
	a.auxJobHandler.Add(dID, jID)
	return jID, nil
}

func (a *Api) DeleteAuxDeployment(ctx context.Context, dID, aID string, force bool) (string, error) {
	metaStr := fmt.Sprintf("delete aux deployment (deployment_id=%s aux_deployment_id=%v force=%v)", dID, aID, force)
	jID, err := a.jobHandler.Create(ctx, metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer cf()
		err := a.auxDeploymentHandler.Delete(ctx, dID, aID, force)
		if err == nil {
			err = ctx.Err()
		}
		return nil, err
	})
	if err != nil {
		return "", newApiErr(metaStr, err)
	}
	a.auxJobHandler.Add(dID, jID)
	return jID, nil
}

func (a *Api) DeleteAuxDeployments(ctx context.Context, dID string, filter lib_model.AuxDepFilter, force bool) (string, error) {
	metaStr := fmt.Sprintf("delete aux deployments (deployment_id=%s %s force=%v)", dID, getAuxDepFilterValues(filter), force)
	jID, err := a.jobHandler.Create(ctx, metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer cf()
		deleted, err := a.auxDeploymentHandler.DeleteAll(ctx, dID, filter, force)
		if err == nil {
			err = ctx.Err()
		}
		return deleted, err
	})
	if err != nil {
		return "", newApiErr(metaStr, err)
	}
	a.auxJobHandler.Add(dID, jID)
	return jID, nil
}

func (a *Api) StartAuxDeployment(ctx context.Context, dID, aID string) (string, error) {
	metaStr := fmt.Sprintf("start aux deployment (deployment_id=%s aux_deployment_id=%v)", dID, aID)
	jID, err := a.jobHandler.Create(ctx, metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer cf()
		err := a.auxDeploymentHandler.Start(ctx, dID, aID)
		if err == nil {
			err = ctx.Err()
		}
		return nil, err
	})
	if err != nil {
		return "", newApiErr(metaStr, err)
	}
	a.auxJobHandler.Add(dID, jID)
	return jID, nil
}

func (a *Api) StartAuxDeployments(ctx context.Context, dID string, filter lib_model.AuxDepFilter) (string, error) {
	metaStr := fmt.Sprintf("start aux deployments (deployment_id=%s %s)", dID, getAuxDepFilterValues(filter))
	jID, err := a.jobHandler.Create(ctx, metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer cf()
		started, err := a.auxDeploymentHandler.StartAll(ctx, dID, filter)
		if err == nil {
			err = ctx.Err()
		}
		return started, err
	})
	if err != nil {
		return "", newApiErr(metaStr, err)
	}
	a.auxJobHandler.Add(dID, jID)
	return jID, nil
}

func (a *Api) StopAuxDeployment(ctx context.Context, dID, aID string) (string, error) {
	metaStr := fmt.Sprintf("stop aux deployment (deployment_id=%s aux_deployment_id=%v)", dID, aID)
	jID, err := a.jobHandler.Create(ctx, metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer cf()
		err := a.auxDeploymentHandler.Stop(ctx, dID, aID, false)
		if err == nil {
			err = ctx.Err()
		}
		return nil, err
	})
	if err != nil {
		return "", newApiErr(metaStr, err)
	}
	a.auxJobHandler.Add(dID, jID)
	return jID, nil
}

func (a *Api) StopAuxDeployments(ctx context.Context, dID string, filter lib_model.AuxDepFilter) (string, error) {
	metaStr := fmt.Sprintf("stop aux deployments (deployment_id=%s %s)", dID, getAuxDepFilterValues(filter))
	jID, err := a.jobHandler.Create(ctx, metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer cf()
		stopped, err := a.auxDeploymentHandler.StopAll(ctx, dID, filter, false)
		if err == nil {
			err = ctx.Err()
		}
		return stopped, err
	})
	if err != nil {
		return "", newApiErr(metaStr, err)
	}
	a.auxJobHandler.Add(dID, jID)
	return jID, nil
}

func (a *Api) RestartAuxDeployment(ctx context.Context, dID, aID string) (string, error) {
	metaStr := fmt.Sprintf("restart aux deployment (deployment_id=%s aux_deployment_id=%v)", dID, aID)
	jID, err := a.jobHandler.Create(ctx, metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer cf()
		err := a.auxDeploymentHandler.Restart(ctx, dID, aID)
		if err == nil {
			err = ctx.Err()
		}
		return nil, err
	})
	if err != nil {
		return "", newApiErr(metaStr, err)
	}
	a.auxJobHandler.Add(dID, jID)
	return jID, nil
}

func (a *Api) RestartAuxDeployments(ctx context.Context, dID string, filter lib_model.AuxDepFilter) (string, error) {
	metaStr := fmt.Sprintf("restart aux deployments (deployment_id=%s %s)", dID, getAuxDepFilterValues(filter))
	jID, err := a.jobHandler.Create(ctx, metaStr, func(ctx context.Context, cf context.CancelFunc) (any, error) {
		defer cf()
		restarted, err := a.auxDeploymentHandler.RestartAll(ctx, dID, filter)
		if err == nil {
			err = ctx.Err()
		}
		return restarted, err
	})
	if err != nil {
		return "", newApiErr(metaStr, err)
	}
	a.auxJobHandler.Add(dID, jID)
	return jID, nil
}

func (a *Api) GetAuxJobs(ctx context.Context, dID string, filter job_hdl_lib.JobFilter) ([]job_hdl_lib.Job, error) {
	jobs, err := a.jobHandler.List(ctx, filter)
	if err != nil {
		return nil, newApiErr(fmt.Sprintf("get jobs (%s)", getJobFilterValues(filter)), err)
	}
	var jobs2 []job_hdl_lib.Job
	for _, job := range jobs {
		if a.auxJobHandler.Check(dID, job.ID) {
			jobs2 = append(jobs2, job)
		}
	}
	return jobs2, nil
}

func (a *Api) GetAuxJob(ctx context.Context, dID string, jID string) (job_hdl_lib.Job, error) {
	metaStr := fmt.Sprintf("get job (id=%v)", jID)
	if !a.auxJobHandler.Check(dID, jID) {
		return job_hdl_lib.Job{}, newApiErr(metaStr, lib_model.NewForbiddenError(errors.New("forbidden")))
	}
	job, err := a.jobHandler.Get(ctx, jID)
	if err != nil {
		return job_hdl_lib.Job{}, newApiErr(metaStr, err)
	}
	return job, nil
}

func (a *Api) CancelAuxJob(ctx context.Context, dID string, jID string) error {
	metaStr := fmt.Sprintf("cancel job (id=%v)", jID)
	if !a.auxJobHandler.Check(dID, jID) {
		return newApiErr(metaStr, lib_model.NewForbiddenError(errors.New("forbidden")))
	}
	if err := a.jobHandler.Cancel(ctx, jID); err != nil {
		return newApiErr(metaStr, err)
	}
	return nil
}

func (a *Api) updateAllAuxDeployments(ctx context.Context, dID string, mod *module.Module) ([]string, error) {
	dep, err := a.deploymentHandler.Get(ctx, dID, true, true, true, false)
	if err != nil {
		return nil, err
	}
	var requiredDep map[string]lib_model.Deployment
	if len(dep.RequiredDep) > 0 {
		requiredDep, err = a.deploymentHandler.List(ctx, lib_model.DepFilter{IDs: dep.RequiredDep}, false, false, true, false)
		if err != nil {
			return nil, err
		}
	}
	return a.auxDeploymentHandler.UpdateAll(ctx, mod, dep, requiredDep)
}

func getAuxDepFilterValues(filter lib_model.AuxDepFilter) string {
	return fmt.Sprintf("labels=%v image=%s", filter.Labels, filter.Image)
}
