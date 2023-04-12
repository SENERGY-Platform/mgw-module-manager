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

package api

import (
	"context"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-lib/tsort"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
)

type Api struct {
	moduleHandler     handler.ModuleHandler
	deploymentHandler handler.DeploymentHandler
}

func New(moduleHandler handler.ModuleHandler, deploymentHandler handler.DeploymentHandler) *Api {
	return &Api{moduleHandler: moduleHandler, deploymentHandler: deploymentHandler}
}

func getOrder(modules map[string]*module.Module) ([]string, error) {
	nodes := make(tsort.Nodes)
	for id, m := range modules {
		if len(m.Dependencies) > 0 {
			reqIDs := make(map[string]struct{})
			for i := range m.Dependencies {
				reqIDs[i] = struct{}{}
			}
			nodes.Add(id, reqIDs, nil)
		}
	}
	order, err := tsort.GetTopOrder(nodes)
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	return order, nil
}

func (a *Api) getReqModules(ctx context.Context, m *module.Module, modules map[string]*module.Module) error {
	for id := range m.Dependencies {
		if _, ok := modules[id]; !ok {
			dm, err := a.moduleHandler.Get(ctx, id)
			if err != nil {
				return err
			}
			modules[id] = dm
			err = a.getReqModules(ctx, dm, modules)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
