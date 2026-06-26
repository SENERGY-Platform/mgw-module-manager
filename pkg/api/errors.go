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

package api

import (
	"errors"
	"fmt"
	"net/http"

	lib_constants "github.com/SENERGY-Platform/mgw-module-manager/lib/constants"
	lib_errors "github.com/SENERGY-Platform/mgw-module-manager/lib/errors"
	"github.com/gin-gonic/gin"
)

func getCodes(err error) (int, string) {
	for {
		switch err.(type) {
		case *lib_errors.ErrNotFound:
			return http.StatusNotFound, "001"
		case *lib_errors.ErrExists:
			return http.StatusBadRequest, "002"
		case *lib_errors.ErrInvalidInput:
			return http.StatusBadRequest, "003"
		case *lib_errors.ErrActiveJob:
			return http.StatusServiceUnavailable, "004"
		}
		err = errors.Unwrap(err)
		if err == nil {
			break
		}
	}
	return 0, ""
}

func errorHandler(format string) gin.HandlerFunc {
	return func(gc *gin.Context) {
		gc.Next()
		if !gc.IsAborted() && len(gc.Errors) > 0 {
			var statusCode int
			var errCode string
			var errs []error
			for _, err := range gc.Errors {
				tmpSC, tmpEC := getCodes(err)
				if tmpSC > statusCode {
					statusCode = tmpSC
					errCode = tmpEC
				}
				errs = append(errs, err)
			}
			if statusCode == 0 {
				statusCode = http.StatusInternalServerError
			}
			if errCode != "" {
				gc.Header(lib_constants.HttpHeaderErrorCode, errCode)
			}
			gc.String(statusCode, combineErrorMessages(format, errs))
		}
	}
}

func combineErrorMessages(format string, errs []error) string {
	if len(errs) == 0 {
		return ""
	}
	if len(errs) == 1 {
		return errs[0].Error()
	}
	var msg string
	msg += fmt.Sprintf(format, 0, errs[0].Error())
	for i, err := range errs[1:] {
		msg += ", " + fmt.Sprintf(format, i+1, err.Error())
	}
	return msg
}
