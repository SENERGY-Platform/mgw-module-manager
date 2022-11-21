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
	"golang.org/x/mod/semver"
	"regexp"
)

func Validate(m Module) error {
	if !IsValidModuleID(m.ID) {
		return fmt.Errorf("invalid module ID format '%s'", m.ID)
	}
	if !IsValidSemVer(m.Version) {
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
	if m.Dependencies != nil && len(m.Dependencies) > 0 {
		for mid, dependency := range m.Dependencies {
			if !IsValidModuleID(mid) {
				return fmt.Errorf("invalid dependency module ID format '%s'", mid)
			}
			if err := ValidateSemVerRange(dependency.Version); err != nil {
				return fmt.Errorf("dependency '%s': %s", mid, err)
			}
		}
	}
	if m.Resources != nil && len(m.Resources) > 0 {
		for ref := range m.Resources {
			if ref == "" {
				return errors.New("invalid resource reference")
			}
		}
	}
	if m.Secrets != nil && len(m.Secrets) > 0 {
		for ref := range m.Secrets {
			if ref == "" {
				return errors.New("invalid secret reference")
			}
		}
	}
	if m.UserInput.Groups != nil && len(m.UserInput.Groups) > 0 {
		for ref := range m.UserInput.Groups {
			if ref == "" {
				return errors.New("invalid user input group reference")
			}
		}
	}
	if m.UserInput.Resources != nil && len(m.UserInput.Resources) > 0 {
		if m.Resources == nil || len(m.Resources) == 0 {
			return errors.New("missing resources for user inputs")
		}
		for ref, input := range m.UserInput.Resources {
			if ref == "" {
				return errors.New("invalid user input reference")
			}
			if _, ok := m.Resources[ref]; !ok {
				return fmt.Errorf("missing resource for input '%s'", ref)
			}
			if input.Group != nil {
				if m.UserInput.Groups == nil || len(m.UserInput.Groups) == 0 {
					return errors.New("missing groups for user inputs")
				}
				if _, ok := m.UserInput.Groups[*input.Group]; !ok {
					return fmt.Errorf("missing group for input '%s'", ref)
				}
			}
		}
	}
	if m.UserInput.Secrets != nil && len(m.UserInput.Secrets) > 0 {
		if m.Secrets == nil || len(m.Secrets) == 0 {
			return errors.New("missing secrets for user inputs")
		}
		for ref, input := range m.UserInput.Secrets {
			if ref == "" {
				return errors.New("invalid user input reference")
			}
			if _, ok := m.Secrets[ref]; !ok {
				return fmt.Errorf("missing secret for input '%s'", ref)
			}
			if input.Group != nil {
				if m.UserInput.Groups == nil || len(m.UserInput.Groups) == 0 {
					return errors.New("missing groups for user inputs")
				}
				if _, ok := m.UserInput.Groups[*input.Group]; !ok {
					return fmt.Errorf("missing group for input '%s'", ref)
				}
			}
		}
	}
	if m.UserInput.Configs != nil && len(m.UserInput.Configs) > 0 {
		if m.Configs == nil || len(m.Configs) == 0 {
			return errors.New("missing secrets for user inputs")
		}
		for ref, input := range m.UserInput.Configs {
			if ref == "" {
				return errors.New("invalid user input reference")
			}
			if _, ok := m.Configs[ref]; !ok {
				return fmt.Errorf("missing secret for input '%s'", ref)
			}
			if input.Group != nil {
				if m.UserInput.Groups == nil || len(m.UserInput.Groups) == 0 {
					return errors.New("missing groups for user inputs")
				}
				if _, ok := m.UserInput.Groups[*input.Group]; !ok {
					return fmt.Errorf("missing group for input '%s'", ref)
				}
			}
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
		if service.Volumes != nil && len(service.Volumes) > 0 {
			for _, volume := range service.Volumes {
				if _, ok := m.Volumes[volume]; !ok {
					return fmt.Errorf("invalid service volume: '%s' -> '%s'", ref, volume)
				}
			}
		}
		if service.Resources != nil && len(service.Resources) > 0 {
			for _, target := range service.Resources {
				if _, ok := m.Resources[target.Ref]; !ok {
					return fmt.Errorf("invalid service resource: '%s' -> '%s'", ref, target.Ref)
				}
			}
		}
		if service.Secrets != nil && len(service.Secrets) > 0 {
			for _, secretRef := range service.Secrets {
				if _, ok := m.Secrets[secretRef]; !ok {
					return fmt.Errorf("invalid service secret: '%s' -> '%s'", ref, secretRef)
				}
			}
		}
		if service.Configs != nil && len(service.Configs) > 0 {
			for _, confRef := range service.Configs {
				if _, ok := m.Configs[confRef]; !ok {
					return fmt.Errorf("invalid service secret: '%s' -> '%s'", ref, confRef)
				}
			}
		}
		if service.HttpEndpoints != nil && len(service.HttpEndpoints) > 0 {
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
		if service.Dependencies != nil && len(service.Dependencies) > 0 {
			for _, target := range service.Dependencies {
				if _, ok := m.Services[target.Service]; !ok {
					return fmt.Errorf("invalid service dependency: '%s' -> '%s'", ref, target.Service)
				}
				if !IsValidSrvDepCondition(target.Condition) {
					return fmt.Errorf("invalid service dependency condition: '%s' -> '%s'", ref, target.Condition)
				}
			}
		}
		if service.ExternalDependencies != nil && len(service.ExternalDependencies) > 0 {
			for _, target := range service.ExternalDependencies {
				if !IsValidModuleID(target.ID) {
					return fmt.Errorf("invalid service external dependency: '%s' -> '%s'", ref, target.ID)
				}
			}
		}
		if service.PortMappings != nil && len(service.PortMappings) > 0 {
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

func validateServiceRunConfig(rc RunConfig) error {
	if !IsValidRestartStrategy(rc.RestartStrategy) {
		return fmt.Errorf("invalid restart strategy '%s'", rc.RestartStrategy)
	}
	if rc.RestartStrategy == RestartOnFail && rc.Retries == nil {
		return errors.New("missing retries")
	}
	if rc.RestartStrategy == RestartAlways && rc.RemoveAfterRun {
		return fmt.Errorf("remove after run and restart strategy '%s'", rc.RestartStrategy)
	}
	return nil
}

func validateServiceRefVars(service *Service) error {
	refVars := make(map[string]string)
	if service.Configs != nil && len(service.Configs) > 0 {
		for rv := range service.Configs {
			refVars[rv] = "configs"
		}
	}
	if service.Dependencies != nil && len(service.Dependencies) > 0 {
		for rv := range service.Dependencies {
			if v, ok := refVars[rv]; ok {
				return fmt.Errorf("'%s' -> '%s' & '%s'", rv, v, "dependencies")
			}
			refVars[rv] = "dependencies"
		}
	}
	if service.ExternalDependencies != nil && len(service.ExternalDependencies) > 0 {
		for rv := range service.ExternalDependencies {
			if v, ok := refVars[rv]; ok {
				return fmt.Errorf("'%s' -> '%s' & '%s'", rv, v, "external dependencies")
			}
			refVars[rv] = "external dependencies"
		}
	}
	return nil
}

func validateServiceMountPoints(service *Service) error {
	mountPoints := make(map[string]string)
	if service.Include != nil && len(service.Include) > 0 {
		for mp := range service.Include {
			mountPoints[mp] = "include"
		}
	}
	if service.Tmpfs != nil && len(service.Tmpfs) > 0 {
		for mp := range service.Tmpfs {
			if v, ok := mountPoints[mp]; ok {
				return fmt.Errorf("'%s' -> '%s' & '%s'", mp, v, "tmpfs")
			}
			mountPoints[mp] = "tmpfs"
		}
	}
	if service.Volumes != nil && len(service.Volumes) > 0 {
		for mp := range service.Volumes {
			if v, ok := mountPoints[mp]; ok {
				return fmt.Errorf("'%s' -> '%s' & '%s'", mp, v, "volumes")
			}
			mountPoints[mp] = "volumes"
		}
	}
	if service.Resources != nil && len(service.Resources) > 0 {
		for mp := range service.Resources {
			if v, ok := mountPoints[mp]; ok {
				return fmt.Errorf("'%s' -> '%s' & '%s'", mp, v, "resources")
			}
			mountPoints[mp] = "resources"
		}
	}
	if service.Secrets != nil && len(service.Secrets) > 0 {
		for mp := range service.Secrets {
			if v, ok := mountPoints[mp]; ok {
				return fmt.Errorf("'%s' -> '%s' & '%s'", mp, v, "secrets")
			}
			mountPoints[mp] = "secrets"
		}
	}
	return nil
}

func IsValidPort(p []uint) bool {
	return !(p == nil || len(p) == 0 || len(p) > 2 || (len(p) > 1 && p[0] == p[1]) || (len(p) > 1 && p[1] < p[0]))
}

func IsValidModuleType(s string) bool {
	_, ok := ModuleTypeMap[s]
	return ok
}

func IsValidDeploymentType(s string) bool {
	_, ok := DeploymentTypeMap[s]
	return ok
}

func IsValidModuleID(s string) bool {
	re := regexp.MustCompile(`^([a-z0-9A-Z-_.]+)(:\d+)?([\/a-zA-Z0-9-\.]+)?$`)
	return re.MatchString(s)
}

func IsValidSrvDepCondition(s string) bool {
	_, ok := SrvDepConditionMap[s]
	return ok
}

func IsValidRestartStrategy(s string) bool {
	_, ok := RestartStrategyMap[s]
	return ok
}

func IsValidPortType(s string) bool {
	_, ok := PortTypeMap[s]
	return ok
}

func IsValidSemVer(s string) bool {
	return semver.IsValid(s)
}

func IsValidOperator(s string) bool {
	_, ok := OperatorMap[s]
	return ok
}

func ValidateSemVerRange(s string) error {
	_, _, err := semVerRangeParse(s)
	return err
}
