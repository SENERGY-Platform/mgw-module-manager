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

package validation

import (
	"encoding/json"
	"fmt"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"module-manager/itf"
	"module-manager/model"
	"os"
	"regexp"
	"strings"
)

type ConfigValidationHandler struct {
	definitions map[string]model.ConfigDefinition
	validators  map[string]itf.Validator
}

func NewConfigValidationHandler(definitions map[string]model.ConfigDefinition, validators map[string]itf.Validator) (*ConfigValidationHandler, error) {
	if err := validateDefs(definitions, validators); err != nil {
		return nil, err
	}
	return &ConfigValidationHandler{definitions: definitions, validators: validators}, nil
}

func (h *ConfigValidationHandler) ValidateBase(cType string, cTypeOpts module.ConfigTypeOptions, dataType module.DataType) error {
	cDef, ok := h.definitions[cType]
	if !ok {
		return fmt.Errorf("config type '%s' not defined", cType)
	}
	return vltBase(cDef, cTypeOpts, dataType)
}

func (h *ConfigValidationHandler) ValidateOptions(cType string, cTypeOpts module.ConfigTypeOptions) error {
	cDef, ok := h.definitions[cType]
	if !ok {
		return fmt.Errorf("config type '%s' not defined", cType)
	}
	return vltOptions(cDef.Validators, cTypeOpts, h.validators)
}

func (h *ConfigValidationHandler) ValidateValue(cType string, cTypeOpts module.ConfigTypeOptions, value any) error {
	cDef, ok := h.definitions[cType]
	if !ok {
		return fmt.Errorf("config type '%s' not defined", cType)
	}
	return vltValue(cDef.Validators, cTypeOpts, h.validators, value)
}

func LoadDefs(path string) (map[string]model.ConfigDefinition, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	var d map[string]model.ConfigDefinition
	if err = decoder.Decode(&d); err != nil {
		return nil, err
	}
	return d, nil
}

func validateDefs(configDefs map[string]model.ConfigDefinition, validators map[string]itf.Validator) error {
	for ref, cDef := range configDefs {
		if cDef.DataType == nil || len(cDef.DataType) == 0 {
			return fmt.Errorf("config definition '%s' missing data type", ref)
		}
		if cDef.Options != nil {
			for key, cDefOpt := range cDef.Options {
				if !cDefOpt.Inherit && (cDefOpt.DataType == nil || len(cDefOpt.DataType) == 0) {
					return fmt.Errorf("config definition '%s' option '%s' missing data type", ref, key)
				}
			}
		}
		if cDef.Validators != nil && validators != nil {
			for _, validator := range cDef.Validators {
				if _, ok := validators[validator.Name]; !ok {
					return fmt.Errorf("config definition '%s' unknown validator '%s'", ref, validator.Name)
				}
				for key, param := range validator.Parameter {
					if param.Ref == nil && param.Value == nil {
						return fmt.Errorf("config definition '%s' validator '%s' parameter '%s' missing input", ref, validator.Name, key)
					}
					if param.Ref != nil {
						re := regexp.MustCompile(`^options\.[a-z0-9A-Z_]+$|^value$`)
						if !re.MatchString(*param.Ref) {
							return fmt.Errorf("config definition '%s' validator '%s' parameter '%s' invalid refrence '%s'", ref, validator.Name, key, *param.Ref)
						}
					}
				}
			}
		}
	}
	return nil
}

func inStrSlice(c string, sl []string) bool {
	for _, s := range sl {
		if c == s {
			return true
		}
	}
	return false
}

func vltBase(cDef model.ConfigDefinition, cTypeOpts module.ConfigTypeOptions, dataType module.DataType) error {
	if _, ok := cDef.DataType[dataType]; !ok {
		return fmt.Errorf("data type '%s' not supported", dataType)
	}
	if len(cTypeOpts) > 0 && len(cDef.Options) == 0 {
		return fmt.Errorf("options not supported")
	}
	for name := range cTypeOpts {
		if _, ok := cDef.Options[name]; !ok {
			return fmt.Errorf("option '%s' not supported", name)
		}
	}
	for name, cDefO := range cDef.Options {
		if cTypeO, ok := cTypeOpts[name]; ok {
			if cDefO.Inherit {
				if cTypeO.DataType != dataType {
					return fmt.Errorf("data type '%s' not supported by option '%s'", cTypeO.DataType, name)
				}
			} else {
				if _, ok := cDefO.DataType[cTypeO.DataType]; !ok {
					return fmt.Errorf("data type '%s' not supported by option '%s'", cTypeO.DataType, name)
				}
			}
		} else if cDefO.Required {
			return fmt.Errorf("option '%s' required", name)
		}
	}
	return nil
}

func genVltOptParams(cDefVltParams map[string]model.ConfigDefinitionValidatorParam, cTypeOpts module.ConfigTypeOptions) map[string]any {
	vp := make(map[string]any)
	for name, cDefVP := range cDefVltParams {
		if cDefVP.Ref != nil {
			if *cDefVP.Ref == "value" {
				if cDefVP.Value != nil {
					vp[name] = cDefVP.Value
				} else {
					vp = nil
					break
				}
			} else {
				cTypeOName := strings.Split(*cDefVP.Ref, ".")[1]
				if cTypeO, ok := cTypeOpts[cTypeOName]; ok {
					vp[name] = cTypeO.Value
				} else {
					if cDefVP.Value != nil {
						vp[name] = cDefVP.Value
					} else {
						vp = nil
						break
					}
				}
			}
		} else {
			vp[name] = cDefVP.Value
		}
	}
	return vp
}

func vltOptions(cDefVlts []model.ConfigDefinitionValidator, cTypeOpts module.ConfigTypeOptions, validators map[string]itf.Validator) error {
	for _, cDefVlt := range cDefVlts {
		p := genVltOptParams(cDefVlt.Parameter, cTypeOpts)
		if len(p) > 0 {
			vFunc, ok := validators[cDefVlt.Name]
			if !ok {
				return fmt.Errorf("validator '%s' not defined", cDefVlt.Name)
			}
			err := vFunc(p)
			if err != nil {
				return fmt.Errorf("validator '%s' returned with: %s", cDefVlt.Name, err)
			}
		}
	}
	return nil
}

func genVltValParams(cDefVltParams map[string]model.ConfigDefinitionValidatorParam, cTypeOpts module.ConfigTypeOptions, value any) map[string]any {
	vp := make(map[string]any)
	for name, cDefVP := range cDefVltParams {
		if cDefVP.Ref != nil {
			if *cDefVP.Ref == "value" {
				if value != nil {
					vp[name] = value
				} else {
					if cDefVP.Value != nil {
						vp[name] = cDefVP.Value
					} else {
						vp = nil
						break
					}
				}
			} else {
				cTypeOName := strings.Split(*cDefVP.Ref, ".")[1]
				if cTypeO, ok := cTypeOpts[cTypeOName]; ok {
					vp[name] = cTypeO.Value
				} else {
					if cDefVP.Value != nil {
						vp[name] = cDefVP.Value
					} else {
						vp = nil
						break
					}
				}
			}
		} else {
			vp[name] = cDefVP.Value
		}
	}
	return vp
}

func vltValue(cDefVlts []model.ConfigDefinitionValidator, cTypeOpts module.ConfigTypeOptions, validators map[string]itf.Validator, value any) error {
	for _, cDefVlt := range cDefVlts {
		p := genVltValParams(cDefVlt.Parameter, cTypeOpts, value)
		if len(p) > 0 {
			vFunc, ok := validators[cDefVlt.Name]
			if !ok {
				return fmt.Errorf("validator '%s' not defined", cDefVlt.Name)
			}
			err := vFunc(p)
			if err != nil {
				return fmt.Errorf("validator '%s' returned with: %s", cDefVlt.Name, err)
			}
		}
	}
	return nil
}
