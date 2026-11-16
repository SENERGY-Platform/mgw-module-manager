/*
 * Copyright 2026 InfAI (CC SES)
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

package service

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"testing/fstest"

	module_lib "github.com/SENERGY-Platform/mgw-module-lib/model"
	lib_constants "github.com/SENERGY-Platform/mgw-module-manager/lib/constants"
	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	handler_jobs "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/jobs"
	helper_modfile "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/modfile"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	pkg_constants "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants"
	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

func coModule(mutate func(m *external_models.ModuleLibModule)) external_models.ModuleLibModule {
	m := external_models.ModuleLibModule{
		ID:      "mod-a",
		Version: "v2.0.0",
		Configs: map[string]module_lib.ConfigValue{},
		Inputs: module_lib.Inputs{
			Resources:  map[string]module_lib.Input{},
			Secrets:    map[string]module_lib.Input{},
			Configs:    map[string]module_lib.Input{},
			Files:      map[string]module_lib.Input{},
			FileGroups: map[string]module_lib.Input{},
		},
	}
	if mutate != nil {
		mutate(&m)
	}
	return m
}

func coTextConfig(required bool, def any) module_lib.ConfigValue {
	return module_lib.ConfigValue{Type: "text", DataType: module_lib.StringType, Required: required, Default: def}
}

func coStringValue(s string) pkg_models.Value {
	return pkg_models.Value{DataType: pkg_constants.ValueDataTypeString, String: s}
}

func coStringSliceValue(s ...string) pkg_models.Value {
	return pkg_models.Value{DataType: pkg_constants.ValueDataTypeString, IsSlice: true, StringSlice: s}
}

func coIntValue(i int64) pkg_models.Value {
	return pkg_models.Value{DataType: pkg_constants.ValueDataTypeInt64, Int64: i}
}

func coDeployment(mutate func(d *pkg_models.Deployment)) pkg_models.Deployment {
	d := pkg_models.Deployment{DeploymentBase: pkg_models.DeploymentBase{Id: "dep-1", ModuleId: "mod-a", ModuleVersion: "v1.0.0"}}
	if mutate != nil {
		mutate(&d)
	}
	return d
}

func coInput(mutate func(in *lib_models.DeploymentUserInput)) *lib_models.DeploymentUserInput {
	in := &lib_models.DeploymentUserInput{
		ModuleId:      "mod-a",
		HostResources: map[string]string{},
		Secrets:       map[string]string{},
		Configs:       map[string]interface{}{},
		GlobalConfigs: map[string]string{},
		Files:         map[string]string{},
		FileGroups:    map[string]map[string]lib_models.DeploymentFileGroupUserInput{},
	}
	if mutate != nil {
		mutate(in)
	}
	return in
}

func coMissing(kind, ref, reason string) []lib_models.CarryOverMissing {
	return []lib_models.CarryOverMissing{{Kind: kind, Ref: ref, Reason: reason}}
}

// coWithConfig declares one config input and returns the module for it.
func coWithConfig(cfg module_lib.ConfigValue) external_models.ModuleLibModule {
	return coModule(func(m *external_models.ModuleLibModule) {
		m.Configs["cfg"] = cfg
		m.Inputs.Configs["cfg"] = module_lib.Input{}
	})
}

func coWithStoredConfig(v pkg_models.Value) pkg_models.Deployment {
	return coDeployment(func(d *pkg_models.Deployment) {
		d.Configs = map[string]pkg_models.DeploymentUserConfig{"cfg": {Reference: "cfg", Value: v}}
	})
}

func coWithGlobalConfig(id string) pkg_models.Deployment {
	return coDeployment(func(d *pkg_models.Deployment) {
		d.GlobalConfigs = map[string]pkg_models.DeploymentGlobalConfig{"cfg": {Id: id, Reference: "cfg"}}
	})
}

func TestCarryOver(t *testing.T) {
	cases := []struct {
		name           string
		mod            external_models.ModuleLibModule
		fileDefaults   map[string]bool
		globalConfigs  map[string]pkg_models.Config
		secretTypes    map[string]string
		secretTypesErr error
		dep            pkg_models.Deployment
		want           *lib_models.DeploymentUserInput
		wantMissing    []lib_models.CarryOverMissing
	}{
		{
			name: "module without inputs is carried over unchanged",
			mod:  coModule(nil),
			dep:  coDeployment(nil),
			want: coInput(nil),
		},
		{
			name: "optional secret and host resource stay unanswered",
			mod: coModule(func(m *external_models.ModuleLibModule) {
				m.Secrets = map[string]module_lib.Secret{"cert": {Type: "certificate"}}
				m.HostResources = map[string]module_lib.HostResource{"dev": {}}
				m.Inputs.Secrets["cert"] = module_lib.Input{}
				m.Inputs.Resources["dev"] = module_lib.Input{}
			}),
			dep:  coDeployment(nil),
			want: coInput(nil),
		},
		{
			name: "required secret without stored id needs input",
			mod: coModule(func(m *external_models.ModuleLibModule) {
				m.Secrets = map[string]module_lib.Secret{"cert": {Resource: module_lib.Resource{Required: true}}}
				m.Inputs.Secrets["cert"] = module_lib.Input{}
			}),
			dep:         coDeployment(nil),
			wantMissing: coMissing(lib_constants.CarryOverKindSecret, "cert", lib_constants.CarryOverReasonNotSet),
		},
		{
			name: "secret input without definition needs input",
			mod: coModule(func(m *external_models.ModuleLibModule) {
				m.Inputs.Secrets["cert"] = module_lib.Input{}
			}),
			dep:         coDeployment(nil),
			wantMissing: coMissing(lib_constants.CarryOverKindSecret, "cert", lib_constants.CarryOverReasonNotDeclared),
		},
		{
			name: "stored secret of the declared type is carried over",
			mod: coModule(func(m *external_models.ModuleLibModule) {
				m.Secrets = map[string]module_lib.Secret{"cert": {Type: "certificate"}}
				m.Inputs.Secrets["cert"] = module_lib.Input{}
			}),
			secretTypes: map[string]string{"secret-1": "certificate"},
			dep: coDeployment(func(d *pkg_models.Deployment) {
				d.Secrets = map[string]pkg_models.DeploymentSecret{"cert": {Id: "secret-1"}}
			}),
			want: coInput(func(in *lib_models.DeploymentUserInput) { in.Secrets["cert"] = "secret-1" }),
		},
		{
			name: "stored secret of another type needs input",
			mod: coModule(func(m *external_models.ModuleLibModule) {
				m.Secrets = map[string]module_lib.Secret{"cert": {Type: "certificate"}}
				m.Inputs.Secrets["cert"] = module_lib.Input{}
			}),
			secretTypes: map[string]string{"secret-1": "basic-auth"},
			dep: coDeployment(func(d *pkg_models.Deployment) {
				d.Secrets = map[string]pkg_models.DeploymentSecret{"cert": {Id: "secret-1"}}
			}),
			wantMissing: coMissing(lib_constants.CarryOverKindSecret, "cert", lib_constants.CarryOverReasonTypeMismatch),
		},
		{
			name: "deleted stored secret needs input, also if optional",
			mod: coModule(func(m *external_models.ModuleLibModule) {
				m.Secrets = map[string]module_lib.Secret{"cert": {Type: "certificate"}}
				m.Inputs.Secrets["cert"] = module_lib.Input{}
			}),
			secretTypes: map[string]string{"secret-2": "certificate"},
			dep: coDeployment(func(d *pkg_models.Deployment) {
				d.Secrets = map[string]pkg_models.DeploymentSecret{"cert": {Id: "secret-1"}}
			}),
			wantMissing: coMissing(lib_constants.CarryOverKindSecret, "cert", lib_constants.CarryOverReasonSecretNotFound),
		},
		{
			name: "stored secret needs input if the secret manager could not be read",
			mod: coModule(func(m *external_models.ModuleLibModule) {
				m.Secrets = map[string]module_lib.Secret{"cert": {Type: "certificate"}}
				m.Inputs.Secrets["cert"] = module_lib.Input{}
			}),
			secretTypes:    map[string]string{"secret-1": "certificate"},
			secretTypesErr: errors.New("unreachable"),
			dep: coDeployment(func(d *pkg_models.Deployment) {
				d.Secrets = map[string]pkg_models.DeploymentSecret{"cert": {Id: "secret-1"}}
			}),
			wantMissing: coMissing(lib_constants.CarryOverKindSecret, "cert", lib_constants.CarryOverReasonSecretTypeUnknown),
		},
		{
			name: "required host resource without stored id needs input",
			mod: coModule(func(m *external_models.ModuleLibModule) {
				m.HostResources = map[string]module_lib.HostResource{"dev": {Resource: module_lib.Resource{Required: true}}}
				m.Inputs.Resources["dev"] = module_lib.Input{}
			}),
			dep:         coDeployment(nil),
			wantMissing: coMissing(lib_constants.CarryOverKindHostResource, "dev", lib_constants.CarryOverReasonNotSet),
		},
		{
			name: "every answered declaration is carried over",
			mod: coModule(func(m *external_models.ModuleLibModule) {
				m.Configs["port"] = module_lib.ConfigValue{Type: "number", DataType: module_lib.Int64Type, Required: true}
				m.Secrets = map[string]module_lib.Secret{"cert": {Resource: module_lib.Resource{Required: true}, Type: "certificate"}}
				m.HostResources = map[string]module_lib.HostResource{"dev": {Resource: module_lib.Resource{Required: true}}}
				m.Files = map[string]module_lib.File{"conf": {Required: true}}
				m.Inputs.Configs["port"] = module_lib.Input{}
				m.Inputs.Secrets["cert"] = module_lib.Input{}
				m.Inputs.Resources["dev"] = module_lib.Input{}
				m.Inputs.Files["conf"] = module_lib.Input{}
				m.Inputs.FileGroups["extra"] = module_lib.Input{}
			}),
			secretTypes: map[string]string{"secret-1": "certificate"},
			dep: coDeployment(func(d *pkg_models.Deployment) {
				d.Configs = map[string]pkg_models.DeploymentUserConfig{"port": {Value: coIntValue(8080)}}
				d.Secrets = map[string]pkg_models.DeploymentSecret{"cert": {Id: "secret-1"}}
				d.HostResources = map[string]pkg_models.DeploymentHostResource{"dev": {Id: "resource-1"}}
				d.Files = map[string]pkg_models.DeploymentFile{"conf": {Data: []byte("{}")}}
				d.FileGroups = map[string]pkg_models.DeploymentFileGroup{"extra": {Files: []pkg_models.DeploymentFileGroupFile{
					{Path: "a.yaml", Format: "yaml", Data: []byte("a")},
				}}}
			}),
			want: coInput(func(in *lib_models.DeploymentUserInput) {
				in.Configs["port"] = int64(8080)
				in.Secrets["cert"] = "secret-1"
				in.HostResources["dev"] = "resource-1"
				in.Files["conf"] = "e30="
				in.FileGroups["extra"] = map[string]lib_models.DeploymentFileGroupUserInput{"a.yaml": {Format: "yaml", Data: "YQ=="}}
			}),
		},
		{
			name:        "new required config without default needs input",
			mod:         coWithConfig(coTextConfig(true, nil)),
			dep:         coDeployment(nil),
			wantMissing: coMissing(lib_constants.CarryOverKindConfig, "cfg", lib_constants.CarryOverReasonNotSet),
		},
		{
			name: "new required config with default falls back to it",
			mod:  coWithConfig(coTextConfig(true, "info")),
			dep:  coDeployment(nil),
			want: coInput(nil),
		},
		{
			name: "new optional config without stored value is fine",
			mod:  coWithConfig(coTextConfig(false, nil)),
			dep:  coDeployment(nil),
			want: coInput(nil),
		},
		{
			name:        "required config holding the empty string needs input",
			mod:         coWithConfig(coTextConfig(true, "fallback")),
			dep:         coWithStoredConfig(coStringValue("")),
			wantMissing: coMissing(lib_constants.CarryOverKindConfig, "cfg", lib_constants.CarryOverReasonEmptyValue),
		},
		{
			name:        "optional config holding the empty string needs input",
			mod:         coWithConfig(coTextConfig(false, "fallback")),
			dep:         coWithStoredConfig(coStringValue("")),
			wantMissing: coMissing(lib_constants.CarryOverKindConfig, "cfg", lib_constants.CarryOverReasonEmptyValue),
		},
		{
			name: "empty stored list needs input",
			mod: coWithConfig(module_lib.ConfigValue{
				Type: "text", DataType: module_lib.StringType, IsSlice: true,
				Options: []string{"a", "b"}, Default: []string{"a"},
			}),
			dep:         coWithStoredConfig(coStringSliceValue()),
			wantMissing: coMissing(lib_constants.CarryOverKindConfig, "cfg", lib_constants.CarryOverReasonEmptyValue),
		},
		{
			name: "required list with an empty item needs input",
			mod: coWithConfig(module_lib.ConfigValue{
				Type: "text", DataType: module_lib.StringType, IsSlice: true, Required: true,
			}),
			dep:         coWithStoredConfig(coStringSliceValue("a", "")),
			wantMissing: coMissing(lib_constants.CarryOverKindConfig, "cfg", lib_constants.CarryOverReasonEmptyValue),
		},
		{
			name: "new data type needs input",
			mod:  coWithConfig(module_lib.ConfigValue{Type: "number", DataType: module_lib.Int64Type}),
			// the setup was created against the string version of the same config
			dep:         coWithStoredConfig(coStringValue("8080")),
			wantMissing: coMissing(lib_constants.CarryOverKindConfig, "cfg", lib_constants.CarryOverReasonTypeMismatch),
		},
		{
			name: "config turned into a list needs input",
			mod: coWithConfig(module_lib.ConfigValue{
				Type: "text", DataType: module_lib.StringType, IsSlice: true,
			}),
			dep:         coWithStoredConfig(coStringValue("a")),
			wantMissing: coMissing(lib_constants.CarryOverKindConfig, "cfg", lib_constants.CarryOverReasonTypeMismatch),
		},
		{
			name: "value outside the new options needs input",
			mod: coWithConfig(module_lib.ConfigValue{
				Type: "text", DataType: module_lib.StringType, Options: []string{"fast", "slow"},
			}),
			dep:         coWithStoredConfig(coStringValue("medium")),
			wantMissing: coMissing(lib_constants.CarryOverKindConfig, "cfg", lib_constants.CarryOverReasonInvalidValue),
		},
		{
			name: "value outside the options is carried if the new version allows it",
			mod: coWithConfig(module_lib.ConfigValue{
				Type: "text", DataType: module_lib.StringType, Options: []string{"fast", "slow"}, OptExt: true,
			}),
			dep: coWithStoredConfig(coStringValue("medium")),
			want: coInput(func(in *lib_models.DeploymentUserInput) {
				in.Configs["cfg"] = "medium"
			}),
		},
		{
			name: "value violating a constraint of the new version needs input",
			mod: coWithConfig(module_lib.ConfigValue{
				Type: "number", DataType: module_lib.Int64Type,
				TypeOpt: module_lib.ConfigTypeOptions{"min": {Value: int64(1024), DataType: module_lib.Int64Type}},
			}),
			dep:         coWithStoredConfig(coIntValue(80)),
			wantMissing: coMissing(lib_constants.CarryOverKindConfig, "cfg", lib_constants.CarryOverReasonInvalidValue),
		},
		{
			name: "every item of a list is validated",
			mod: coWithConfig(module_lib.ConfigValue{
				Type: "text", DataType: module_lib.StringType, IsSlice: true, Options: []string{"fast", "slow"},
			}),
			dep:         coWithStoredConfig(coStringSliceValue("fast", "medium")),
			wantMissing: coMissing(lib_constants.CarryOverKindConfig, "cfg", lib_constants.CarryOverReasonInvalidValue),
		},
		{
			name: "valid list is carried as a slice",
			mod: coWithConfig(module_lib.ConfigValue{
				Type: "text", DataType: module_lib.StringType, IsSlice: true, Options: []string{"fast", "slow"},
			}),
			dep: coWithStoredConfig(coStringSliceValue("fast", "slow")),
			want: coInput(func(in *lib_models.DeploymentUserInput) {
				in.Configs["cfg"] = []string{"fast", "slow"}
			}),
		},
		{
			name:          "global config reference is kept",
			mod:           coWithConfig(coTextConfig(true, nil)),
			globalConfigs: map[string]pkg_models.Config{"gc-1": {Id: "gc-1", Value: coStringValue("x")}},
			dep:           coWithGlobalConfig("gc-1"),
			want: coInput(func(in *lib_models.DeploymentUserInput) {
				in.GlobalConfigs["cfg"] = "gc-1"
			}),
		},
		{
			name:        "deleted global config needs input",
			mod:         coWithConfig(coTextConfig(false, "def")),
			dep:         coWithGlobalConfig("gc-1"),
			wantMissing: coMissing(lib_constants.CarryOverKindConfig, "cfg", lib_constants.CarryOverReasonGlobalConfigNotFound),
		},
		{
			name: "global config violating the new options needs input",
			mod: coWithConfig(module_lib.ConfigValue{
				Type: "text", DataType: module_lib.StringType, Required: true, Options: []string{"fast", "slow"},
			}),
			globalConfigs: map[string]pkg_models.Config{"gc-1": {Id: "gc-1", Value: coStringValue("medium")}},
			dep:           coWithGlobalConfig("gc-1"),
			wantMissing:   coMissing(lib_constants.CarryOverKindConfig, "cfg", lib_constants.CarryOverReasonInvalidValue),
		},
		{
			name:          "global config of another data type needs input",
			mod:           coWithConfig(coTextConfig(true, nil)),
			globalConfigs: map[string]pkg_models.Config{"gc-1": {Id: "gc-1", Value: coIntValue(1)}},
			dep:           coWithGlobalConfig("gc-1"),
			wantMissing:   coMissing(lib_constants.CarryOverKindConfig, "cfg", lib_constants.CarryOverReasonTypeMismatch),
		},
		{
			name: "config the new version no longer declares is dropped",
			mod:  coModule(nil),
			dep:  coWithStoredConfig(coStringValue("x")),
			want: coInput(nil),
		},
		{
			name: "config input without definition needs input",
			mod: coModule(func(m *external_models.ModuleLibModule) {
				m.Inputs.Configs["orphan"] = module_lib.Input{}
			}),
			dep:         coDeployment(nil),
			wantMissing: coMissing(lib_constants.CarryOverKindConfig, "orphan", lib_constants.CarryOverReasonNotDeclared),
		},
		{
			name: "file group without files is skipped",
			mod: coModule(func(m *external_models.ModuleLibModule) {
				m.Inputs.FileGroups["extra"] = module_lib.Input{}
			}),
			dep:  coDeployment(nil),
			want: coInput(nil),
		},
		{
			name: "file group file without path needs input",
			mod: coModule(func(m *external_models.ModuleLibModule) {
				m.Inputs.FileGroups["extra"] = module_lib.Input{}
			}),
			dep: coDeployment(func(d *pkg_models.Deployment) {
				d.FileGroups = map[string]pkg_models.DeploymentFileGroup{"extra": {Files: []pkg_models.DeploymentFileGroupFile{
					{Path: "a.yaml", Data: []byte("a")}, {Data: []byte("b")},
				}}}
			}),
			wantMissing: coMissing(lib_constants.CarryOverKindFileGroup, "extra", lib_constants.CarryOverReasonMissingPath),
		},
		{
			name: "stored empty file needs input",
			mod: coModule(func(m *external_models.ModuleLibModule) {
				m.Files = map[string]module_lib.File{"conf": {}}
				m.Inputs.Files["conf"] = module_lib.Input{}
			}),
			fileDefaults: map[string]bool{"conf": true},
			dep: coDeployment(func(d *pkg_models.Deployment) {
				d.Files = map[string]pkg_models.DeploymentFile{"conf": {Data: []byte{}}}
			}),
			wantMissing: coMissing(lib_constants.CarryOverKindFile, "conf", lib_constants.CarryOverReasonEmptyValue),
		},
		{
			name: "required file without default needs input",
			mod: coModule(func(m *external_models.ModuleLibModule) {
				m.Files = map[string]module_lib.File{"conf": {Required: true}}
				m.Inputs.Files["conf"] = module_lib.Input{}
			}),
			dep:         coDeployment(nil),
			wantMissing: coMissing(lib_constants.CarryOverKindFile, "conf", lib_constants.CarryOverReasonNotSet),
		},
		{
			name: "file without stored content falls back to the default",
			mod: coModule(func(m *external_models.ModuleLibModule) {
				m.Files = map[string]module_lib.File{"conf": {Required: true}}
				m.Inputs.Files["conf"] = module_lib.Input{}
			}),
			fileDefaults: map[string]bool{"conf": true},
			dep:          coDeployment(nil),
			want:         coInput(nil),
		},
		{
			name: "file input without definition needs input",
			mod: coModule(func(m *external_models.ModuleLibModule) {
				m.Inputs.Files["conf"] = module_lib.Input{}
			}),
			dep:         coDeployment(nil),
			wantMissing: coMissing(lib_constants.CarryOverKindFile, "conf", lib_constants.CarryOverReasonNotDeclared),
		},
		{
			name: "all open items are reported in a stable order",
			mod: coModule(func(m *external_models.ModuleLibModule) {
				m.Configs["b"] = coTextConfig(true, nil)
				m.Configs["a"] = coTextConfig(true, nil)
				m.Secrets = map[string]module_lib.Secret{"s": {Resource: module_lib.Resource{Required: true}}}
				m.Inputs.Configs["b"] = module_lib.Input{}
				m.Inputs.Configs["a"] = module_lib.Input{}
				m.Inputs.Secrets["s"] = module_lib.Input{}
			}),
			dep: coDeployment(nil),
			wantMissing: []lib_models.CarryOverMissing{
				{Kind: lib_constants.CarryOverKindConfig, Ref: "a", Reason: lib_constants.CarryOverReasonNotSet},
				{Kind: lib_constants.CarryOverKindConfig, Ref: "b", Reason: lib_constants.CarryOverReasonNotSet},
				{Kind: lib_constants.CarryOverKindSecret, Ref: "s", Reason: lib_constants.CarryOverReasonNotSet},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			refs := carryOverRefs{globalConfigs: tc.globalConfigs, secretTypes: tc.secretTypes, secretTypesErr: tc.secretTypesErr}
			input, missing, err := carryOver(tc.mod, tc.fileDefaults, tc.dep, refs)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(missing, tc.wantMissing) {
				t.Fatalf("missing: want %+v, got %+v", tc.wantMissing, missing)
			}
			if tc.want != nil && !reflect.DeepEqual(input, *tc.want) {
				t.Errorf("input: want %+v, got %+v", *tc.want, input)
			}
		})
	}
}

func TestCarryOver_DeploymentError(t *testing.T) {
	dep := coDeployment(func(d *pkg_models.Deployment) { d.Err = errors.New("boom") })
	if _, _, err := carryOver(coModule(nil), nil, dep, carryOverRefs{}); err == nil {
		t.Error("expected an error for a deployment carrying an error")
	}
}

func TestCarryOverInstalled(t *testing.T) {
	t.Run("module error is returned", func(t *testing.T) {
		module := pkg_models.Module{ModuleLibModule: coModule(nil), Err: errors.New("boom")}
		if _, _, err := carryOverInstalled(module, coDeployment(nil), carryOverRefs{}); err == nil {
			t.Error("expected an error for a module carrying an error")
		}
	})
	fileMod := coModule(func(m *external_models.ModuleLibModule) {
		m.Files = map[string]module_lib.File{"conf": {Required: true}}
		m.Inputs.Files["conf"] = module_lib.Input{}
	})
	t.Run("default data of an installed file answers a required file", func(t *testing.T) {
		module := pkg_models.Module{
			ModuleLibModule: fileMod,
			Files:           map[string]pkg_models.ModuleFile{"conf": {DefaultData: []byte("x")}},
		}
		_, missing, err := carryOverInstalled(module, coDeployment(nil), carryOverRefs{})
		if err != nil || len(missing) != 0 {
			t.Errorf("expected carry over, got missing=%+v err=%v", missing, err)
		}
	})
	t.Run("required file without default data needs input", func(t *testing.T) {
		module := pkg_models.Module{
			ModuleLibModule: fileMod,
			Files:           map[string]pkg_models.ModuleFile{"conf": {}},
		}
		_, missing, err := carryOverInstalled(module, coDeployment(nil), carryOverRefs{})
		if err != nil || len(missing) != 1 {
			t.Errorf("expected one missing file, got missing=%+v err=%v", missing, err)
		}
	})
}

func TestCarryOverRepository(t *testing.T) {
	mod := coModule(func(m *external_models.ModuleLibModule) {
		m.Files = map[string]module_lib.File{
			"conf":  {Required: true, Source: "conf.json"},
			"empty": {Required: true, Source: "empty.json"},
			"other": {Required: true},
		}
		m.Inputs.Files["conf"] = module_lib.Input{}
		m.Inputs.Files["empty"] = module_lib.Input{}
		m.Inputs.Files["other"] = module_lib.Input{}
	})
	fSys := fstest.MapFS{"conf.json": {Data: []byte("{}")}, "empty.json": {Data: []byte{}}}

	t.Run("only a non-empty source file is a default, as after the installation", func(t *testing.T) {
		_, missing, err := carryOverRepository(modWrapper{Mod: mod, FS: fSys}, coDeployment(nil), carryOverRefs{})
		if err != nil {
			t.Fatal(err)
		}
		want := []lib_models.CarryOverMissing{
			{Kind: lib_constants.CarryOverKindFile, Ref: "empty", Reason: lib_constants.CarryOverReasonNotSet},
			{Kind: lib_constants.CarryOverKindFile, Ref: "other", Reason: lib_constants.CarryOverReasonNotSet},
		}
		if !reflect.DeepEqual(missing, want) {
			t.Errorf("want %+v, got %+v", want, missing)
		}
		files, err := helper_modfile.GetFiles(fSys, mod.Files)
		if err != nil {
			t.Fatal(err)
		}
		_, installedMissing, err := carryOverInstalled(pkg_models.Module{ModuleLibModule: mod, Files: files}, coDeployment(nil), carryOverRefs{})
		if err != nil || !reflect.DeepEqual(installedMissing, want) {
			t.Errorf("installed check disagrees: missing=%+v err=%v", installedMissing, err)
		}
	})

	t.Run("unreadable source file fails the check", func(t *testing.T) {
		if _, _, err := carryOverRepository(modWrapper{Mod: mod, FS: fstest.MapFS{}}, coDeployment(nil), carryOverRefs{}); err == nil {
			t.Error("expected an error for a source file missing in the repository")
		}
	})
}

// The carried input must pass the validation of the deployment create and update path unchanged.
func TestCarryOver_AcceptedByGetUserInputs(t *testing.T) {
	mod := coModule(func(m *external_models.ModuleLibModule) {
		m.Configs["port"] = module_lib.ConfigValue{Type: "number", DataType: module_lib.Int64Type, Required: true}
		m.Configs["modes"] = module_lib.ConfigValue{
			Type: "text", DataType: module_lib.StringType, IsSlice: true, Options: []string{"fast", "slow"},
		}
		m.Files = map[string]module_lib.File{"conf": {}}
		m.Inputs.Configs["port"] = module_lib.Input{}
		m.Inputs.Configs["modes"] = module_lib.Input{}
		m.Inputs.Files["conf"] = module_lib.Input{}
		m.Inputs.FileGroups["extra"] = module_lib.Input{}
	})
	dep := coDeployment(func(d *pkg_models.Deployment) {
		d.Configs = map[string]pkg_models.DeploymentUserConfig{
			"port":  {Value: coIntValue(8080)},
			"modes": {Value: coStringSliceValue("fast", "slow")},
		}
		d.Files = map[string]pkg_models.DeploymentFile{"conf": {Data: []byte("{}")}}
		d.FileGroups = map[string]pkg_models.DeploymentFileGroup{"extra": {Files: []pkg_models.DeploymentFileGroupFile{
			{Path: "a.yaml", Format: "yaml", Data: []byte("a")},
		}}}
	})
	input, missing, err := carryOver(mod, nil, dep, carryOverRefs{})
	if err != nil || len(missing) != 0 {
		t.Fatalf("expected carry over, got missing=%+v err=%v", missing, err)
	}
	module := pkg_models.Module{ModuleLibModule: mod}
	got, err := getUserInputs(
		[]lib_models.DeploymentUserInput{input},
		map[string]pkg_models.Module{"mod-a": module},
	)
	if err != nil {
		t.Fatalf("getUserInputs rejected the carried input: %v", err)
	}
	userInput := got["mod-a"]
	if userInput.Configs["port"].Int64 != 8080 ||
		!reflect.DeepEqual(userInput.Configs["modes"].StringSlice, []string{"fast", "slow"}) ||
		string(userInput.Files["conf"]) != "{}" ||
		string(userInput.FileGroups["extra"]["a.yaml"].Data) != "a" {
		t.Errorf("unexpected user input %+v", userInput)
	}
}

type carryOverTestModulesHandler struct {
	modulesHandler
	modules map[string]pkg_models.Module
}

func (m *carryOverTestModulesHandler) GetModules(_ context.Context, filter pkg_models.ModulesFilterWithName, _ bool) (map[string]pkg_models.Module, error) {
	res := make(map[string]pkg_models.Module)
	for _, id := range filter.Ids {
		if mod, ok := m.modules[id]; ok {
			res[id] = mod
		}
	}
	return res, nil
}

type carryOverTestDeploymentsHandler struct {
	deploymentsHandler
	deployments      map[string]pkg_models.Deployment
	secretTypes      map[string]string
	secretTypesErr   error
	secretTypesCalls int
	onSecretTypes    func()
}

func (m *carryOverTestDeploymentsHandler) GetSecretTypes(_ context.Context) (map[string]string, error) {
	if m.onSecretTypes != nil {
		m.onSecretTypes()
	}
	m.secretTypesCalls++
	if m.secretTypesErr != nil {
		return nil, m.secretTypesErr
	}
	return m.secretTypes, nil
}

func (m *carryOverTestDeploymentsHandler) GetDeploymentsByModuleIds(_ context.Context, filter pkg_models.DeploymentsFilterWithState) (map[string]pkg_models.Deployment, error) {
	res := make(map[string]pkg_models.Deployment)
	for _, id := range filter.ModuleIds {
		if dep, ok := m.deployments[id]; ok {
			res[id] = dep
		}
	}
	return res, nil
}

func TestService_GetDeploymentsCarryOver(t *testing.T) {
	needsInput := coWithConfig(coTextConfig(true, nil))
	needsInput.ID = "mod-open"
	plain := coModule(nil)
	plain.ID = "mod-ok"
	broken := coModule(nil)
	broken.ID = "mod-err"
	notDeployed := coModule(nil)
	notDeployed.ID = "mod-undeployed"
	modHdl := &carryOverTestModulesHandler{modules: map[string]pkg_models.Module{
		"mod-open":       {ModuleLibModule: needsInput},
		"mod-ok":         {ModuleLibModule: plain},
		"mod-err":        {ModuleLibModule: broken},
		"mod-undeployed": {ModuleLibModule: notDeployed},
	}}
	depHdl := &carryOverTestDeploymentsHandler{deployments: map[string]pkg_models.Deployment{
		"mod-open": coDeployment(nil),
		"mod-ok":   coDeployment(nil),
		"mod-err":  coDeployment(func(d *pkg_models.Deployment) { d.Err = errors.New("boom") }),
	}}
	srv := New(nil, modHdl, depHdl, nil, nil, nil, nil, handler_jobs.New(context.Background(), handler_jobs.Config{}), nil)

	res, err := srv.GetDeploymentsCarryOver(context.Background(), []string{"mod-unknown", "mod-undeployed", "mod-err", "mod-open", "mod-ok"})
	if err != nil {
		t.Fatal(err)
	}
	byId := make(map[string]lib_models.DeploymentCarryOver)
	for _, item := range res {
		byId[item.ModuleId] = item
	}
	if len(res) != 5 {
		t.Fatalf("expected 5 results, got %+v", res)
	}
	if ok := byId["mod-ok"]; ok.HasError || ok.Input == nil || ok.Input.ModuleId != "mod-ok" || len(ok.Missing) != 0 {
		t.Errorf("mod-ok: unexpected %+v", ok)
	}
	if open := byId["mod-open"]; open.HasError || open.Input != nil || len(open.Missing) != 1 || open.Missing[0].Ref != "cfg" {
		t.Errorf("mod-open: unexpected %+v", open)
	}
	for _, id := range []string{"mod-unknown", "mod-undeployed", "mod-err"} {
		if item := byId[id]; !item.HasError || item.Input != nil {
			t.Errorf("%s: expected error result, got %+v", id, item)
		}
	}
	if res, err := srv.GetDeploymentsCarryOver(context.Background(), nil); err != nil || len(res) != 0 {
		t.Errorf("empty filter: expected no results, got %+v err=%v", res, err)
	}
}

type carryOverTestGlobalConfigsHandler struct {
	globalConfigsHandler
	configs map[string]pkg_models.Config
	calls   [][]string
}

func (m *carryOverTestGlobalConfigsHandler) GetGlobalConfigs(_ context.Context, ids []string) (map[string]pkg_models.Config, error) {
	m.calls = append(m.calls, ids)
	res := make(map[string]pkg_models.Config)
	for _, id := range ids {
		if cfg, ok := m.configs[id]; ok {
			res[id] = cfg
		}
	}
	return res, nil
}

func TestService_GetDeploymentsCarryOver_GlobalConfigs(t *testing.T) {
	mod := coWithConfig(module_lib.ConfigValue{Type: "text", DataType: module_lib.StringType, Required: true, Options: []string{"a", "b"}})
	modules := make(map[string]pkg_models.Module)
	for _, id := range []string{"mod-valid", "mod-invalid", "mod-deleted"} {
		m := mod
		m.ID = id
		modules[id] = pkg_models.Module{ModuleLibModule: m}
	}
	depHdl := &carryOverTestDeploymentsHandler{deployments: map[string]pkg_models.Deployment{
		"mod-valid":   coWithGlobalConfig("gc-a"),
		"mod-invalid": coWithGlobalConfig("gc-c"),
		"mod-deleted": coWithGlobalConfig("gc-gone"),
	}}
	gcHdl := &carryOverTestGlobalConfigsHandler{configs: map[string]pkg_models.Config{
		"gc-a": {Id: "gc-a", Value: coStringValue("a")},
		"gc-c": {Id: "gc-c", Value: coStringValue("c")},
	}}
	srv := New(nil, &carryOverTestModulesHandler{modules: modules}, depHdl, nil, gcHdl, nil, nil, handler_jobs.New(context.Background(), handler_jobs.Config{}), nil)

	res, err := srv.GetDeploymentsCarryOver(context.Background(), []string{"mod-valid", "mod-invalid", "mod-deleted"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"mod-valid":   "",
		"mod-invalid": lib_constants.CarryOverReasonInvalidValue,
		"mod-deleted": lib_constants.CarryOverReasonGlobalConfigNotFound,
	}
	for _, item := range res {
		var reason string
		if len(item.Missing) > 0 {
			reason = item.Missing[0].Reason
		}
		if item.HasError || reason != want[item.ModuleId] || (reason == "") != (item.Input != nil) {
			t.Errorf("%s: unexpected %+v", item.ModuleId, item)
		}
	}
	if len(res) != 3 {
		t.Errorf("expected 3 results, got %+v", res)
	}
	if !reflect.DeepEqual(gcHdl.calls, [][]string{{"gc-a", "gc-c", "gc-gone"}}) {
		t.Errorf("expected one read of the referenced global configs, got %v", gcHdl.calls)
	}
}

func TestService_GetDeploymentsCarryOver_Secrets(t *testing.T) {
	mod := coModule(func(m *external_models.ModuleLibModule) {
		m.Secrets = map[string]module_lib.Secret{"cert": {Type: "certificate"}}
		m.Inputs.Secrets["cert"] = module_lib.Input{}
	})
	modules := make(map[string]pkg_models.Module)
	for _, id := range []string{"mod-same", "mod-other", "mod-gone"} {
		m := mod
		m.ID = id
		modules[id] = pkg_models.Module{ModuleLibModule: m}
	}
	withSecret := func(id string) pkg_models.Deployment {
		return coDeployment(func(d *pkg_models.Deployment) {
			d.Secrets = map[string]pkg_models.DeploymentSecret{"cert": {Id: id}}
		})
	}
	newSrv := func(depHdl *carryOverTestDeploymentsHandler) *Service {
		return New(nil, &carryOverTestModulesHandler{modules: modules}, depHdl, nil, nil, nil, nil, handler_jobs.New(context.Background(), handler_jobs.Config{}), nil)
	}
	reasons := func(t *testing.T, res []lib_models.DeploymentCarryOver) map[string]string {
		t.Helper()
		got := make(map[string]string)
		for _, item := range res {
			if item.HasError {
				t.Errorf("%s: unexpected error %s", item.ModuleId, item.ErrorMsg)
			}
			if len(item.Missing) > 0 {
				got[item.ModuleId] = item.Missing[0].Reason
			} else if item.Input == nil {
				t.Errorf("%s: neither input nor missing", item.ModuleId)
			}
		}
		return got
	}
	ids := []string{"mod-same", "mod-other", "mod-gone"}
	deployments := map[string]pkg_models.Deployment{
		"mod-same":  withSecret("s-cert"),
		"mod-other": withSecret("s-auth"),
		"mod-gone":  withSecret("s-deleted"),
	}

	t.Run("type compared with the secret manager", func(t *testing.T) {
		depHdl := &carryOverTestDeploymentsHandler{
			deployments: deployments,
			secretTypes: map[string]string{"s-cert": "certificate", "s-auth": "basic-auth"},
		}
		res, err := newSrv(depHdl).GetDeploymentsCarryOver(context.Background(), ids)
		if err != nil {
			t.Fatal(err)
		}
		want := map[string]string{
			"mod-other": lib_constants.CarryOverReasonTypeMismatch,
			"mod-gone":  lib_constants.CarryOverReasonSecretNotFound,
		}
		if got := reasons(t, res); !reflect.DeepEqual(got, want) {
			t.Errorf("want %v, got %v", want, got)
		}
		if depHdl.secretTypesCalls != 1 {
			t.Errorf("expected one read of the secret types, got %d", depHdl.secretTypesCalls)
		}
	})

	t.Run("secret manager unreachable", func(t *testing.T) {
		depHdl := &carryOverTestDeploymentsHandler{deployments: deployments, secretTypesErr: errors.New("connection refused")}
		res, err := newSrv(depHdl).GetDeploymentsCarryOver(context.Background(), ids)
		if err != nil {
			t.Fatal(err)
		}
		want := map[string]string{
			"mod-same":  lib_constants.CarryOverReasonSecretTypeUnknown,
			"mod-other": lib_constants.CarryOverReasonSecretTypeUnknown,
			"mod-gone":  lib_constants.CarryOverReasonSecretTypeUnknown,
		}
		if got := reasons(t, res); !reflect.DeepEqual(got, want) {
			t.Errorf("want %v, got %v", want, got)
		}
	})

	t.Run("secret manager asked without the service lock", func(t *testing.T) {
		depHdl := &carryOverTestDeploymentsHandler{
			deployments: deployments,
			secretTypes: map[string]string{"s-cert": "certificate"},
		}
		srv := newSrv(depHdl)
		underLock := false
		depHdl.onSecretTypes = func() { underLock = underLock || auHeldLock(srv) }
		if _, err := srv.GetDeploymentsCarryOver(context.Background(), ids); err != nil {
			t.Fatal(err)
		}
		if underLock || depHdl.secretTypesCalls != 1 {
			t.Errorf("secret types read %d times, under the service lock: %v", depHdl.secretTypesCalls, underLock)
		}
	})

	t.Run("secret manager not asked without stored secrets", func(t *testing.T) {
		depHdl := &carryOverTestDeploymentsHandler{
			deployments:    map[string]pkg_models.Deployment{"mod-same": coDeployment(nil)},
			secretTypesErr: errors.New("connection refused"),
		}
		res, err := newSrv(depHdl).GetDeploymentsCarryOver(context.Background(), []string{"mod-same"})
		if err != nil {
			t.Fatal(err)
		}
		if got := reasons(t, res); len(got) != 0 || depHdl.secretTypesCalls != 0 {
			t.Errorf("unexpected missing %v, %d reads of the secret types", got, depHdl.secretTypesCalls)
		}
	})
}
