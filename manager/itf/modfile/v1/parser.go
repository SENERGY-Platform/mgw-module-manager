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

package v1

import (
	"fmt"
	"module-manager/manager/itf/module"
	"strconv"
	"strings"
)

func (mf Module) Parse() (module.Module, error) {
	m := module.Module{
		ID:             mf.ID,
		Name:           mf.Name,
		Description:    mf.Description,
		License:        mf.License,
		Author:         mf.Author,
		Version:        mf.Version,
		Type:           mf.Type,
		DeploymentType: mf.DeploymentType,
	}
	services, err := parseModuleServices(mf.Services)
	if err != nil {
		return m, err
	}
	volumes, err := parseModuleVolumes(mf.Volumes, services)
	if err != nil {
		return m, err
	}
	dependencies, err := parseModuleDependencies(mf.Dependencies, services)
	if err != nil {
		return m, err
	}
	resources, rInputs, err := parseModuleResources(mf.Resources, services)
	if err != nil {
		return m, err
	}
	secrets, sInputs, err := parseModuleSecrets(mf.Secrets, services)
	if err != nil {
		return m, err
	}
	configs, cInputs, err := parseModuleConfigs(mf.Configs, services)
	if err != nil {
		return m, err
	}
	m.Services = services
	m.Volumes = volumes
	m.Dependencies = dependencies
	m.Resources = resources
	m.Secrets = secrets
	m.Configs = configs
	userInput := module.UserInput{}
	if mf.InputGroups != nil && len(mf.InputGroups) > 0 {
		userInput.Groups = make(map[string]module.InputGroup)
		for ref, mfInputGroup := range mf.InputGroups {
			userInput.Groups[ref] = module.InputGroup{
				Name:        mfInputGroup.Name,
				Description: mfInputGroup.Description,
				Group:       mfInputGroup.Group,
			}
		}
	}
	if rInputs != nil && len(rInputs) > 0 {
		userInput.Resources = rInputs
	}
	if sInputs != nil && len(sInputs) > 0 {
		userInput.Secrets = sInputs
	}
	if cInputs != nil && len(cInputs) > 0 {
		userInput.Configs = cInputs
	}
	m.UserInput = userInput
	return m, nil
}

func parseModuleServices(mfServices map[string]Service) (map[string]*module.Service, error) {
	services := make(map[string]*module.Service)
	for name, mfService := range mfServices {
		include, err := parseServiceInclude(mfService.Include)
		if err != nil {
			return services, fmt.Errorf("service '%s' invalid include: %s", name, err)
		}
		tmpfs, err := parseServiceTmpfs(mfService.Tmpfs)
		if err != nil {
			return services, fmt.Errorf("service '%s' invalid tmpfs: %s", name, err)
		}
		httpEndpoints, err := parseServiceHttpEndpoints(mfService.HttpEndpoints)
		if err != nil {
			return services, fmt.Errorf("service '%s' invalid http endpoint: %s", name, err)
		}
		portMappings, err := parseServicePortMappings(mfService.PortMappings)
		if err != nil {
			return services, fmt.Errorf("service '%s' invalid port mapping: %s", name, err)
		}
		dependencies, err := parseServiceDependencies(mfService.Dependencies)
		if err != nil {
			return services, fmt.Errorf("service '%s' invalid depency: %s", name, err)
		}
		services[name] = &module.Service{
			Name:          mfService.Name,
			Image:         mfService.Image,
			RunConfig:     parseServiceRunConfig(mfService.RunConfig),
			Include:       include,
			Tmpfs:         tmpfs,
			HttpEndpoints: httpEndpoints,
			Dependencies:  dependencies,
			PortMappings:  portMappings,
		}
	}
	return services, nil
}

func parseServiceRunConfig(mfRunConfig RunConfig) module.RunConfig {
	rc := module.RunConfig{
		RestartStrategy: mfRunConfig.RestartStrategy,
		Retries:         mfRunConfig.Retries,
		RemoveAfterRun:  mfRunConfig.RemoveAfterRun,
		StopSignal:      mfRunConfig.StopSignal,
		PseudoTTY:       mfRunConfig.PseudoTTY,
	}
	if mfRunConfig.StopTimeout != nil {
		rc.StopTimeout = &mfRunConfig.StopTimeout.Duration
	}
	return rc
}

func parseServiceInclude(mfInclude []BindMount) (map[string]module.BindMount, error) {
	if mfInclude != nil && len(mfInclude) > 0 {
		include := make(map[string]module.BindMount)
		for _, mfBindMount := range mfInclude {
			if v, ok := include[mfBindMount.MountPoint]; ok {
				if v.Source == mfBindMount.Source && v.ReadOnly == mfBindMount.ReadOnly {
					continue
				}
				return nil, fmt.Errorf("duplicate '%s'", mfBindMount.MountPoint)
			}
			include[mfBindMount.MountPoint] = module.BindMount{
				Source:   mfBindMount.Source,
				ReadOnly: mfBindMount.ReadOnly,
			}
		}
		return include, nil
	}
	return nil, nil
}

func parseServiceTmpfs(mfTmpfs []TmpfsMount) (map[string]module.TmpfsMount, error) {
	if mfTmpfs != nil && len(mfTmpfs) > 0 {
		tmpfs := make(map[string]module.TmpfsMount)
		for _, mfTmpf := range mfTmpfs {
			if _, ok := tmpfs[mfTmpf.MountPoint]; ok {
				return nil, fmt.Errorf("duplicate '%s'", mfTmpf.MountPoint)
			}
			tm := module.TmpfsMount{Size: uint64(mfTmpf.Size)}
			if mfTmpf.Mode != nil {
				tm.Mode = &mfTmpf.Mode.FileMode
			}
			tmpfs[mfTmpf.MountPoint] = tm
		}
		return tmpfs, nil
	}
	return nil, nil
}

func parseServiceHttpEndpoints(mfHttpEndpoints []HttpEndpoint) (map[string]module.HttpEndpoint, error) {
	if mfHttpEndpoints != nil && len(mfHttpEndpoints) > 0 {
		httpEndpoints := make(map[string]module.HttpEndpoint)
		for _, mfHttpEndpoint := range mfHttpEndpoints {
			if v, ok := httpEndpoints[mfHttpEndpoint.Path]; ok {
				if v.Name == mfHttpEndpoint.Name && v.Port == mfHttpEndpoint.Port && v.GwPath == mfHttpEndpoint.GwPath {
					continue
				}
				return nil, fmt.Errorf("duplicate '%s'", mfHttpEndpoint.Path)
			}
			httpEndpoints[mfHttpEndpoint.Path] = module.HttpEndpoint{
				Name:   mfHttpEndpoint.Name,
				Port:   mfHttpEndpoint.Port,
				GwPath: mfHttpEndpoint.GwPath,
			}
		}
		return httpEndpoints, nil
	}
	return nil, nil
}

func parsePort(p Port) (sl []uint) {
	parts := strings.Split(string(p), "-")
	for i := 0; i < len(parts); i++ {
		n, _ := strconv.ParseInt(parts[i], 10, 64)
		sl = append(sl, uint(n))
	}
	return
}

func parseServicePortMappings(mfPortMappings []PortMapping) (module.PortMappings, error) {
	if mfPortMappings != nil && len(mfPortMappings) > 0 {
		mappings := make(module.PortMappings)
		for _, mfPortMapping := range mfPortMappings {
			var hp []uint
			if mfPortMapping.HostPort != nil {
				hp = parsePort(*mfPortMapping.HostPort)
			}
			err := mappings.Add(mfPortMapping.Name, parsePort(mfPortMapping.Port), hp, mfPortMapping.Protocol)
			if err != nil {
				return mappings, err
			}
		}
		return mappings, nil
	}
	return nil, nil
}

func parseServiceDependencies(mfServiceDependencies map[string]ServiceDependencyTarget) (map[string]module.ServiceDependencyTarget, error) {
	if mfServiceDependencies != nil && len(mfServiceDependencies) > 0 {
		serviceDependencies := make(map[string]module.ServiceDependencyTarget)
		for srv, mfTarget := range mfServiceDependencies {
			if _, ok := serviceDependencies[mfTarget.RefVar]; ok {
				return serviceDependencies, fmt.Errorf("duplicate '%s'", mfTarget.RefVar)
			}
			serviceDependencies[mfTarget.RefVar] = module.ServiceDependencyTarget{
				Service:   srv,
				Condition: mfTarget.Condition,
			}
		}
		return serviceDependencies, nil
	}
	return nil, nil
}

func parseModuleVolumes(mfVolumes map[string][]VolumeTarget, services map[string]*module.Service) (module.Set[string], error) {
	if mfVolumes != nil && len(mfVolumes) > 0 {
		volumes := make(module.Set[string])
		for name, mfTargets := range mfVolumes {
			volumes[name] = struct{}{}
			for _, mfTarget := range mfTargets {
				for _, srv := range mfTarget.Services {
					if v, ok := services[srv]; ok {
						if v.Volumes == nil {
							v.Volumes = make(map[string]string)
						}
						if n, k := v.Volumes[mfTarget.MountPoint]; k {
							if n == name {
								continue
							}
							return volumes, fmt.Errorf("service '%s' invalid volume: duplicate '%s'", srv, mfTarget.MountPoint)
						}
						v.Volumes[mfTarget.MountPoint] = name
					}
				}
			}
		}
		return volumes, nil
	}
	return nil, nil
}

func parseModuleDependencies(mfModuleDependencies map[string]ModuleDependency, services map[string]*module.Service) (map[string]module.ModuleDependency, error) {
	if mfModuleDependencies != nil && len(mfModuleDependencies) > 0 {
		moduleDependencies := make(map[string]module.ModuleDependency)
		for id, dependency := range mfModuleDependencies {
			rs := make(module.Set[string])
			for rqSrv, mfTargets := range dependency.RequiredServices {
				rs[rqSrv] = struct{}{}
				for _, mfTarget := range mfTargets {
					if mfTarget.Services != nil && len(mfTarget.Services) > 0 {
						for _, srv := range mfTarget.Services {
							if v, ok := services[srv]; ok {
								if v.ExternalDependencies == nil {
									v.ExternalDependencies = make(map[string]module.ExternalDependencyTarget)
								}
								if ep, k := v.ExternalDependencies[mfTarget.RefVar]; k {
									if ep.ID == id && ep.Service == rqSrv {
										continue
									}
									return moduleDependencies, fmt.Errorf("service '%s' invalid module dependency: duplicate '%s'", srv, mfTarget.RefVar)
								}
								v.ExternalDependencies[mfTarget.RefVar] = module.ExternalDependencyTarget{
									ID:      id,
									Service: rqSrv,
								}
							}
						}
					}
				}
			}
			moduleDependencies[id] = module.ModuleDependency{
				Version:          dependency.Version,
				RequiredServices: rs,
			}
		}
		return moduleDependencies, nil
	}
	return nil, nil
}

func parseModuleResources(mfResources map[string]Resource, services map[string]*module.Service) (map[string]module.Resource, map[string]module.Input, error) {
	if mfResources != nil && len(mfResources) > 0 {
		resources := make(map[string]module.Resource)
		inputs := make(map[string]module.Input)
		for ref, mfResource := range mfResources {
			if mfResource.Targets != nil && len(mfResource.Targets) > 0 {
				for _, mfTarget := range mfResource.Targets {
					if mfTarget.Services != nil && len(mfTarget.Services) > 0 {
						for _, srv := range mfTarget.Services {
							if v, ok := services[srv]; ok {
								if v.Resources == nil {
									v.Resources = make(map[string]module.ResourceTarget)
								}
								if rt, k := v.Resources[mfTarget.MountPoint]; k {
									if rt.Ref == ref && rt.ReadOnly == mfTarget.ReadOnly {
										continue
									}
									return resources, inputs, fmt.Errorf("'%s' & '%s' -> '%s' -> '%s'", rt.Ref, ref, srv, mfTarget.MountPoint)
								}
								v.Resources[mfTarget.MountPoint] = module.ResourceTarget{
									Ref:      ref,
									ReadOnly: mfTarget.ReadOnly,
								}
							}
						}
					}
				}
			}
			r := module.Resource{Type: mfResource.Type}
			if mfResource.Tags != nil && len(mfResource.Tags) > 0 {
				r.Tags = make(module.Set[string])
				for _, tag := range mfResource.Tags {
					r.Tags[tag] = struct{}{}
				}
			}
			resources[ref] = r
			if mfResource.UserInput != nil {
				inputs[ref] = module.Input{
					Name:        mfResource.UserInput.Name,
					Description: mfResource.UserInput.Description,
					Required:    mfResource.UserInput.Required,
					Group:       mfResource.UserInput.Group,
				}
			}
		}
		return resources, inputs, nil
	}
	return nil, nil, nil
}

func parseModuleSecrets(mfSecrets map[string]Secret, services map[string]*module.Service) (map[string]module.Resource, map[string]module.Input, error) {
	if mfSecrets != nil && len(mfSecrets) > 0 {
		secrets := make(map[string]module.Resource)
		inputs := make(map[string]module.Input)
		for ref, mfSecret := range mfSecrets {
			if mfSecret.Targets != nil && len(mfSecret.Targets) > 0 {
				for _, mfTarget := range mfSecret.Targets {
					if mfTarget.Services != nil && len(mfTarget.Services) > 0 {
						for _, srv := range mfTarget.Services {
							if v, ok := services[srv]; ok {
								if v.Secrets == nil {
									v.Secrets = make(map[string]string)
								}
								if r, k := v.Secrets[mfTarget.MountPoint]; k {
									if r == ref {
										continue
									}
									return secrets, inputs, fmt.Errorf("'%s' & '%s' -> '%s' -> '%s'", r, ref, srv, mfTarget.MountPoint)
								}
								v.Secrets[mfTarget.MountPoint] = ref
							}
						}
					}
				}
			}
			r := module.Resource{Type: mfSecret.Type}
			if mfSecret.Tags != nil && len(mfSecret.Tags) > 0 {
				r.Tags = make(module.Set[string])
				for _, tag := range mfSecret.Tags {
					r.Tags[tag] = struct{}{}
				}
			}
			secrets[ref] = r
			if mfSecret.UserInput != nil {
				inputs[ref] = module.Input{
					Name:        mfSecret.UserInput.Name,
					Description: mfSecret.UserInput.Description,
					Required:    mfSecret.UserInput.Required,
					Group:       mfSecret.UserInput.Group,
					Type:        mfSecret.UserInput.Type,
					Constraints: mfSecret.UserInput.Constraints,
				}
			}
		}
		return secrets, inputs, nil
	}
	return nil, nil, nil
}

func parseModuleConfigs(mfConfigs map[string]ConfigValue, services map[string]*module.Service) (module.Configs, map[string]module.Input, error) {
	configs := make(module.Configs)
	if mfConfigs != nil && len(mfConfigs) > 0 {
		inputs := make(map[string]module.Input)
		for ref, mfConfig := range mfConfigs {
			if mfConfig.Targets != nil && len(mfConfig.Targets) > 0 {
				for _, mfTarget := range mfConfig.Targets {
					if mfTarget.Services != nil && len(mfTarget.Services) > 0 {
						for _, srv := range mfTarget.Services {
							if v, ok := services[srv]; ok {
								if v.Configs == nil {
									v.Configs = make(map[string]string)
								}
								if r, k := v.Configs[mfTarget.RefVar]; k {
									if r == ref {
										continue
									}
									return configs, inputs, fmt.Errorf("'%s' & '%s' -> '%s' -> '%s'", r, ref, srv, mfTarget.RefVar)
								}
								v.Configs[mfTarget.RefVar] = ref
							}
						}
					}
				}
			}
			switch mfConfig.Type {
			case module.TextData:
				var d *string
				var o []string
				if mfConfig.Value != nil {
					v, ok := mfConfig.Value.(string)
					if !ok {
						return configs, inputs, fmt.Errorf("%s type missmatch: string != %T", ref, mfConfig.Value)
					}
					d = &v
				}
				if mfConfig.Options != nil {
					for _, opt := range mfConfig.Options {
						v, ok := opt.(string)
						if !ok {
							return configs, inputs, fmt.Errorf("%s type missmatch: string != %T", ref, opt)
						}
						o = append(o, v)
					}
				}
				configs.SetString(ref, d, o...)
			case module.BoolData:
				var d *bool
				var o []bool
				if mfConfig.Value != nil {
					v, ok := mfConfig.Value.(bool)
					if !ok {
						return configs, inputs, fmt.Errorf("%s type missmatch: bool != %T", ref, mfConfig.Value)
					}
					d = &v
				}
				if mfConfig.Options != nil {
					for _, opt := range mfConfig.Options {
						v, ok := opt.(bool)
						if !ok {
							return configs, inputs, fmt.Errorf("%s type missmatch: bool != %T", ref, opt)
						}
						o = append(o, v)
					}
				}
				configs.SetBool(ref, d, o...)
			case module.IntData:
				var d *int64
				var o []int64
				if mfConfig.Value != nil {
					v, ok := mfConfig.Value.(int)
					if !ok {
						return configs, inputs, fmt.Errorf("%s type missmatch: int != %T", ref, mfConfig.Value)
					}
					tmp := int64(v)
					d = &tmp
				}
				if mfConfig.Options != nil {
					for _, opt := range mfConfig.Options {
						v, ok := opt.(int)
						if !ok {
							return configs, inputs, fmt.Errorf("%s type missmatch: int != %T", ref, opt)
						}
						o = append(o, int64(v))
					}
				}
				configs.SetInt64(ref, d, o...)
			case module.FloatData:
				var d *float64
				var o []float64
				if mfConfig.Value != nil {
					v, ok := mfConfig.Value.(float64)
					if !ok {
						return configs, inputs, fmt.Errorf("%s type missmatch: float64 != %T", ref, mfConfig.Value)
					}
					d = &v
				}
				if mfConfig.Options != nil {
					for _, opt := range mfConfig.Options {
						v, ok := opt.(float64)
						if !ok {
							return configs, inputs, fmt.Errorf("%s type missmatch: float64 != %T", ref, opt)
						}
						o = append(o, v)
					}
				}
				configs.SetFloat64(ref, d, o...)
			default:
				return configs, inputs, fmt.Errorf("invalid data type '%s'", mfConfig.Type)
			}
			if mfConfig.UserInput != nil {
				inputs[ref] = module.Input{
					Name:        mfConfig.UserInput.Name,
					Description: mfConfig.UserInput.Description,
					Required:    mfConfig.UserInput.Required,
					Group:       mfConfig.UserInput.Group,
					Type:        mfConfig.UserInput.Type,
					Constraints: mfConfig.UserInput.Constraints,
				}
			}
		}
		return configs, inputs, nil
	}
	return configs, nil, nil
}
