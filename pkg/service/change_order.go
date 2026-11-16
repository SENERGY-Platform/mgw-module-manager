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
	deps := make(map[string][]string)
	for id, op := range ops {
		for _, dep := range slices.Sorted(maps.Keys(op.mod.Mod.Dependencies)) {
			if _, ok := ops[dep]; ok && dep != id {
				deps[id] = append(deps[id], dep)
			}
		}
	}
	components := dependencyComponents(slices.Sorted(maps.Keys(ops)), deps)
	componentOf := make(map[string]int)
	for i, members := range components {
		for _, id := range members {
			componentOf[id] = i
		}
	}
	// pending[i] counts the other components component i depends on that are not emitted yet.
	pending := make([]int, len(components))
	dependents := make([][]int, len(components))
	for i, members := range components {
		seen := make(map[int]bool)
		for _, id := range members {
			for _, dep := range deps[id] {
				c := componentOf[dep]
				if c != i && !seen[c] {
					seen[c] = true
					pending[i]++
					dependents[c] = append(dependents[c], i)
				}
			}
		}
	}
	emitted := make([]bool, len(components))
	result := make([]changeOp, 0, len(ops))
	for range components {
		next := -1
		for i, members := range components {
			// Members are sorted, so the first one is the id the component is ordered by.
			if !emitted[i] && pending[i] == 0 && (next < 0 || members[0] < components[next][0]) {
				next = i
			}
		}
		emitted[next] = true
		for _, id := range components[next] {
			result = append(result, ops[id])
		}
		for _, c := range dependents[next] {
			pending[c]--
		}
	}
	return result
}

// dependencyComponents returns the strongly connected components of the dependency graph, each sorted by id. A module
// without a cycle forms a component of its own.
func dependencyComponents(ids []string, deps map[string][]string) [][]string {
	index := make(map[string]int)
	lowLink := make(map[string]int)
	onStack := make(map[string]bool)
	var stack []string
	var components [][]string
	next := 0
	var visit func(id string)
	visit = func(id string) {
		index[id] = next
		lowLink[id] = next
		next++
		stack = append(stack, id)
		onStack[id] = true
		for _, dep := range deps[id] {
			if _, ok := index[dep]; !ok {
				visit(dep)
				lowLink[id] = min(lowLink[id], lowLink[dep])
			} else if onStack[dep] {
				lowLink[id] = min(lowLink[id], index[dep])
			}
		}
		if lowLink[id] != index[id] {
			return
		}
		var members []string
		for {
			top := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			onStack[top] = false
			members = append(members, top)
			if top == id {
				break
			}
		}
		slices.Sort(members)
		components = append(components, members)
	}
	for _, id := range ids {
		if _, ok := index[id]; !ok {
			visit(id)
		}
	}
	return components
}

// failedDependency returns the error for a module that must not be applied because a module of the request it depends
// on failed or was not applied itself. failed maps every such module to the module whose failure caused it.
func failedDependency(mod modWrapper, failed map[string]string) (cause string, msg string, ok bool) {
	for _, dep := range slices.Sorted(maps.Keys(mod.Mod.Dependencies)) {
		cause, ok = failed[dep]
		if !ok {
			continue
		}
		if cause == dep {
			return cause, fmt.Sprintf("not applied, dependency '%s' failed", dep), true
		}
		return cause, fmt.Sprintf("not applied, dependency '%s' not applied because '%s' failed", dep, cause), true
	}
	return "", "", false
}
