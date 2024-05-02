/*
 * Copyright 2024 InfAI (CC SES)
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

package aux_client

import (
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"net/http"
	"strings"
)

func genLabels(m map[string]string, eqs, sep string) string {
	var sl []string
	for k, v := range m {
		if v != "" {
			sl = append(sl, k+eqs+v)
		} else {
			sl = append(sl, k)
		}
	}
	return strings.Join(sl, sep)
}

func setDepIdHeader(r *http.Request, dID string) {
	r.Header.Set(model.AuxDepIdHeaderKey, dID)
}
