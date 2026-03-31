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

package helper_slices

import (
	"iter"
)

func AllFunc[S ~[]E, E any, K comparable](sl S, keyF func(item E) K) iter.Seq2[K, E] {
	return func(yield func(K, E) bool) {
		for i, v := range sl {
			if !yield(keyF(sl[i]), v) {
				return
			}
		}
	}
}

func CollectFunc[E any, K any](seq iter.Seq[E], valF func(item E) K) []K {
	s := []K(nil)
	for v := range seq {
		s = append(s, valF(v))
	}
	return s
}

func ToAny[S ~[]E, E any](sl S) []any {
	anySl := make([]any, len(sl))
	for i, v := range sl {
		anySl[i] = v
	}
	return anySl
}

func RemoveDuplicates[S ~[]E, E comparable](sl S) []E {
	if len(sl) < 2 {
		return sl
	}
	set := make(map[E]struct{})
	var sl2 []E
	for _, s := range sl {
		if _, ok := set[s]; !ok {
			sl2 = append(sl2, s)
		}
		set[s] = struct{}{}
	}
	return sl2
}

func RemoveDuplicatesFunc[S ~[]E, E any, K comparable](sl S, f func(E) K) []E {
	if len(sl) < 2 {
		return sl
	}
	set := make(map[K]struct{})
	var sl2 []E
	for _, s := range sl {
		key := f(s)
		if _, ok := set[key]; !ok {
			sl2 = append(sl2, s)
		}
		set[key] = struct{}{}
	}
	return sl2
}
