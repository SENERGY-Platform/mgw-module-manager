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
	"reflect"
	"slices"
	"testing"
)

func TestOrder(t *testing.T) {
	tests := []struct {
		name string
		ids  []string
		deps map[string][]string
		want []string
	}{
		{
			name: "independent ids in id order",
			ids:  []string{"c", "a", "b"},
			want: []string{"a", "b", "c"},
		},
		{
			name: "chain",
			ids:  []string{"a", "b", "c"},
			deps: map[string][]string{"a": {"b"}, "b": {"c"}},
			want: []string{"c", "b", "a"},
		},
		{
			name: "diamond",
			ids:  []string{"a", "c", "b", "z"},
			deps: map[string][]string{"a": {"c", "b"}, "c": {"z"}, "b": {"z"}},
			want: []string{"z", "b", "c", "a"},
		},
		{
			name: "cycle between its dependency and its dependent",
			ids:  []string{"a", "m", "n", "z"},
			deps: map[string][]string{"a": {"m"}, "m": {"n"}, "n": {"m", "z"}},
			want: []string{"z", "m", "n", "a"},
		},
		{
			name: "cycle of three",
			ids:  []string{"c", "b", "a"},
			deps: map[string][]string{"c": {"a"}, "b": {"c"}, "a": {"b"}},
			want: []string{"a", "b", "c"},
		},
		{
			name: "dependencies outside ids, on itself and duplicates are ignored",
			ids:  []string{"b", "a", "a"},
			deps: map[string][]string{"a": {"a", "outside", "b", "b"}, "b": {"outside"}},
			want: []string{"b", "a"},
		},
		{
			name: "empty",
			want: []string{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Order(tc.ids, tc.deps); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNearest(t *testing.T) {
	// a depends on b and c, b on d, c on e, d and e on each other
	deps := map[string][]string{"a": {"b", "c"}, "b": {"d"}, "c": {"e"}, "d": {"e"}, "e": {"d"}}
	tests := []struct {
		name    string
		start   string
		matches []string
		want    string
		wantOk  bool
	}{
		{name: "start itself", start: "a", matches: []string{"a", "b"}, want: "a", wantOk: true},
		{name: "fewest steps wins", start: "a", matches: []string{"d", "c"}, want: "c", wantOk: true},
		{name: "same distance in order of deps", start: "a", matches: []string{"e", "d"}, want: "d", wantOk: true},
		{name: "through a cycle", start: "e", matches: []string{"d"}, want: "d", wantOk: true},
		{name: "dependents are not followed", start: "c", matches: []string{"a", "b"}},
		{name: "none", start: "a"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := Nearest(tc.start, deps, func(id string) bool { return slices.Contains(tc.matches, id) })
			if got != tc.want || ok != tc.wantOk {
				t.Errorf("got %q %v, want %q %v", got, ok, tc.want, tc.wantOk)
			}
		})
	}
}
