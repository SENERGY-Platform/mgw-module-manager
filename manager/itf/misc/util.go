/*
 * Copyright 2022 InfAI (CC SES)
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

package misc

import (
	"encoding/json"
	"fmt"
)

func (s *Set[T]) UnmarshalJSON(b []byte) error {
	var sl []T
	if err := json.Unmarshal(b, &sl); err != nil {
		return err
	}
	set := make(Set[T])
	for _, item := range sl {
		set[item] = struct{}{}
	}
	*s = set
	return nil
}

func (s Set[T]) MarshalJSON() ([]byte, error) {
	var sl []T
	for item := range s {
		sl = append(sl, item)
	}
	return json.Marshal(sl)
}

func (s Set[T]) Slice() []T {
	var sl []T
	for item := range s {
		sl = append(sl, item)
	}
	return sl
}

func (d DataType) MarshalJSON() ([]byte, error) {
	return json.Marshal(DataTypeRef[d])
}

func (d *DataType) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	t, ok := DataTypeRefMap[s]
	if !ok {
		return fmt.Errorf("invalid data type '%s'", s)
	}
	*d = t
	return nil
}

func (d DataType) String() string {
	return DataTypeRef[d]
}
