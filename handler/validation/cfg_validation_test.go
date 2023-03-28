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
	"errors"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-lib/util"
	"module-manager/itf"
	"module-manager/model"
	"reflect"
	"testing"
)

func TestGenVltOptParams(t *testing.T) {
	cDefVP := make(map[string]model.ConfigDefinitionValidatorParam)
	var cTypeO module.ConfigTypeOptions
	if b := genVltOptParams(cDefVP, cTypeO); len(b) != 0 {
		t.Errorf("len(%v) != 0", b)
	}
	// ------------------------------
	str := "test"
	vRef := "value"
	oRef := ".opt"
	cDefVP[""] = model.ConfigDefinitionValidatorParam{
		Value: str,
		Ref:   nil,
	}
	a := map[string]any{
		"": str,
	}
	b := genVltOptParams(cDefVP, cTypeO)
	if reflect.DeepEqual(a, b) == false {
		t.Errorf("%v != %v", a, b)
	}
	// ------------------------------
	cDefVP[""] = model.ConfigDefinitionValidatorParam{
		Value: str,
		Ref:   &vRef,
	}
	b = genVltOptParams(cDefVP, cTypeO)
	if reflect.DeepEqual(a, b) == false {
		t.Errorf("%v != %v", a, b)
	}
	// ------------------------------
	cTypeO = make(module.ConfigTypeOptions)
	cDefVP[""] = model.ConfigDefinitionValidatorParam{
		Value: str,
		Ref:   &oRef,
	}
	b = genVltOptParams(cDefVP, cTypeO)
	if reflect.DeepEqual(a, b) == false {
		t.Errorf("%v != %v", a, b)
	}
	// ------------------------------
	cTypeO.SetString("opt", str)
	cDefVP[""] = model.ConfigDefinitionValidatorParam{
		Value: nil,
		Ref:   &oRef,
	}
	b = genVltOptParams(cDefVP, cTypeO)
	if reflect.DeepEqual(a, b) == false {
		t.Errorf("%v != %v", a, b)
	}
	// ------------------------------
	cDefVP[""] = model.ConfigDefinitionValidatorParam{
		Value: "val",
		Ref:   &oRef,
	}
	b = genVltOptParams(cDefVP, cTypeO)
	if reflect.DeepEqual(a, b) == false {
		t.Errorf("%v != %v", a, b)
	}
	// ------------------------------
	cDefVP[""] = model.ConfigDefinitionValidatorParam{
		Value: nil,
		Ref:   &vRef,
	}
	if b = genVltOptParams(cDefVP, cTypeO); len(b) != 0 {
		t.Errorf("len(%v) != 0", b)
	}
	// ------------------------------
	cDefVP[""] = model.ConfigDefinitionValidatorParam{
		Value: nil,
		Ref:   &vRef,
	}
	if b = genVltOptParams(cDefVP, cTypeO); len(b) != 0 {
		t.Errorf("len(%v) != 0", b)
	}
	// ------------------------------
	oRef2 := ".opt2"
	cDefVP[""] = model.ConfigDefinitionValidatorParam{
		Value: nil,
		Ref:   &oRef2,
	}
	if b = genVltOptParams(cDefVP, cTypeO); len(b) != 0 {
		t.Errorf("len(%v) != 0", b)
	}
}

func TestGenVltValParams(t *testing.T) {
	cDefVP := make(map[string]model.ConfigDefinitionValidatorParam)
	var cTypeO module.ConfigTypeOptions
	if b := genVltValParams(cDefVP, cTypeO, nil); len(b) != 0 {
		t.Errorf("len(%v) != 0", b)
	}
	// ------------------------------
	str := "test"
	vRef := "value"
	oRef := ".opt"
	cDefVP[""] = model.ConfigDefinitionValidatorParam{
		Value: str,
		Ref:   nil,
	}
	a := map[string]any{
		"": str,
	}
	b := genVltValParams(cDefVP, cTypeO, nil)
	if reflect.DeepEqual(a, b) == false {
		t.Errorf("%v != %v", a, b)
	}
	// ------------------------------
	cDefVP[""] = model.ConfigDefinitionValidatorParam{
		Value: nil,
		Ref:   &vRef,
	}
	b = genVltValParams(cDefVP, cTypeO, str)
	if reflect.DeepEqual(a, b) == false {
		t.Errorf("%v != %v", a, b)
	}
	// ------------------------------
	cDefVP[""] = model.ConfigDefinitionValidatorParam{
		Value: "test2",
		Ref:   &vRef,
	}
	b = genVltValParams(cDefVP, cTypeO, str)
	if reflect.DeepEqual(a, b) == false {
		t.Errorf("%v != %v", a, b)
	}
	// ------------------------------
	cTypeO = make(module.ConfigTypeOptions)
	cDefVP[""] = model.ConfigDefinitionValidatorParam{
		Value: str,
		Ref:   &oRef,
	}
	b = genVltValParams(cDefVP, cTypeO, nil)
	if reflect.DeepEqual(a, b) == false {
		t.Errorf("%v != %v", a, b)
	}
	// ------------------------------
	cDefVP[""] = model.ConfigDefinitionValidatorParam{
		Value: nil,
		Ref:   &oRef,
	}
	if b = genVltValParams(cDefVP, cTypeO, nil); len(b) != 0 {
		t.Errorf("len(%v) != 0", b)
	}
	// ------------------------------
	cTypeO.SetString("opt", str)
	cDefVP[""] = model.ConfigDefinitionValidatorParam{
		Value: nil,
		Ref:   &oRef,
	}
	b = genVltValParams(cDefVP, cTypeO, nil)
	if reflect.DeepEqual(a, b) == false {
		t.Errorf("%v != %v", a, b)
	}
	// ------------------------------
	cDefVP[""] = model.ConfigDefinitionValidatorParam{
		Value: "test2",
		Ref:   &oRef,
	}
	b = genVltValParams(cDefVP, cTypeO, nil)
	if reflect.DeepEqual(a, b) == false {
		t.Errorf("%v != %v", a, b)
	}
	// ------------------------------
	cDefVP[""] = model.ConfigDefinitionValidatorParam{
		Value: nil,
		Ref:   &vRef,
	}
	if b = genVltValParams(cDefVP, cTypeO, nil); len(b) != 0 {
		t.Errorf("len(%v) != 0", b)
	}
}

func TestVltOptions(t *testing.T) {
	var cDefVlts []model.ConfigDefinitionValidator
	vlts := make(map[string]itf.Validator)
	if err := vltTypeOpts(cDefVlts, nil, vlts); err != nil {
		t.Error("err != nil")
	}
	// ------------------------------
	cDefVlts = []model.ConfigDefinitionValidator{
		{
			Name: "vlt",
			Parameter: map[string]model.ConfigDefinitionValidatorParam{
				"": {
					Value: "val",
					Ref:   nil,
				},
			},
		},
	}
	if err := vltTypeOpts(cDefVlts, nil, vlts); err == nil {
		t.Error("err == nil")
	}
	// ------------------------------
	vlts["vlt"] = func(params map[string]any) error {
		return nil
	}
	if err := vltTypeOpts(cDefVlts, nil, vlts); err != nil {
		t.Error("err != nil")
	}
	// ------------------------------
	vlts["vlt"] = func(params map[string]any) error {
		return errors.New("test")
	}
	if err := vltTypeOpts(cDefVlts, nil, vlts); err == nil {
		t.Error("err == nil")
	}
	// ------------------------------
	vRef := "value"
	cDefVlts = []model.ConfigDefinitionValidator{
		{
			Name: "vlt",
			Parameter: map[string]model.ConfigDefinitionValidatorParam{
				"a": {
					Value: "val",
					Ref:   nil,
				},
				"b": {
					Value: nil,
					Ref:   &vRef,
				},
			},
		},
	}
	if err := vltTypeOpts(cDefVlts, nil, vlts); err != nil {
		t.Error("err != nil")
	}
}

func TestVltValue(t *testing.T) {
	var cDefVlts []model.ConfigDefinitionValidator
	vlts := make(map[string]itf.Validator)
	if err := vltValue(cDefVlts, nil, vlts, nil); err != nil {
		t.Error("err != nil")
	}
	// ------------------------------
	cDefVlts = []model.ConfigDefinitionValidator{
		{
			Name: "vlt",
			Parameter: map[string]model.ConfigDefinitionValidatorParam{
				"": {
					Value: "val",
					Ref:   nil,
				},
			},
		},
	}
	if err := vltValue(cDefVlts, nil, vlts, nil); err == nil {
		t.Error("err == nil")
	}
	// ------------------------------
	vlts["vlt"] = func(params map[string]any) error {
		return nil
	}
	if err := vltValue(cDefVlts, nil, vlts, nil); err != nil {
		t.Error("err != nil")
	}
	// ------------------------------
	vlts["vlt"] = func(params map[string]any) error {
		return errors.New("test")
	}
	if err := vltValue(cDefVlts, nil, vlts, nil); err == nil {
		t.Error("err == nil")
	}
	// ------------------------------
	vRef := "value"
	cDefVlts = []model.ConfigDefinitionValidator{
		{
			Name: "vlt",
			Parameter: map[string]model.ConfigDefinitionValidatorParam{
				"a": {
					Value: "val",
					Ref:   nil,
				},
				"b": {
					Value: nil,
					Ref:   &vRef,
				},
			},
		},
	}
	if err := vltValue(cDefVlts, nil, vlts, nil); err != nil {
		t.Error("err != nil")
	}
}

func TestVltBase(t *testing.T) {
	var cDef model.ConfigDefinition
	var cTypeOpts module.ConfigTypeOptions
	if err := vltBase(cDef, cTypeOpts, ""); err == nil {
		t.Error("err == nil")
	}
	// ------------------------------
	dt := "dt"
	opt := "opt"
	cDef = model.ConfigDefinition{
		DataType:   util.Set[string]{dt: {}},
		Options:    nil,
		Validators: nil,
	}
	if err := vltBase(cDef, cTypeOpts, dt); err != nil {
		t.Error("err != nil")
	}
	// ------------------------------
	cDef = model.ConfigDefinition{
		DataType: util.Set[string]{dt: {}},
		Options: map[string]model.ConfigDefinitionOption{
			opt: {},
		},
		Validators: nil,
	}
	if err := vltBase(cDef, cTypeOpts, dt); err != nil {
		t.Error("err != nil")
	}
	// ------------------------------
	cDef = model.ConfigDefinition{
		DataType: util.Set[string]{dt: {}},
		Options: map[string]model.ConfigDefinitionOption{
			opt: {
				Required: true,
			},
		},
		Validators: nil,
	}
	if err := vltBase(cDef, cTypeOpts, dt); err == nil {
		t.Error("err == nil")
	}
	// ------------------------------
	cTypeOpts = make(module.ConfigTypeOptions)
	cTypeOpts.SetString(opt, dt)
	cDef = model.ConfigDefinition{
		DataType:   util.Set[string]{dt: {}},
		Options:    nil,
		Validators: nil,
	}
	if err := vltBase(cDef, cTypeOpts, dt); err == nil {
		t.Error("err == nil")
	}
	// ------------------------------
	cDef = model.ConfigDefinition{
		DataType: util.Set[string]{dt: {}},
		Options: map[string]model.ConfigDefinitionOption{
			opt: {},
		},
		Validators: nil,
	}
	if err := vltBase(cDef, cTypeOpts, dt); err == nil {
		t.Error("err == nil")
	}
	// ------------------------------
	cDef = model.ConfigDefinition{
		DataType: util.Set[string]{dt: {}},
		Options: map[string]model.ConfigDefinitionOption{
			opt: {
				DataType: util.Set[string]{dt: {}},
			},
		},
		Validators: nil,
	}
	if err := vltBase(cDef, cTypeOpts, dt); err == nil {
		t.Error("err == nil")
	}
	// ------------------------------
	cDef = model.ConfigDefinition{
		DataType: util.Set[string]{dt: {}},
		Options: map[string]model.ConfigDefinitionOption{
			opt: {
				Inherit: true,
			},
		},
		Validators: nil,
	}
	if err := vltBase(cDef, cTypeOpts, dt); err == nil {
		t.Error("err == nil")
	}
	// ------------------------------
	cDef = model.ConfigDefinition{
		DataType: util.Set[string]{dt: {}},
		Options: map[string]model.ConfigDefinitionOption{
			opt: {
				DataType: util.Set[string]{module.StringType: {}},
			},
		},
		Validators: nil,
	}
	if err := vltBase(cDef, cTypeOpts, dt); err != nil {
		t.Error("err != nil")
	}
	// ------------------------------
	cDef = model.ConfigDefinition{
		DataType: util.Set[string]{module.StringType: {}},
		Options: map[string]model.ConfigDefinitionOption{
			opt: {
				Inherit: true,
			},
		},
		Validators: nil,
	}
	if err := vltBase(cDef, cTypeOpts, module.StringType); err != nil {
		t.Error("err != nil")
	}
	// ------------------------------
	cTypeOpts.SetString("test", dt)
	if err := vltBase(cDef, cTypeOpts, module.StringType); err == nil {
		t.Error("err == nil")
	}
}
