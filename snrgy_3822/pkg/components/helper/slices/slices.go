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

package slices

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
