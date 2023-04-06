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

package api

import (
	"context"
	"github.com/SENERGY-Platform/mgw-module-lib/module"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
)

func (a *Api) AddModule(ctx context.Context, id string) error {
	panic("not implemented")
}

func (a *Api) GetModules(ctx context.Context) ([]*module.Module, error) {
	return a.moduleHandler.List(ctx)
}

func (a *Api) GetModule(ctx context.Context, id string) (*module.Module, error) {
	return a.moduleHandler.Get(ctx, id)
}

func (a *Api) DeleteModule(ctx context.Context, id string) error {
	panic("not implemented")
}

func (a *Api) GetInputTemplate(ctx context.Context, id string) (model.InputTemplate, error) {
	return a.moduleHandler.InputTemplate(ctx, id)
}
