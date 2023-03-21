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
	"errors"
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

func NewConfigValidationHandler(definitionsPath string, validators map[string]itf.Validator) (*ConfigValidationHandler, error) {
	definitions, err := loadDefs(definitionsPath)
	if err != nil {
		return nil, err
	}
	if err = validateDefs(definitions, validators); err != nil {
		return nil, err
	}
	return &ConfigValidationHandler{definitions: definitions, validators: validators}, nil
}

func (h *ConfigValidationHandler) ValidateBase(cType string, cTypeOpt module.ConfigTypeOptions, dataType module.DataType) error {
	if h.definitions != nil {
		def, ok := h.definitions[cType]
		if !ok {
			return fmt.Errorf("unknown config type '%s'", cType)
		}
		if !inStrSlice(dataType, def.DataType) {
			return fmt.Errorf("data type '%s' not supported by '%s'", dataType, cType)
		}
		if cTypeOpt != nil && def.Options == nil {
			return fmt.Errorf("options not supported by '%s'", cType)
		}
		if cTypeOpt == nil && def.Options != nil {
			for key, defOpt := range def.Options {
				if defOpt.Required {
					return fmt.Errorf("option '%s' is required by '%s'", key, cType)
				}
			}
		}
		if cTypeOpt != nil && def.Options != nil {
			for key := range cTypeOpt {
				if _, ok := def.Options[key]; !ok {
					return fmt.Errorf("option '%s' not supported by '%s'", key, cType)
				}
			}
			for key, defOpt := range def.Options {
				if tOpt, ok := cTypeOpt[key]; ok {
					if defOpt.Inherit {
						if tOpt.DataType != dataType {
							return fmt.Errorf("data type '%s' not supported by option '%s' of '%s'", tOpt.DataType, key, cType)
						}
					} else {
						if !inStrSlice(tOpt.DataType, defOpt.DataType) {
							return fmt.Errorf("data type '%s' not supported by option '%s' of '%s'", tOpt.DataType, key, cType)
						}
					}
				} else if defOpt.Required {
					return fmt.Errorf("option '%s' is required by '%s'", key, cType)
				}
			}
		}
	} else {
		return errors.New("no config definitions")
	}
	return nil
}

func (h *ConfigValidationHandler) ValidateOptions(cType string, cTypeOpt module.ConfigTypeOptions) error {
	if h.definitions != nil {
		def, ok := h.definitions[cType]
		if !ok {
			return fmt.Errorf("unknown config type '%s'", cType)
		}
		if def.Validators != nil && cTypeOpt != nil {
			for _, validator := range def.Validators {
				params := make(map[string]any)
				for key, val := range validator.Parameter {
					if val.Ref == nil {
						params[key] = val.Value
					} else {
						if *val.Ref == "value" && val.Value == nil {
							params = nil
							break
						}
						oKey := strings.Split(*val.Ref, ".")
						if v, ok := cTypeOpt[oKey[1]]; ok {
							params[key] = v.Value
						} else {
							if val.Value == nil {
								params = nil
								break
							}
							params[key] = val.Value
						}
					}
				}
				if params != nil {
					validatorFunc := h.validators[validator.Name]
					err := validatorFunc(params)
					if err != nil {
						return fmt.Errorf("options '%s' validation failed: %s", validator.Name, err)
					}
				}
			}
		}
	} else {
		return errors.New("no config definitions")
	}
	return nil
}

func (h *ConfigValidationHandler) ValidateValue(cType string, cTypeOpt module.ConfigTypeOptions, value any) error {
	if h.definitions != nil {
		def, ok := h.definitions[cType]
		if !ok {
			return fmt.Errorf("unknown config type '%s'", cType)
		}
		if def.Validators != nil {
			for _, validator := range def.Validators {
				params := make(map[string]any)
				for key, val := range validator.Parameter {
					if val.Ref == nil {
						params[key] = val.Value
					} else {
						if *val.Ref == "value" {
							params[key] = value
						} else {
							if cTypeOpt == nil && val.Value == nil {
								params = nil
								break
							}
							oKey := strings.Split(*val.Ref, ".")
							if v, ok := cTypeOpt[oKey[1]]; ok {
								params[key] = v.Value
							} else {
								if val.Value == nil {
									params = nil
									break
								}
								params[key] = val.Value
							}
						}
					}
				}
				if params != nil {
					validatorFunc := h.validators[validator.Name]
					err := validatorFunc(params)
					if err != nil {
						return fmt.Errorf("options '%s' validation failed: %s", validator.Name, err)
					}
				}
			}
		}
	} else {
		return errors.New("no config definitions")
	}
	return nil
}

func loadDefs(path string) (map[string]model.ConfigDefinition, error) {
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
