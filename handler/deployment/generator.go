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

package deployment

import (
	"fmt"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"module-manager/model"
)

func genDeployment(m *module.Module, name *string, hostRes map[string]string, secrets map[string]string, configs map[string]any) (*model.Deployment, []string, []string, error) {
	dRs, rad, err := genDepHostRes(hostRes, m.HostResources)
	if err != nil {
		return nil, nil, nil, err
	}
	dSs, sad, err := genDepSecrets(secrets, m.Secrets)
	if err != nil {
		return nil, nil, nil, err
	}
	dCs, err := genDepConfigs(configs, m.Configs)
	if err != nil {
		return nil, nil, nil, err
	}
	d := model.Deployment{
		DepMeta: model.DepMeta{
			ModuleID: m.ID,
			Created:  time.Now().UTC(),
			Updated:  time.Now().UTC(),
		},
		HostResources: dRs,
		Secrets:       dSs,
		Configs:       dCs,
	}
	if name != nil {
		d.Name = *name
	}
	return &d, rad, sad, nil
}

func genDepHostRes(hrs map[string]string, mHRs map[string]module.HostResource) (map[string]string, []string, error) {
	dRs := make(map[string]string)
	var ad []string
	for ref, mRH := range mHRs {
		id, ok := hrs[ref]
		if ok {
			dRs[ref] = id
		} else {
			if mRH.Required {
				if len(mRH.Tags) > 0 {
					ad = append(ad, ref)
				} else {
					return nil, nil, fmt.Errorf("host resource '%s' required", ref)
				}
			}
		}
	}
	return dRs, ad, nil
}

func genDepSecrets(s map[string]string, mSs map[string]module.Secret) (map[string]string, []string, error) {
	dSs := make(map[string]string)
	var ad []string
	for ref, mS := range mSs {
		id, ok := s[ref]
		if ok {
			dSs[ref] = id
		} else {
			if mS.Required {
				if len(mS.Tags) > 0 {
					ad = append(ad, ref)
				} else {
					return nil, nil, fmt.Errorf("secret '%s' required", ref)
				}
			}
		}
	}
	return dSs, ad, nil
}

func genDepConfigs(cfgs map[string]any, mCs module.Configs) (map[string]any, error) {
	dCs := make(map[string]any)
	for ref, mC := range mCs {
		val, ok := cfgs[ref]
		if !ok {
			if mC.Default != nil {
				dCs[ref] = mC.Default
			} else {
				if mC.Required {
					return nil, fmt.Errorf("config '%s' requried", ref)
				}
			}
		} else {
			var v any
			var err error
			if mC.IsSlice {
				v, err = parseCfgValSlice(val, mC.DataType)
				dCs[ref] = v
			} else {
				v, err = parseCfgVal(val, mC.DataType)
			}
			if err != nil {
				return nil, fmt.Errorf("parsing config '%s' failed: %s", ref, err)
			}
			dCs[ref] = v
		}
	}
	return dCs, nil
}
