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
)

func ParseModule(mfModule Module) (module.Module, error) {
	m := module.Module{
		Name:        mfModule.Name,
		Description: mfModule.Description,
		License:     mfModule.License,
		Author:      mfModule.Author,
	}
	if !module.IsValidModuleID(mfModule.ID) {
		return m, fmt.Errorf("invalid module ID format '%s'", mfModule.ID)
	}
	m.ID = mfModule.ID
	if !module.IsValidSemVer(mfModule.Version) {
		return m, fmt.Errorf("invalid version format '%s'", mfModule.Version)
	}
	m.Version = mfModule.Version
	if !module.IsValidModuleType(mfModule.Type) {
		return m, fmt.Errorf("invalid module type '%s'", mfModule.Type)
	}
	m.Type = mfModule.Type
	if !module.IsValidDeploymentType(mfModule.DeploymentType) {
		return m, fmt.Errorf("invalid deployment type '%s'", mfModule.DeploymentType)
	}
	m.DeploymentType = mfModule.DeploymentType
	services, err := parseModuleServices(mfModule.Services)
	if err != nil {
		return m, err
	}
	volumes, err := parseModuleVolumes(mfModule.Volumes, services)
	if err != nil {
		return m, err
	}
	dependencies, err := parseModuleDependencies(mfModule.Dependencies, services)
	if err != nil {
		return m, err
	}
	resources, err := parseModuleResources(mfModule.Resources, services)
	if err != nil {
		return m, err
	}
	secrets, err := parseModuleSecrets(mfModule.Secrets, services)
	if err != nil {
		return m, err
	}
	configs, err := parseModuleConfigs(mfModule.Configs, services)
	if err != nil {
		return m, err
	}
	m.Services = services
	m.Volumes = volumes
	m.Dependencies = dependencies
	m.Resources = resources
	m.Secrets = secrets
	m.Configs = configs
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
			RunConfig:     mfService.RunConfig,
			Include:       include,
			Tmpfs:         tmpfs,
			HttpEndpoints: httpEndpoints,
			Dependencies:  dependencies,
			PortMappings:  portMappings,
		}
	}
	return services, nil
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
			if v, ok := tmpfs[mfTmpf.MountPoint]; ok {
				if v.Size == uint64(mfTmpf.Size) && v.Mode == mfTmpf.Mode {
					continue
				}
				return nil, fmt.Errorf("duplicate '%s'", mfTmpf.MountPoint)
			}
			tmpfs[mfTmpf.MountPoint] = module.TmpfsMount{
				Size: uint64(mfTmpf.Size),
				Mode: mfTmpf.Mode,
			}
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

func parseServicePortMappings(mfPortMappings []PortMapping) ([]module.PortMapping, error) {
	if mfPortMappings != nil && len(mfPortMappings) > 0 {
		var mappings []module.PortMapping
		hostPorts := make(map[int]struct{})
		for _, mfPortMapping := range mfPortMappings {
			portMapping := module.PortMapping{
				Name:     mfPortMapping.Name,
				Port:     mfPortMapping.Port.Range(),
				Protocol: mfPortMapping.Protocol,
			}
			if mfPortMapping.HostPort != nil {
				hpr := mfPortMapping.HostPort.Range()
				lp := len(portMapping.Port)
				lhp := len(hpr)
				if lp != lhp {
					if lp > lhp {
						return mappings, fmt.Errorf("range mismatch '%s > %s'", mfPortMapping.Port, *mfPortMapping.HostPort)
					}
					if lp > 1 && lp < lhp {
						return mappings, fmt.Errorf("range mismatch '%s < %s'", mfPortMapping.Port, *mfPortMapping.HostPort)
					}
				}
				for _, hp := range hpr {
					if _, ok := hostPorts[hp]; ok {
						return mappings, fmt.Errorf("duplicate '%d'", hp)
					}
					hostPorts[hp] = struct{}{}
				}
				portMapping.HostPort = hpr
			}
			mappings = append(mappings, portMapping)
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

func parseModuleVolumes(mfVolumes map[string][]VolumeTarget, services map[string]*module.Service) ([]string, error) {
	if mfVolumes != nil && len(mfVolumes) > 0 {
		var volumes []string
		for name, mfTargets := range mfVolumes {
			volumes = append(volumes, name)
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
			if !module.IsValidModuleID(id) {
				return moduleDependencies, fmt.Errorf("invalid module ID format '%s'", id)
			}
			var rs []string
			for rqSrv, mfTargets := range dependency.RequiredServices {
				rs = append(rs, rqSrv)
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

func parseModuleResources(mfResources map[string]Resource, services map[string]*module.Service) (map[string]module.Resource, error) {
	if mfResources != nil && len(mfResources) > 0 {
		resources := make(map[string]module.Resource)
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
									if rt.Reference == ref && rt.ReadOnly == mfTarget.ReadOnly {
										continue
									}
									return resources, fmt.Errorf("'%s' & '%s' -> '%s' -> '%s'", rt.Reference, ref, srv, mfTarget.MountPoint)
								}
								v.Resources[mfTarget.MountPoint] = module.ResourceTarget{
									Reference: ref,
									ReadOnly:  mfTarget.ReadOnly,
								}
							}
						}
					}
				}
			}
			resources[ref] = module.Resource{
				ResourceBase: module.ResourceBase{
					Type: mfResource.Type,
					Tags: mfResource.Tags,
				},
				UserInput: mfResource.UserInput,
			}
		}
		return resources, nil
	}
	return nil, nil
}

func parseModuleSecrets(mfSecrets map[string]Secret, services map[string]*module.Service) (map[string]module.Secret, error) {
	if mfSecrets != nil && len(mfSecrets) > 0 {
		secrets := make(map[string]module.Secret)
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
									return secrets, fmt.Errorf("'%s' & '%s' -> '%s' -> '%s'", r, ref, srv, mfTarget.MountPoint)
								}
								v.Secrets[mfTarget.MountPoint] = ref
							}
						}
					}
				}
			}
			secrets[ref] = module.Secret{
				ResourceBase: module.ResourceBase{
					Type: mfSecret.Type,
					Tags: mfSecret.Tags,
				},
				UserInput: mfSecret.UserInput,
			}
		}
		return secrets, nil
	}
	return nil, nil
}

func parseModuleConfigs(mfConfigs map[string]ConfigValue, services map[string]*module.Service) (map[string]module.ConfigValue, error) {
	if mfConfigs != nil && len(mfConfigs) > 0 {
		configs := make(map[string]module.ConfigValue)
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
									return configs, fmt.Errorf("'%s' & '%s' -> '%s' -> '%s'", r, ref, srv, mfTarget.RefVar)
								}
								v.Configs[mfTarget.RefVar] = ref
							}
						}
					}
				}
			}
			configs[ref] = module.ConfigValue{
				Value:     mfConfig.Value,
				Options:   mfConfig.Options,
				Type:      mfConfig.Type,
				UserInput: mfConfig.UserInput,
			}
		}
		return configs, nil
	}
	return nil, nil
}
