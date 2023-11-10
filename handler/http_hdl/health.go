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

package http_hdl

import (
	"github.com/SENERGY-Platform/mgw-module-manager/lib"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/gin-gonic/gin"
	"net/http"
)

func getServiceHealthH(a lib.Api) gin.HandlerFunc {
	return func(gc *gin.Context) {
		_, err := a.GetDeployments(gc.Request.Context(), model.DepFilter{}, false, false)
		if err != nil {
			_ = gc.Error(err)
			return
		}
		_, err = a.GetModules(gc.Request.Context(), model.ModFilter{})
		if err != nil {
			_ = gc.Error(err)
			return
		}
		_, err = a.GetJobs(gc.Request.Context(), model.JobFilter{})
		if err != nil {
			_ = gc.Error(err)
			return
		}
		gc.Status(http.StatusOK)
	}
}
