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
		for _, v := range sl {
			if !yield(keyF(v), v) {
				return
			}
		}
	}
}

func Extract[S ~[]E, E, V any, K comparable](sl S, pairF func(item E) (K, V)) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for _, v := range sl {
			if !yield(pairF(v)) {
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
