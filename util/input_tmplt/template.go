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

package input_tmplt

import (
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
)

func GetModDepTemplate(mod *module.Module) model.InputTemplate {
	it := model.InputTemplate{
		HostResources: make(map[string]model.InputTemplateHostRes),
		Secrets:       make(map[string]model.InputTemplateSecret),
		Configs:       make(map[string]model.InputTemplateConfig),
		InputGroups:   mod.Inputs.Groups,
	}
	for ref, input := range mod.Inputs.Resources {
		it.HostResources[ref] = model.InputTemplateHostRes{
			Input:        input,
			HostResource: mod.HostResources[ref],
		}
	}
	for ref, input := range mod.Inputs.Secrets {
		it.Secrets[ref] = model.InputTemplateSecret{
			Input:  input,
			Secret: mod.Secrets[ref],
		}
	}
	for ref, input := range mod.Inputs.Configs {
		cv := mod.Configs[ref]
		itc := model.InputTemplateConfig{
			Input:    input,
			Default:  cv.Default,
			Options:  cv.Options,
			OptExt:   cv.OptExt,
			Type:     cv.Type,
			TypeOpt:  make(map[string]any),
			DataType: cv.DataType,
			IsList:   cv.IsSlice,
			Required: cv.Required,
		}
		for key, opt := range cv.TypeOpt {
			itc.TypeOpt[key] = opt.Value
		}
		it.Configs[ref] = itc
	}
	return it
}

func GetDepUpTemplate(mod *module.Module, dep *model.Deployment) model.InputTemplate {
	it := model.InputTemplate{
		HostResources: make(map[string]model.InputTemplateHostRes),
		Secrets:       make(map[string]model.InputTemplateSecret),
		Configs:       make(map[string]model.InputTemplateConfig),
		InputGroups:   mod.Inputs.Groups,
	}
	for ref, input := range mod.Inputs.Resources {
		hr := model.InputTemplateHostRes{
			Input:        input,
			HostResource: mod.HostResources[ref],
		}
		if hrID, ok := dep.HostResources[ref]; ok {
			hr.Value = hrID
		}
		it.HostResources[ref] = hr
	}
	for ref, input := range mod.Inputs.Secrets {
		s := model.InputTemplateSecret{
			Input:  input,
			Secret: mod.Secrets[ref],
		}
		if sID, ok := dep.Secrets[ref]; ok {
			s.Value = sID
		}
		it.Secrets[ref] = s
	}
	for ref, input := range mod.Inputs.Configs {
		cv := mod.Configs[ref]
		itc := model.InputTemplateConfig{
			Input:    input,
			Default:  cv.Default,
			Options:  cv.Options,
			OptExt:   cv.OptExt,
			Type:     cv.Type,
			TypeOpt:  make(map[string]any),
			DataType: cv.DataType,
			IsList:   cv.IsSlice,
			Required: cv.Required,
		}
		for key, opt := range cv.TypeOpt {
			itc.TypeOpt[key] = opt.Value
		}
		if dc, ok := dep.Configs[ref]; ok {
			itc.Value = dc.Value
		}
		it.Configs[ref] = itc
	}
	return it
}
