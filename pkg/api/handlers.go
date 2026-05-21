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

	"github.com/SENERGY-Platform/mgw-module-manager/pkg/api/handlers"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/service"
	"github.com/gin-gonic/gin"
)

var standardApiHandlers = []handlerFunc[*service.Service]{
	handlers.GetModule,
	handlers.GetModules,
	handlers.GetModulesChangeRequest,
	handlers.CreateModulesChangeRequest,
	handlers.ExecModulesChangeRequest,
	handlers.CancelModulesChangeRequest,
	handlers.GetModulesAvailableUpdatesCount,
	handlers.RefreshRepositories,
	handlers.GetRepositoryModules,
	handlers.CreateGlobalConfig,
	handlers.GetGlobalConfig,
	handlers.GetGlobalConfigs,
	handlers.UpdateGlobalConfig,
	handlers.DeleteGlobalConfig,
	handlers.DeleteGlobalConfigs,
}

var restrictedApiHandlers = []handlerFunc[*service.Service]{}

var sharedApiHandlers = []handlerFunc[*service.Service]{}

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
