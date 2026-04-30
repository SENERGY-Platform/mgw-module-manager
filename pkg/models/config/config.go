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

package models_config

const (
	StringType = iota
	Int64Type
	Float64Type
	BoolType
)

type Value struct {
	DataType     int
	IsSlice      bool
	String       string
	StringSlice  []string
	Int64        int64
	Int64Slice   []int64
	Float64      float64
	Float64Slice []float64
	Bool         bool
	BoolSlice    []bool
}

type InterfaceValue struct {
	DataType int         `json:"data_type"`
	IsSlice  bool        `json:"is_slice"`
	Value    interface{} `json:"value"`
}
