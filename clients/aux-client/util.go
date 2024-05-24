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
	"context"
	job_hdl_lib "github.com/SENERGY-Platform/go-service-base/job-hdl/lib"
	"github.com/SENERGY-Platform/mgw-module-manager/lib"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"net/http"
	"strings"
	"time"
)

func AwaitJob(ctx context.Context, client lib.AuxDeploymentApi, dID, jID string, delay, httpTimeout time.Duration, logger interface{ Error(arg ...any) }) (job_hdl_lib.Job, error) {
	ticker := time.NewTicker(delay)
	defer ticker.Stop()
	var cfs []context.CancelFunc
	defer func() {
		for _, cf := range cfs {
			cf()
		}
	}()
	for {
		select {
		case <-ctx.Done():
			c, cf := context.WithTimeout(context.Background(), httpTimeout)
			err := client.CancelAuxJob(c, dID, jID)
			if err != nil && logger != nil {
				logger.Error(err)
			}
			cf()
			return job_hdl_lib.Job{}, ctx.Err()
		case <-ticker.C:
			c, cf := context.WithTimeout(context.Background(), httpTimeout)
			cfs = append(cfs, cf)
			j, err := client.GetAuxJob(c, dID, jID)
			if err != nil {
				return job_hdl_lib.Job{}, err
			}
			if j.Completed != nil {
				return j, nil
			}
		}
	}
}

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
