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

package handlers

import (
	"reflect"
)

// visitedRef identifies a map, slice or pointer that has already been guarded to prevent endless recursion on cyclic values.
type visitedRef struct {
	typ reflect.Type
	ptr uintptr
	len int
}

// guardNil initialises every nil map and slice reachable from v, so that they are serialized as '{}' or '[]' instead of 'null'.
// The guarded value is returned to allow inline use, e.g. gc.JSON(http.StatusOK, guardNil(res)).
// Values reachable only through unexported struct fields are left untouched.
func guardNil[T any](v T) T {
	guardNilValue(reflect.ValueOf(&v).Elem(), make(map[visitedRef]struct{}))
	return v
}

func guardNilValue(val reflect.Value, visited map[visitedRef]struct{}) {
	switch val.Kind() {
	case reflect.Pointer:
		if val.IsNil() || isVisited(val, 0, visited) {
			return
		}
		guardNilValue(val.Elem(), visited)
	case reflect.Interface:
		if val.IsNil() {
			return
		}
		// the value behind an interface is not addressable, therefore a copy is guarded and written back
		cpy := copyValue(val.Elem())
		guardNilValue(cpy, visited)
		if val.CanSet() {
			val.Set(cpy)
		}
	case reflect.Slice:
		if val.IsNil() {
			if val.CanSet() {
				val.Set(reflect.MakeSlice(val.Type(), 0, 0))
			}
			return
		}
		if isVisited(val, val.Len(), visited) {
			return
		}
		for i := 0; i < val.Len(); i++ {
			guardNilValue(val.Index(i), visited)
		}
	case reflect.Array:
		for i := 0; i < val.Len(); i++ {
			guardNilValue(val.Index(i), visited)
		}
	case reflect.Map:
		if val.IsNil() {
			if val.CanSet() {
				val.Set(reflect.MakeMap(val.Type()))
			}
			return
		}
		if isVisited(val, 0, visited) {
			return
		}
		for _, key := range val.MapKeys() {
			// map values are not addressable, therefore a copy is guarded and written back
			cpy := copyValue(val.MapIndex(key))
			guardNilValue(cpy, visited)
			val.SetMapIndex(key, cpy)
		}
	case reflect.Struct:
		typ := val.Type()
		for i := 0; i < val.NumField(); i++ {
			if !typ.Field(i).IsExported() {
				continue
			}
			guardNilValue(val.Field(i), visited)
		}
	}
}

func copyValue(val reflect.Value) reflect.Value {
	cpy := reflect.New(val.Type()).Elem()
	cpy.Set(val)
	return cpy
}

func isVisited(val reflect.Value, length int, visited map[visitedRef]struct{}) bool {
	ref := visitedRef{
		typ: val.Type(),
		ptr: val.Pointer(),
		len: length,
	}
	if _, ok := visited[ref]; ok {
		return true
	}
	visited[ref] = struct{}{}
	return false
}
