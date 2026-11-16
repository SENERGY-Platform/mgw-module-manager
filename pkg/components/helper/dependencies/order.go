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

package dependencies

import (
	"maps"
	"slices"
)

// Order returns ids so that every id follows the ids it depends on, ties broken by id. deps maps an id to the ids it
// depends on; ids not in ids, duplicates and an id depending on itself are ignored. Ids in a dependency cycle cannot
// all follow each other and are returned in id order.
func Order(ids []string, deps map[string][]string) []string {
	set := make(map[string]bool)
	for _, id := range ids {
		set[id] = true
	}
	sortedIds := slices.Sorted(maps.Keys(set))
	edges := make(map[string][]string)
	for _, id := range sortedIds {
		depIds := slices.Clone(deps[id])
		slices.Sort(depIds)
		for _, dep := range slices.Compact(depIds) {
			if set[dep] && dep != id {
				edges[id] = append(edges[id], dep)
			}
		}
	}
	components := stronglyConnected(sortedIds, edges)
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
			for _, dep := range edges[id] {
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
	result := make([]string, 0, len(sortedIds))
	for range components {
		next := -1
		for i, members := range components {
			// Members are sorted, so the first one is the id the component is ordered by.
			if !emitted[i] && pending[i] == 0 && (next < 0 || members[0] < components[next][0]) {
				next = i
			}
		}
		emitted[next] = true
		result = append(result, components[next]...)
		for _, c := range dependents[next] {
			pending[c]--
		}
	}
	return result
}

// stronglyConnected returns the strongly connected components of the dependency graph, each sorted by id. An id
// without a cycle forms a component of its own.
func stronglyConnected(ids []string, edges map[string][]string) [][]string {
	index := make(map[string]int)
	lowLink := make(map[string]int)
	onStack := make(map[string]bool)
	var stack []string
	var result [][]string
	next := 0
	var visit func(id string)
	visit = func(id string) {
		index[id] = next
		lowLink[id] = next
		next++
		stack = append(stack, id)
		onStack[id] = true
		for _, dep := range edges[id] {
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
		result = append(result, members)
	}
	for _, id := range ids {
		if _, ok := index[id]; !ok {
			visit(id)
		}
	}
	return result
}

// Nearest returns the id matching match that is nearest to start along deps, start included, ties broken by the order
// of deps. Dependency cycles are followed, so a match is found from every id of the cycle it belongs to.
func Nearest(start string, deps map[string][]string, match func(id string) bool) (string, bool) {
	seen := map[string]bool{start: true}
	queue := []string{start}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if match(id) {
			return id, true
		}
		for _, dep := range deps[id] {
			if !seen[dep] {
				seen[dep] = true
				queue = append(queue, dep)
			}
		}
	}
	return "", false
}
