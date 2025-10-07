/*
 * Copyright 2023 InfAI (CC SES)
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

package set

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestSet_UnmarshalJSON(t *testing.T) {
	var b Set[string]
	if err := json.Unmarshal([]byte("[\"test\"]"), &b); err != nil {
		t.Error("err != nil")
	}
	a := Set[string]{"test": {}}
	if reflect.DeepEqual(a, b) == false {
		t.Errorf("%v != %v", a, b)
	}
	if err := json.Unmarshal([]byte("[1]"), &b); err == nil {
		t.Error("err == nil")
	}
}

func TestSet_MarshalJSON(t *testing.T) {
	s := Set[string]{"test": {}}
	a := "[\"test\"]"
	if b, err := json.Marshal(s); err != nil {
		t.Error("err != nil")
	} else if a != string(b) {
		t.Errorf("%s != %s", a, string(b))
	}

}
