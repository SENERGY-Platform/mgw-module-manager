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

const (
	Greater      = ">"
	Less         = "<"
	Equal        = "="
	GreaterEqual = ">="
	LessEqual    = "<="
)

var OperatorMap = map[string]struct{}{
	Greater:      {},
	Less:         {},
	Equal:        {},
	GreaterEqual: {},
	LessEqual:    {},
}

const (
	Bool DataType = iota
	Int64
	Float64
	String
)

var DataTypeRef = []string{
	Bool:    "bool",
	Int64:   "int",
	Float64: "float",
	String:  "string",
}

var DataTypeRefMap = func() map[string]DataType {
	m := make(map[string]DataType)
	for i := 0; i < len(DataTypeRef); i++ {
		m[DataTypeRef[i]] = DataType(i)
	}
	return m
}()
