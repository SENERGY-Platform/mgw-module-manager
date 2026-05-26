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
	"path"
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
	handlers.GetDeploymentRequest,
	handlers.CreateDeployments,
	handlers.UpdateDeployments,
	handlers.RecreateDeployments,
	handlers.DeleteDeployments,
	handlers.EnableDeployments,
	handlers.DisableDeployments,
	handlers.GetDeploymentsJobResult,
	handlers.GetUpdateDeploymentsJobResult,
	handlers.GetModuleChangeJobResult,
	handlers.GetRefreshRepositoriesJobResult,
	handlers.ServiceHealth,
}

var restrictedApiHandlers = []handlerFunc[*service.Service]{
	handlers.CreateAuxiliaryDeployment,
	handlers.UpdateAuxiliaryDeployment,
	handlers.RecreateAuxiliaryDeployments,
	handlers.DeleteAuxiliaryDeployment,
	handlers.DeleteAuxiliaryDeployments,
	handlers.EnableAuxiliaryDeployments,
	handlers.DisableAuxiliaryDeployments,
	handlers.DeleteAuxiliaryDeploymentVolumes,
	handlers.GetDeploymentAdvertisement,
	handlers.GetDeploymentAdvertisementById,
	handlers.GetDeploymentAdvertisements,
	handlers.PutDeploymentAdvertisement,
	handlers.PutDeploymentAdvertisements,
	handlers.DeleteDeploymentAdvertisements,
}

var sharedApiHandlers = []handlerFunc[*service.Service]{
	handlers.GetAuxiliaryDeployment,
	handlers.GetAuxiliaryDeployments,
	handlers.GetReducedAuxiliaryDeployments,
	handlers.GetAuxiliaryDeploymentVolumes,
	handlers.GetAuxiliaryDeploymentVolumesWithMounts,
	handlers.QueryDeploymentAdvertisements,
	handlers.QueryDeploymentAdvertisement,
	handlers.GetCreateAuxiliaryDeploymentJobResult,
	handlers.GetUpdateAuxiliaryDeploymentJobResult,
	handlers.GetAuxiliaryDeploymentsJobResult,
	handlers.GetJobs,
	handlers.GetJob,
	handlers.CancelJobs,
	handlers.CancelJob,
	handlers.DeploymentsHealth,
	handlers.ServiceInfo,
}

type handlerFunc[T any] func(srv T) (method, path string, handlerFunc gin.HandlerFunc)

type httpEngine interface {
	BasePath() string
	Handle(httpMethod, relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
}

func registerHandlers[T any](engine httpEngine, srv T, handlers ...handlerFunc[T]) error {
	paths := make(map[string]handlerFunc[T])
	for _, hf := range handlers {
		m, p, ginHf := hf(srv)
		if tmpHf, ok := paths[m+engine.BasePath()+p]; ok {
			if reflect.ValueOf(tmpHf) == reflect.ValueOf(hf) {
				continue
			}
			return errors.New(
				fmt.Sprintf(
					"handler conflict: '%s %s' mapped by '%s' and '%s'",
					m,
					path.Join(engine.BasePath(), p),
					getFuncName(tmpHf),
					getFuncName(hf),
				),
			)
		}
		engine.Handle(m, p, ginHf)
		paths[m+engine.BasePath()+p] = hf
		_, _ = fmt.Fprintf(os.Stderr, "register http engine handler: %-7s %s\n", m, path.Join(engine.BasePath(), p))
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
