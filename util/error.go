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

package util

import (
	"errors"
	lib_model "github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"net/http"
)

func GetErrCode(err error) *int {
	c := GetStatusCode(err)
	if c > 0 {
		return &c
	}
	return nil
}

func GetStatusCode(err error) int {
	var nfe *lib_model.NotFoundError
	if errors.As(err, &nfe) {
		return http.StatusNotFound
	}
	var iie *lib_model.InvalidInputError
	if errors.As(err, &iie) {
		return http.StatusBadRequest
	}
	var rbe *lib_model.ResourceBusyError
	if errors.As(err, &rbe) {
		return http.StatusConflict
	}
	var ie *lib_model.InternalError
	if errors.As(err, &ie) {
		return http.StatusInternalServerError
	}
	return 0
}
