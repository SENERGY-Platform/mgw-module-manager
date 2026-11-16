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
	"fmt"
	"maps"
	"slices"

	lib_constants "github.com/SENERGY-Platform/mgw-module-manager/lib/constants"
	helper_dependencies "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/dependencies"
)

// changeOp is an install or change item of a modules change request.
type changeOp struct {
	action string
	mod    modWrapper
}

// orderChangeOps returns the install and change items of changeRequest so that every module follows the modules of the
// request it depends on, ties broken by id. Modules in a dependency cycle cannot all follow each other and are applied
// in id order.
func orderChangeOps(changeRequest *modulesChangeRequest) []changeOp {
	ops := make(map[string]changeOp)
	for _, mod := range changeRequest.Install {
		ops[mod.Mod.ID] = changeOp{action: lib_constants.ActionInstall, mod: mod}
	}
	for _, item := range changeRequest.Change {
		ops[item.Next.Mod.ID] = changeOp{action: lib_constants.ActionChange, mod: item.Next}
	}
	deps := changeOpDependencies(slices.Collect(maps.Values(ops)))
	var result []changeOp
	for _, id := range helper_dependencies.Order(slices.Collect(maps.Keys(ops)), deps) {
		result = append(result, ops[id])
	}
	return result
}

// changeOpDependencies maps the id of every module of ops to the sorted ids of the other modules of ops it depends on.
func changeOpDependencies(ops []changeOp) map[string][]string {
	mods := make(map[string]modWrapper)
	for _, op := range ops {
		mods[op.mod.Mod.ID] = op.mod
	}
	deps := make(map[string][]string)
	for id, mod := range mods {
		for _, dep := range slices.Sorted(maps.Keys(mod.Mod.Dependencies)) {
			if _, ok := mods[dep]; ok && dep != id {
				deps[id] = append(deps[id], dep)
			}
		}
	}
	return deps
}

// failedDependency returns the error for a module that must not be applied because a module of the request it depends
// on, directly or transitively, failed. Within a dependency cycle a module can be applied before a module it depends on
// fails, so the failure is searched for along all dependencies. notApplied holds the modules kept back by a failure.
func failedDependency(id string, deps map[string][]string, failed, notApplied map[string]bool) (string, bool) {
	for _, dep := range deps[id] {
		cause, ok := helper_dependencies.Nearest(dep, deps, func(id string) bool { return failed[id] })
		switch {
		case !ok:
			continue
		case cause == dep:
			return fmt.Sprintf("not applied, dependency '%s' failed", dep), true
		case notApplied[dep]:
			return fmt.Sprintf("not applied, dependency '%s' not applied because '%s' failed", dep, cause), true
		default:
			return fmt.Sprintf("not applied, dependency '%s' depends on failed '%s'", dep, cause), true
		}
	}
	return "", false
}
