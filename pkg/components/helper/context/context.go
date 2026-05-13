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

package context

import (
	"context"
	"iter"
	"maps"
)

const ValuesKey = "__context_helper_values__"

func WithValues(parent context.Context, kvArgs ...any) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}
	lenKv := len(kvArgs)
	switch {
	case lenKv == 0:
		return parent
	case lenKv%2 != 0:
		panic("not enough arguments")
	}
	values, ok := parent.Value(ValuesKey).(Values)
	if !ok {
		values.m = make(map[string]interface{})
	}
	c := 2
	for c <= lenKv {
		kv := kvArgs[c-2 : c]
		k, ok := kv[0].(string)
		if !ok {
			panic("key is not a string")
		}
		values.m[k] = kv[1]
		c += 2
	}
	if ok {
		return parent
	}
	return context.WithValue(parent, ValuesKey, values)
}

func GetValues(ctx context.Context) (Values, bool) {
	values, ok := ctx.Value(ValuesKey).(Values)
	return values, ok
}

type Values struct {
	m map[string]interface{}
}

func (v Values) Get(key string) (interface{}, bool) {
	val, ok := v.m[key]
	return val, ok
}

func (v Values) All() iter.Seq2[string, interface{}] {
	return maps.All(v.m)
}
