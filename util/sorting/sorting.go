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

package sorting

import (
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-lib/tsort"
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
)

func GetModOrder(modules map[string]*module.Module) (order []string, err error) {
	if len(modules) > 1 {
		nodes := make(tsort.Nodes)
		for _, m := range modules {
			var reqIDs map[string]struct{}
			if len(m.Dependencies) > 0 {
				reqIDs = make(map[string]struct{})
				for i := range m.Dependencies {
					reqIDs[i] = struct{}{}
				}
			}
			nodes.Add(m.ID, reqIDs, nil)
		}
		order, err = tsort.GetTopOrder(nodes)
		if err != nil {
			return nil, err
		}
	} else if len(modules) > 0 {
		for _, m := range modules {
			order = append(order, m.ID)
		}
	}
	return
}

func GetDepOrder(dep map[string]lib_model.Deployment) (order []string, err error) {
	if len(dep) > 1 {
		nodes := make(tsort.Nodes)
		for _, d := range dep {
			var reqIDs map[string]struct{}
			if len(d.RequiredDep) > 0 {
				reqIDs = make(map[string]struct{})
				for _, i := range d.RequiredDep {
					reqIDs[i] = struct{}{}
				}

			}
			nodes.Add(d.ID, reqIDs, nil)
		}
		order, err = tsort.GetTopOrder(nodes)
		if err != nil {
			return
		}
	} else if len(dep) > 0 {
		for _, d := range dep {
			order = append(order, d.ID)
		}
	}
	return
}

func GetSrvOrder(services map[string]*module.Service) (order []string, err error) {
	if len(services) > 1 {
		nodes := make(tsort.Nodes)
		for ref, srv := range services {
			nodes.Add(ref, srv.RequiredSrv, srv.RequiredBySrv)
		}
		order, err = tsort.GetTopOrder(nodes)
		if err != nil {
			return nil, err
		}
	} else if len(services) > 0 {
		for ref := range services {
			order = append(order, ref)
		}
	}
	return
}
