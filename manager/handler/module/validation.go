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

package module

import (
	"errors"
	"fmt"
	"module-manager/manager/handler/sem_ver"
	"module-manager/manager/itf"
	"regexp"
)

func Validate(m itf.Module, cDef map[string]itf.ConfigDefinition) error {
	if !IsValidModuleID(m.ID) {
		return fmt.Errorf("invalid module ID format '%s'", m.ID)
	}
	if !sem_ver.IsValidSemVer(m.Version) {
		return fmt.Errorf("invalid version format '%s'", m.Version)
	}
	if !IsValidModuleType(m.Type) {
		return fmt.Errorf("invalid module type '%s'", m.Type)
	}
	if !IsValidDeploymentType(m.DeploymentType) {
		return fmt.Errorf("invlaid deployment type '%s'", m.DeploymentType)
	}
	for v := range m.Volumes {
		if v == "" {
			return errors.New("invalid volume name")
		}
	}
	if m.Dependencies != nil {
		for mid, dependency := range m.Dependencies {
			if !IsValidModuleID(mid) {
				return fmt.Errorf("invalid dependency module ID format '%s'", mid)
			}
			if err := sem_ver.ValidateSemVerRange(dependency.Version); err != nil {
				return fmt.Errorf("dependency '%s': %s", mid, err)
			}
			if dependency.RequiredServices == nil {
				return fmt.Errorf("missing services for dependency '%s'", mid)
			}
			for s := range dependency.RequiredServices {
				if s == "" {
					return fmt.Errorf("invalid service for dependency '%s'", mid)
				}
			}
		}
	}
	if m.Resources != nil {
		for ref := range m.Resources {
			if ref == "" {
				return errors.New("invalid resource reference")
			}
		}
	}
	if m.Secrets != nil {
		for ref := range m.Secrets {
			if ref == "" {
				return errors.New("invalid secret reference")
			}
		}
	}
	if m.Configs != nil {
		err := ValidateConfigs(m.Configs, cDef)
		if err != nil {
			return err
		}
	}
	if m.Inputs.Groups != nil {
		for ref := range m.Inputs.Groups {
			if ref == "" {
				return errors.New("invalid user input group reference")
			}
		}
	}
	if m.Inputs.Resources != nil {
		err := ValidateInputs(m.Inputs.Resources, m.Resources, "resource", m.Inputs.Groups)
		if err != nil {
			return err
		}
	}
	if m.Inputs.Secrets != nil {
		err := ValidateInputs(m.Inputs.Secrets, m.Secrets, "secret", m.Inputs.Groups)
		if err != nil {
			return err
		}
	}
	if m.Inputs.Configs != nil {
		err := ValidateInputs(m.Inputs.Configs, m.Configs, "config", m.Inputs.Groups)
		if err != nil {
			return err
		}
	}
	if m.Services == nil || len(m.Services) == 0 {
		return errors.New("missing services")
	}
	hostPorts := make(map[uint]struct{})
	for ref, service := range m.Services {
		if ref == "" {
			return errors.New("invalid service reference")
		}
		if err := validateServiceRunConfig(service.RunConfig); err != nil {
			return fmt.Errorf("invalid service run config: '%s' %s", ref, err)
		}
		if err := validateServiceMountPoints(service); err != nil {
			return fmt.Errorf("invalid service mount point: '%s' -> %s", ref, err)
		}
		if err := validateServiceRefVars(service); err != nil {
			return fmt.Errorf("invalid service reference variable: '%s' -> %s", ref, err)
		}
		if service.Volumes != nil {
			for _, volume := range service.Volumes {
				if _, ok := m.Volumes[volume]; !ok {
					return fmt.Errorf("invalid service volume: '%s' -> '%s'", ref, volume)
				}
			}
		}
		if service.Resources != nil {
			for _, target := range service.Resources {
				if _, ok := m.Resources[target.Ref]; !ok {
					return fmt.Errorf("invalid service resource: '%s' -> '%s'", ref, target.Ref)
				}
			}
		}
		if service.Secrets != nil {
			for _, secretRef := range service.Secrets {
				if _, ok := m.Secrets[secretRef]; !ok {
					return fmt.Errorf("invalid service secret: '%s' -> '%s'", ref, secretRef)
				}
			}
		}
		if service.Configs != nil {
			for _, confRef := range service.Configs {
				if _, ok := m.Configs[confRef]; !ok {
					return fmt.Errorf("invalid service secret: '%s' -> '%s'", ref, confRef)
				}
			}
		}
		if service.HttpEndpoints != nil {
			gwPaths := make(map[string]string)
			for path, edpt := range service.HttpEndpoints {
				if path == "" {
					return errors.New("invalid path")
				}
				if edpt.GwPath != nil {
					if v, ok := gwPaths[*edpt.GwPath]; ok {
						return fmt.Errorf("invalid service http endpoint: '%s' -> '%s' & '%s' -> '%s'", ref, path, v, *edpt.GwPath)
					}
					gwPaths[*edpt.GwPath] = path
				}

			}
		}
		if service.Dependencies != nil {
			for _, target := range service.Dependencies {
				if _, ok := m.Services[target.Service]; !ok {
					return fmt.Errorf("invalid service dependency: '%s' -> '%s'", ref, target.Service)
				}
				if !IsValidSrvDepCondition(target.Condition) {
					return fmt.Errorf("invalid service dependency condition: '%s' -> '%s'", ref, target.Condition)
				}
			}
		}
		if service.ExternalDependencies != nil {
			for _, target := range service.ExternalDependencies {
				if !IsValidModuleID(target.ID) {
					return fmt.Errorf("invalid service external dependency: '%s' -> '%s'", ref, target.ID)
				}
			}
		}
		if service.PortMappings != nil {
			for _, pm := range service.PortMappings {
				if pm.HostPort != nil && len(pm.HostPort) > 0 {
					if len(pm.HostPort) > 1 {
						for i := pm.HostPort[0]; i <= pm.HostPort[1]; i++ {
							if _, ok := hostPorts[i]; ok {
								return fmt.Errorf("duplicate host port '%d'", i)
							}
							hostPorts[i] = struct{}{}
						}
					} else {
						if _, ok := hostPorts[pm.HostPort[0]]; ok {
							return fmt.Errorf("duplicate host port '%d'", pm.HostPort[0])
						}
						hostPorts[pm.HostPort[0]] = struct{}{}
					}
				}
			}
		}
	}
	return nil
}

func validateServiceRunConfig(rc itf.RunConfig) error {
	if !IsValidRestartStrategy(rc.RestartStrategy) {
		return fmt.Errorf("invalid restart strategy '%s'", rc.RestartStrategy)
	}
	if rc.RestartStrategy == itf.RestartOnFail && rc.Retries == nil {
		return errors.New("missing retries")
	}
	if rc.RestartStrategy == itf.RestartAlways && rc.RemoveAfterRun {
		return fmt.Errorf("remove after run and restart strategy '%s'", rc.RestartStrategy)
	}
	return nil
}

func validateServiceRefVars(service *itf.Service) error {
	refVars := make(map[string]string)
	if service.Configs != nil {
		for rv := range service.Configs {
			if rv == "" {
				return errors.New("invalid ref var")
			}
			refVars[rv] = "configs"
		}
	}
	if service.Dependencies != nil {
		for rv := range service.Dependencies {
			if rv == "" {
				return errors.New("invalid ref var")
			}
			if v, ok := refVars[rv]; ok {
				return fmt.Errorf("'%s' -> '%s' & '%s'", rv, v, "dependencies")
			}
			refVars[rv] = "dependencies"
		}
	}
	if service.ExternalDependencies != nil {
		for rv := range service.ExternalDependencies {
			if rv == "" {
				return errors.New("invalid ref var")
			}
			if v, ok := refVars[rv]; ok {
				return fmt.Errorf("'%s' -> '%s' & '%s'", rv, v, "external dependencies")
			}
			refVars[rv] = "external dependencies"
		}
	}
	return nil
}

func validateServiceMountPoints(service *itf.Service) error {
	mountPoints := make(map[string]string)
	if service.Include != nil {
		for mp := range service.Include {
			if mp == "" {
				return errors.New("invalid mount point")
			}
			mountPoints[mp] = "include"
		}
	}
	if service.Tmpfs != nil {
		for mp := range service.Tmpfs {
			if mp == "" {
				return errors.New("invalid mount point")
			}
			if v, ok := mountPoints[mp]; ok {
				return fmt.Errorf("'%s' -> '%s' & '%s'", mp, v, "tmpfs")
			}
			mountPoints[mp] = "tmpfs"
		}
	}
	if service.Volumes != nil {
		for mp := range service.Volumes {
			if mp == "" {
				return errors.New("invalid mount point")
			}
			if v, ok := mountPoints[mp]; ok {
				return fmt.Errorf("'%s' -> '%s' & '%s'", mp, v, "volumes")
			}
			mountPoints[mp] = "volumes"
		}
	}
	if service.Resources != nil {
		for mp := range service.Resources {
			if mp == "" {
				return errors.New("invalid mount point")
			}
			if v, ok := mountPoints[mp]; ok {
				return fmt.Errorf("'%s' -> '%s' & '%s'", mp, v, "resources")
			}
			mountPoints[mp] = "resources"
		}
	}
	if service.Secrets != nil {
		for mp := range service.Secrets {
			if mp == "" {
				return errors.New("invalid mount point")
			}
			if v, ok := mountPoints[mp]; ok {
				return fmt.Errorf("'%s' -> '%s' & '%s'", mp, v, "secrets")
			}
			mountPoints[mp] = "secrets"
		}
	}
	return nil
}

func IsValidModuleType(s string) bool {
	_, ok := itf.ModuleTypeMap[s]
	return ok
}

func IsValidDeploymentType(s string) bool {
	_, ok := itf.DeploymentTypeMap[s]
	return ok
}

func IsValidModuleID(s string) bool {
	re := regexp.MustCompile(`^([a-z0-9A-Z-_.]+)(:\d+)?([\/a-zA-Z0-9-\.]+)?$`)
	return re.MatchString(s)
}

func IsValidSrvDepCondition(s string) bool {
	_, ok := itf.SrvDepConditionMap[s]
	return ok
}

func IsValidRestartStrategy(s string) bool {
	_, ok := itf.RestartStrategyMap[s]
	return ok
}

func ValidateInputs[T any](inputs map[string]itf.Input, refs map[string]T, refName string, groups map[string]itf.InputGroup) error {
	if refs == nil {
		return fmt.Errorf("missing %ss for user inputs", refName)
	}
	for ref, input := range inputs {
		if ref == "" {
			return errors.New("invalid input reference")
		}
		if _, ok := refs[ref]; !ok {
			return fmt.Errorf("missing %s for input '%s'", refName, ref)
		}
		if input.Group != nil {
			if groups == nil {
				return errors.New("missing groups for inputs")
			}
			if _, ok := groups[*input.Group]; !ok {
				return fmt.Errorf("missing group for input '%s'", ref)
			}
		}
	}
	return nil
}

func ValidateConfigs(c itf.Configs, cDef map[string]itf.ConfigDefinition) error {
	for ref, cv := range c {
		if ref == "" {
			return errors.New("invalid config reference")
		}
		def, ok := cDef[cv.Type]
		if !ok {
			return fmt.Errorf("invalid config type '%s'", cv.Type)
		}
		if _, ok = def.DataType[cv.DataType]; !ok {
			return fmt.Errorf("invalid data type '%s' for config type '%s'", cv.DataType, cv.Type)
		}
		if cv.TypeOpt != nil && def.Options == nil {
			return fmt.Errorf("config type '%s' does not support options", cv.Type)
		}
		if cv.TypeOpt == nil && def.Options != nil {
			for key, defOpt := range def.Options {
				if defOpt.Required {
					return fmt.Errorf("option '%s' is required by config type '%s'", key, cv.Type)
				}
			}
		}
		if cv.TypeOpt != nil && def.Options != nil {
			for key := range cv.TypeOpt {
				if _, ok := def.Options[key]; !ok {
					return fmt.Errorf("option '%s' not supported by config type '%s'", ref, cv.Type)
				}
			}
			for key, defOpt := range def.Options {
				if tOpt, ok := cv.TypeOpt[key]; ok {
					if defOpt.Inherit {
						if tOpt.DataType != cv.DataType {
							return fmt.Errorf("invalid data type for option '%s' of config type '%s'", key, cv.Type)
						}
					} else {
						if _, ok = defOpt.DataType[tOpt.DataType]; !ok {
							return fmt.Errorf("invalid data type for option '%s' of config type '%s'", key, cv.Type)
						}
					}
				} else if defOpt.Required {
					return fmt.Errorf("option '%s' is required by config type '%s'", key, cv.Type)
				}
			}
		}
	}
	return nil
}
