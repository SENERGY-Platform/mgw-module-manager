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

package api

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"runtime"

	api_handler "github.com/SENERGY-Platform/mgw-module-manager/pkg/api/handler"
	pkg_service "github.com/SENERGY-Platform/mgw-module-manager/pkg/service"
	"github.com/gin-gonic/gin"
)

var standardApiHandlers = []handlerFunc[*pkg_service.Service]{
	api_handler.GetModule,
	api_handler.GetModules,
	api_handler.GetModulesChangeRequest,
	api_handler.CreateModulesChangeRequest,
	api_handler.ExecModulesChangeRequest,
	api_handler.CancelModulesChangeRequest,
	api_handler.GetModulesAvailableUpdatesCount,
	api_handler.RefreshRepositories,
	api_handler.GetRepositoryModules,
	api_handler.CreateGlobalConfig,
	api_handler.GetGlobalConfig,
	api_handler.GetGlobalConfigs,
	api_handler.UpdateGlobalConfig,
	api_handler.DeleteGlobalConfig,
	api_handler.DeleteGlobalConfigs,
}

var restrictedApiHandlers = []handlerFunc[*pkg_service.Service]{}

var sharedApiHandlers = []handlerFunc[*pkg_service.Service]{}

type handlerFunc[T any] func(srv T) (method, path string, handlerFunc gin.HandlerFunc)

func registerHandlers[T any](ginEngine *gin.Engine, srv T, handlers ...handlerFunc[T]) error {
	paths := make(map[string]handlerFunc[T])
	for _, hf := range handlers {
		m, p, ginHf := hf(srv)
		if tmpHf, ok := paths[m+ginEngine.BasePath()+p]; ok {
			if reflect.ValueOf(tmpHf) == reflect.ValueOf(hf) {
				continue
			}
			return errors.New(
				fmt.Sprintf(
					"handler conflict: '%s %s' mapped by '%s' and '%s'",
					m,
					ginEngine.BasePath()+p,
					getFuncName(tmpHf),
					getFuncName(hf),
				),
			)
		}
		ginEngine.Handle(m, p, ginHf)
		paths[m+ginEngine.BasePath()+p] = hf
		_, _ = fmt.Fprintf(os.Stderr, "register http engine handler: %s %s\n", m, ginEngine.BasePath()+p)
	}
	return nil
}

func getFuncName(i any) string {
	val := reflect.ValueOf(i)
	f := runtime.FuncForPC(val.Pointer())
	if f != nil {
		return f.Name()
	}
	return fmt.Sprintf("addr=%+v", i)
}
