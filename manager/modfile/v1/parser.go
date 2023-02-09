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
	"io/fs"
	"module-manager/manager/itf"
	"module-manager/manager/util/set"
	"strconv"
	"strings"
	"time"
)

func (mf Module) Parse() (itf.Module, error) {
	m := itf.Module{
		ID:             mf.ID,
		Name:           mf.Name,
		Description:    mf.Description,
		License:        mf.License,
		Author:         mf.Author,
		Version:        mf.Version,
		Type:           mf.Type,
		DeploymentType: mf.DeploymentType,
	}
	m.Tags = parseModuleTags(mf.Tags)
	services, err := parseModuleServices(mf.Services)
	if err != nil {
		return m, err
	}
	err = parseModuleSrvReferences(mf.ServiceReferences, services)
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
	userInput := itf.Inputs{}
	if mf.InputGroups != nil && len(mf.InputGroups) > 0 {
		userInput.Groups = parseInputGroups(mf.InputGroups)
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
	m.Inputs = userInput
	return m, nil
}

func parseModuleTags(mfTags []string) set.Set[string] {
	if mfTags != nil && len(mfTags) > 0 {
		tags := make(set.Set[string])
		for _, tag := range mfTags {
			tags[tag] = struct{}{}
		}
		return tags
	}
	return nil
}

func parseModuleServices(mfServices map[string]Service) (map[string]*itf.Service, error) {
	services := make(map[string]*itf.Service)
	if mfServices != nil && len(mfServices) > 0 {
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
			services[name] = &itf.Service{
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
	}
	return services, nil
}

func parseServiceRunConfig(mfRunConfig RunConfig) itf.RunConfig {
	rc := itf.RunConfig{
		MaxRetries:     mfRunConfig.MaxRetries,
		RemoveAfterRun: mfRunConfig.RemoveAfterRun,
		StopSignal:     mfRunConfig.StopSignal,
		PseudoTTY:      mfRunConfig.PseudoTTY,
	}
	if mfRunConfig.StopTimeout != nil {
		rc.StopTimeout = (*time.Duration)(mfRunConfig.StopTimeout)
	}
	return rc
}

func parseServiceInclude(mfInclude []BindMount) (map[string]itf.BindMount, error) {
	if mfInclude != nil && len(mfInclude) > 0 {
		include := make(map[string]itf.BindMount)
		for _, mfBindMount := range mfInclude {
			if v, ok := include[mfBindMount.MountPoint]; ok {
				if v.Source == mfBindMount.Source && v.ReadOnly == mfBindMount.ReadOnly {
					continue
				}
				return nil, fmt.Errorf("duplicate '%s'", mfBindMount.MountPoint)
			}
			include[mfBindMount.MountPoint] = itf.BindMount{
				Source:   mfBindMount.Source,
				ReadOnly: mfBindMount.ReadOnly,
			}
		}
		return include, nil
	}
	return nil, nil
}

func parseServiceTmpfs(mfTmpfs []TmpfsMount) (map[string]itf.TmpfsMount, error) {
	if mfTmpfs != nil && len(mfTmpfs) > 0 {
		tmpfs := make(map[string]itf.TmpfsMount)
		for _, mfTmpf := range mfTmpfs {
			if _, ok := tmpfs[mfTmpf.MountPoint]; ok {
				return nil, fmt.Errorf("duplicate '%s'", mfTmpf.MountPoint)
			}
			tm := itf.TmpfsMount{Size: uint64(mfTmpf.Size)}
			if mfTmpf.Mode != nil {
				tm.Mode = (*fs.FileMode)(mfTmpf.Mode)
			}
			tmpfs[mfTmpf.MountPoint] = tm
		}
		return tmpfs, nil
	}
	return nil, nil
}

func parseServiceHttpEndpoints(mfHttpEndpoints []HttpEndpoint) (map[string]itf.HttpEndpoint, error) {
	if mfHttpEndpoints != nil && len(mfHttpEndpoints) > 0 {
		httpEndpoints := make(map[string]itf.HttpEndpoint)
		for _, mfHttpEndpoint := range mfHttpEndpoints {
			p := mfHttpEndpoint.Path
			if mfHttpEndpoint.ExtPath != nil {
				p = *mfHttpEndpoint.ExtPath
			}
			if v, ok := httpEndpoints[p]; ok {
				if v.Name == mfHttpEndpoint.Name && v.Port == mfHttpEndpoint.Port && v.Path == mfHttpEndpoint.Path {
					continue
				}
				return nil, fmt.Errorf("duplicate '%s'", mfHttpEndpoint.Path)
			}
			httpEndpoints[p] = itf.HttpEndpoint{
				Name: mfHttpEndpoint.Name,
				Port: mfHttpEndpoint.Port,
				Path: mfHttpEndpoint.Path,
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

func parseServicePortMappings(mfPortMappings []PortMapping) (itf.PortMappings, error) {
	if mfPortMappings != nil && len(mfPortMappings) > 0 {
		mappings := make(itf.PortMappings)
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

func parseServiceDependencies(mfServiceDependencies []string) (set.Set[string], error) {
	if mfServiceDependencies != nil && len(mfServiceDependencies) > 0 {
		serviceDependencies := make(set.Set[string])
		for _, ref := range mfServiceDependencies {
			if _, ok := serviceDependencies[ref]; ok {
				return serviceDependencies, fmt.Errorf("duplicate '%s'", ref)
			}
			serviceDependencies[ref] = struct{}{}
		}
		return serviceDependencies, nil
	}
	return nil, nil
}

func parseModuleSrvReferences(mfSrvRefs map[string][]DependencyTarget, services map[string]*itf.Service) error {
	if mfSrvRefs != nil && len(mfSrvRefs) > 0 {
		for mfSrv, mfTargets := range mfSrvRefs {
			for _, mfTarget := range mfTargets {
				for _, srv := range mfTarget.Services {
					if v, ok := services[srv]; ok {
						if v.SrvReferences == nil {
							v.SrvReferences = make(map[string]string)
						}
						if s, k := v.SrvReferences[mfTarget.RefVar]; k {
							if s == mfSrv {
								continue
							}
							return fmt.Errorf("service '%s' invalid service reference: duplicate '%s'", srv, mfTarget.RefVar)
						}
						v.SrvReferences[mfTarget.RefVar] = mfSrv
					} else {
						return fmt.Errorf("invalid service reference: service '%s' not defined", srv)
					}
				}
			}
		}
	}
	return nil
}

func parseModuleVolumes(mfVolumes map[string][]VolumeTarget, services map[string]*itf.Service) (set.Set[string], error) {
	if mfVolumes != nil && len(mfVolumes) > 0 {
		volumes := make(set.Set[string])
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
					} else {
						return volumes, fmt.Errorf("invalid volume: service '%s' not defined", srv)
					}
				}
			}
		}
		return volumes, nil
	}
	return nil, nil
}

func parseModuleDependencies(mfModuleDependencies map[string]ModuleDependency, services map[string]*itf.Service) (map[string]itf.ModuleDependency, error) {
	if mfModuleDependencies != nil && len(mfModuleDependencies) > 0 {
		moduleDependencies := make(map[string]itf.ModuleDependency)
		for id, dependency := range mfModuleDependencies {
			rs := make(set.Set[string])
			for rqSrv, mfTargets := range dependency.RequiredServices {
				rs[rqSrv] = struct{}{}
				for _, mfTarget := range mfTargets {
					if mfTarget.Services != nil && len(mfTarget.Services) > 0 {
						for _, srv := range mfTarget.Services {
							if v, ok := services[srv]; ok {
								if v.ExternalDependencies == nil {
									v.ExternalDependencies = make(map[string]itf.ExternalDependencyTarget)
								}
								if ep, k := v.ExternalDependencies[mfTarget.RefVar]; k {
									if ep.ID == id && ep.Service == rqSrv {
										continue
									}
									return moduleDependencies, fmt.Errorf("service '%s' invalid module dependency: duplicate '%s'", srv, mfTarget.RefVar)
								}
								v.ExternalDependencies[mfTarget.RefVar] = itf.ExternalDependencyTarget{
									ID:      id,
									Service: rqSrv,
								}
							} else {
								return moduleDependencies, fmt.Errorf("invalid module dependency: service '%s' not defined", srv)
							}
						}
					}
				}
			}
			moduleDependencies[id] = itf.ModuleDependency{
				Version:          dependency.Version,
				RequiredServices: rs,
			}
		}
		return moduleDependencies, nil
	}
	return nil, nil
}

func parseModuleResources(mfResources map[string]Resource, services map[string]*itf.Service) (map[string]set.Set[string], map[string]itf.Input, error) {
	if mfResources != nil && len(mfResources) > 0 {
		resources := make(map[string]set.Set[string])
		inputs := make(map[string]itf.Input)
		for ref, mfResource := range mfResources {
			if mfResource.Targets != nil && len(mfResource.Targets) > 0 {
				for _, mfTarget := range mfResource.Targets {
					if mfTarget.Services != nil && len(mfTarget.Services) > 0 {
						for _, srv := range mfTarget.Services {
							if v, ok := services[srv]; ok {
								if v.Resources == nil {
									v.Resources = make(map[string]itf.ResourceTarget)
								}
								if rt, k := v.Resources[mfTarget.MountPoint]; k {
									if rt.Ref == ref && rt.ReadOnly == mfTarget.ReadOnly {
										continue
									}
									return resources, inputs, fmt.Errorf("'%s' & '%s' -> '%s' -> '%s'", rt.Ref, ref, srv, mfTarget.MountPoint)
								}
								v.Resources[mfTarget.MountPoint] = itf.ResourceTarget{
									Ref:      ref,
									ReadOnly: mfTarget.ReadOnly,
								}
							} else {
								return resources, inputs, fmt.Errorf("invalid resource: service '%s' not defined", srv)
							}
						}
					}
				}
			}
			var r set.Set[string]
			if mfResource.Tags != nil && len(mfResource.Tags) > 0 {
				r = make(set.Set[string])
				for _, tag := range mfResource.Tags {
					r[tag] = struct{}{}
				}
			}
			resources[ref] = r
			if mfResource.UserInput != nil {
				inputs[ref] = itf.Input{
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

func parseModuleSecrets(mfSecrets map[string]Secret, services map[string]*itf.Service) (map[string]itf.Secret, map[string]itf.Input, error) {
	if mfSecrets != nil && len(mfSecrets) > 0 {
		secrets := make(map[string]itf.Secret)
		inputs := make(map[string]itf.Input)
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
							} else {
								return secrets, inputs, fmt.Errorf("invalid secret: service '%s' not defined", srv)
							}
						}
					}
				}
			}
			r := itf.Secret{Type: mfSecret.Type}
			if mfSecret.Tags != nil && len(mfSecret.Tags) > 0 {
				r.Tags = make(set.Set[string])
				for _, tag := range mfSecret.Tags {
					r.Tags[tag] = struct{}{}
				}
			}
			secrets[ref] = r
			if mfSecret.UserInput != nil {
				inputs[ref] = itf.Input{
					Name:        mfSecret.UserInput.Name,
					Description: mfSecret.UserInput.Description,
					Required:    mfSecret.UserInput.Required,
					Group:       mfSecret.UserInput.Group,
				}
			}
		}
		return secrets, inputs, nil
	}
	return nil, nil, nil
}

func parseModuleConfigs(mfConfigs map[string]ConfigValue, services map[string]*itf.Service) (itf.Configs, map[string]itf.Input, error) {
	if mfConfigs != nil && len(mfConfigs) > 0 {
		configs := make(itf.Configs)
		inputs := make(map[string]itf.Input)
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
			dt, ok := itf.DataTypeRefMap[mfConfig.DataType]
			if !ok {
				return configs, inputs, fmt.Errorf("%s ivalid data type '%s'", ref, mfConfig.DataType)
			}
			if mfConfig.IsList {
				switch dt {
				case itf.StringType:
					d, o, co, err := parseConfigSlice(mfConfig.Value, mfConfig.Options, mfConfig.TypeOptions, parseConfigValueString)
					if err != nil {
						return configs, inputs, fmt.Errorf("error parsing config '%s': %s", ref, err)
					}
					configs.SetStringSlice(ref, d, o, mfConfig.OptionsExt, mfConfig.Type, co, mfConfig.Delimiter)
				case itf.BoolType:
					d, o, co, err := parseConfigSlice(mfConfig.Value, mfConfig.Options, mfConfig.TypeOptions, parseConfigValueBool)
					if err != nil {
						return configs, inputs, fmt.Errorf("error parsing config '%s': %s", ref, err)
					}
					configs.SetBoolSlice(ref, d, o, mfConfig.OptionsExt, mfConfig.Type, co, mfConfig.Delimiter)
				case itf.Int64Type:
					d, o, co, err := parseConfigSlice(mfConfig.Value, mfConfig.Options, mfConfig.TypeOptions, parseConfigValueInt64)
					if err != nil {
						return configs, inputs, fmt.Errorf("error parsing config '%s': %s", ref, err)
					}
					configs.SetInt64Slice(ref, d, o, mfConfig.OptionsExt, mfConfig.Type, co, mfConfig.Delimiter)
				case itf.Float64Type:
					d, o, co, err := parseConfigSlice(mfConfig.Value, mfConfig.Options, mfConfig.TypeOptions, parseConfigValueFloat64)
					if err != nil {
						return configs, inputs, fmt.Errorf("error parsing config '%s': %s", ref, err)
					}
					configs.SetFloat64Slice(ref, d, o, mfConfig.OptionsExt, mfConfig.Type, co, mfConfig.Delimiter)
				}
			} else {
				switch dt {
				case itf.StringType:
					d, o, co, err := parseConfig(mfConfig.Value, mfConfig.Options, mfConfig.TypeOptions, parseConfigValueString)
					if err != nil {
						return configs, inputs, fmt.Errorf("error parsing config '%s': %s", ref, err)
					}
					configs.SetString(ref, d, o, mfConfig.OptionsExt, mfConfig.Type, co)
				case itf.BoolType:
					d, o, co, err := parseConfig(mfConfig.Value, mfConfig.Options, mfConfig.TypeOptions, parseConfigValueBool)
					if err != nil {
						return configs, inputs, fmt.Errorf("error parsing config '%s': %s", ref, err)
					}
					configs.SetBool(ref, d, o, mfConfig.OptionsExt, mfConfig.Type, co)
				case itf.Int64Type:
					d, o, co, err := parseConfig(mfConfig.Value, mfConfig.Options, mfConfig.TypeOptions, parseConfigValueInt64)
					if err != nil {
						return configs, inputs, fmt.Errorf("error parsing config '%s': %s", ref, err)
					}
					configs.SetInt64(ref, d, o, mfConfig.OptionsExt, mfConfig.Type, co)
				case itf.Float64Type:
					d, o, co, err := parseConfig(mfConfig.Value, mfConfig.Options, mfConfig.TypeOptions, parseConfigValueFloat64)
					if err != nil {
						return configs, inputs, fmt.Errorf("error parsing config '%s': %s", ref, err)
					}
					configs.SetFloat64(ref, d, o, mfConfig.OptionsExt, mfConfig.Type, co)
				}
			}

			if mfConfig.UserInput != nil {
				inputs[ref] = itf.Input{
					Name:        mfConfig.UserInput.Name,
					Description: mfConfig.UserInput.Description,
					Required:    mfConfig.UserInput.Required,
					Group:       mfConfig.UserInput.Group,
				}
			}
		}
		return configs, inputs, nil
	}
	return nil, nil, nil
}

func parseConfig[T any](val any, opt []any, ctOpt map[string]any, valParser func(any) (T, error)) (d *T, o []T, co itf.ConfigTypeOptions, err error) {
	if val != nil {
		v, er := valParser(val)
		if er != nil {
			err = er
			return
		}
		d = &v
	}
	if opt != nil && len(opt) > 0 {
		if o, err = parseConfigOptions(opt, valParser); err != nil {
			return
		}
	}
	if ctOpt != nil && len(ctOpt) > 0 {
		co, err = parseConfigTypeOptions(ctOpt)
	}
	return
}

func parseConfigSlice[T any](val any, opt []any, ctOpt map[string]any, valParser func(any) (T, error)) (d []T, o []T, co itf.ConfigTypeOptions, err error) {
	if val != nil {
		v, ok := val.([]any)
		if !ok {
			err = fmt.Errorf("type missmatch: %T != slice", val)
			return
		}
		for _, i := range v {
			pi, e := valParser(i)
			if e != nil {
				err = e
				return
			}
			d = append(d, pi)
		}
	}
	if opt != nil && len(opt) > 0 {
		if o, err = parseConfigOptions(opt, valParser); err != nil {
			return
		}
	}
	if ctOpt != nil && len(ctOpt) > 0 {
		co, err = parseConfigTypeOptions(ctOpt)
	}
	return
}

func parseConfigOptions[T any](opt []any, valParser func(any) (T, error)) ([]T, error) {
	var opts []T
	for _, o := range opt {
		v, err := valParser(o)
		if err != nil {
			return nil, err
		}
		opts = append(opts, v)
	}
	return opts, nil
}

func parseConfigValueString(val any) (string, error) {
	v, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("invalid data type '%T'", val)
	}
	return v, nil
}

func parseConfigValueBool(val any) (bool, error) {
	v, ok := val.(bool)
	if !ok {
		return false, fmt.Errorf("invalid data type '%T'", val)
	}
	return v, nil
}

func parseConfigValueInt64(val any) (int64, error) {
	var i int64
	switch v := val.(type) {
	case int:
		i = int64(v)
	case int8:
		i = int64(v)
	case int16:
		i = int64(v)
	case int32:
		i = int64(v)
	case int64:
		i = v
	default:
		return i, fmt.Errorf("invalid data type '%T'", val)
	}
	return i, nil
}

func parseConfigValueFloat64(val any) (float64, error) {
	var f float64
	switch v := val.(type) {
	case float32:
		f = float64(v)
	case float64:
		f = v
	default:
		return f, fmt.Errorf("invalid data type '%T'", val)
	}
	return f, nil
}

func parseConfigTypeOptions(opt map[string]any) (itf.ConfigTypeOptions, error) {
	o := make(itf.ConfigTypeOptions)
	for key, val := range opt {
		switch v := val.(type) {
		case string:
			o.SetString(key, v)
		case bool:
			o.SetBool(key, v)
		case int:
			o.SetInt64(key, int64(v))
		case int8:
			o.SetInt64(key, int64(v))
		case int16:
			o.SetInt64(key, int64(v))
		case int32:
			o.SetInt64(key, int64(v))
		case int64:
			o.SetInt64(key, v)
		case float32:
			o.SetFloat64(key, float64(v))
		case float64:
			o.SetFloat64(key, v)
		default:
			return nil, fmt.Errorf("unknown data type '%T'", val)
		}
	}
	return o, nil
}

func parseInputGroups(mfInputGroups map[string]InputGroup) map[string]itf.InputGroup {
	iGroups := make(map[string]itf.InputGroup)
	for ref, mfInputGroup := range mfInputGroups {
		iGroups[ref] = itf.InputGroup{
			Name:        mfInputGroup.Name,
			Description: mfInputGroup.Description,
			Group:       mfInputGroup.Group,
		}
	}
	return iGroups
}
