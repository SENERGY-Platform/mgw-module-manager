/*
 * Copyright 2025 InfAI (CC SES)
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

package util

import "fmt"

func SliceToMap[S ~[]E, E any](sl S, keyF func(item E) string) map[string]E {
	m := make(map[string]E)
	for i := 0; i < len(sl); i++ {
		m[keyF(sl[i])] = sl[i]
	}
	return m
}

func SliceToMapSafe[S ~[]E, E any](sl S, keyF func(item E) string) (map[string]E, error) {
	m := make(map[string]E)
	for i := 0; i < len(sl); i++ {
		k := keyF(sl[i])
		if _, ok := m[k]; ok {
			return nil, fmt.Errorf("duplicate key: %s", k)
		}
		m[k] = sl[i]
	}
	return m, nil
}

func SelectByPriority[S ~[]E, E any](sl S, comp func(item E, lastPrio int) (int, bool)) E {
	var lastPrio int
	var candidate E
	for i := 0; i < len(sl); i++ {
		prio, ok := comp(sl[i], lastPrio)
		if i == 0 || ok {
			lastPrio = prio
			candidate = sl[i]
		}
	}
	return candidate
}
