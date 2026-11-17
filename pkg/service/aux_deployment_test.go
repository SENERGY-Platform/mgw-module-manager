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
	"testing"

	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

type adModules struct {
	modulesHandler
	module pkg_models.Module
	err    error
}

func (m *adModules) GetModule(_ context.Context, _ string) (pkg_models.Module, error) {
	return m.module, m.err
}

func TestService_getReadableModule(t *testing.T) {
	readErr := errors.New("read modfile: broken")
	callErr := errors.New("database down")
	readable := pkg_models.Module{ModuleLibModule: external_models.ModuleLibModule{ID: "a"}}
	tests := []struct {
		name    string
		mods    *adModules
		wantErr error
	}{
		{name: "readable", mods: &adModules{module: readable}},
		{name: "unreadable modfile", mods: &adModules{module: pkg_models.Module{Err: readErr}}, wantErr: readErr},
		{name: "call fails", mods: &adModules{err: callErr}, wantErr: callErr},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := &Service{modulesHandler: tc.mods}
			module, err := srv.getReadableModule(context.Background(), "a")
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Errorf("error %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil || module.ID != "a" {
				t.Errorf("got %+v, %v", module, err)
			}
		})
	}
}

func (m *adModules) GetModules(_ context.Context, _ pkg_models.ModulesFilterWithName, _ bool) (map[string]pkg_models.Module, error) {
	if m.err != nil {
		return nil, m.err
	}
	return map[string]pkg_models.Module{"a": m.module}, nil
}

func TestService_getReadableModules(t *testing.T) {
	readErr := errors.New("read modfile: broken")
	callErr := errors.New("database down")
	readable := pkg_models.Module{ModuleLibModule: external_models.ModuleLibModule{ID: "a"}}
	tests := []struct {
		name    string
		mods    *adModules
		wantErr error
	}{
		{name: "readable", mods: &adModules{module: readable}},
		{name: "unreadable modfile", mods: &adModules{module: pkg_models.Module{Err: readErr}}, wantErr: readErr},
		{name: "call fails", mods: &adModules{err: callErr}, wantErr: callErr},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := &Service{modulesHandler: tc.mods}
			modules, err := srv.getReadableModules(context.Background(), pkg_models.ModulesFilterWithName{}, true)
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) || modules != nil {
					t.Errorf("got %v, %v, want error %v", modules, err, tc.wantErr)
				}
				return
			}
			if err != nil || modules["a"].ID != "a" {
				t.Errorf("got %+v, %v", modules, err)
			}
		})
	}
}
