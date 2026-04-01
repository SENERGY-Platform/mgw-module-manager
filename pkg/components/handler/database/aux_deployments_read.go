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

package handler_database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/slices"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/database"
)

func (h *Handler) ReadAuxiliaryDeployments(
	ctx context.Context,
	deploymentId string,
	filter models_handler_database.AuxiliaryDeploymentsFilter,
) (map[string]models_handler_database.AuxiliaryDeployment, error) {
	fc, val := genAuxiliaryDeploymentsFilter(deploymentId, filter)
	rows, err := h.sqlDB.QueryContext(
		ctx,
		"SELECT id, dep_id, image, ref, name, enabled, command, pseudo_tty, created, updated FROM aux_deployments"+fc+";",
		val...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	auxDeps := make(map[string]models_handler_database.AuxiliaryDeployment)
	for rows.Next() {
		var auxDep models_handler_database.AuxiliaryDeployment
		var ct, ut []uint8
		var command sql.NullString
		var pseudoTTY sql.NullBool
		err = rows.Scan(
			&auxDep.Id,
			&auxDep.DeploymentId,
			&auxDep.Image,
			&auxDep.Reference,
			&auxDep.Name,
			&auxDep.Enabled,
			&command,
			&pseudoTTY,
			&ct,
			&ut,
		)
		if err != nil {
			return nil, err
		}
		if auxDep.Created, err = time.Parse(timeLayout, string(ct)); err != nil {
			return nil, err
		}
		if auxDep.Updated, err = time.Parse(timeLayout, string(ut)); err != nil {
			return nil, err
		}
		auxDep.RunConfig.Command = command.String
		auxDep.RunConfig.PseudoTTY = pseudoTTY.Bool
		auxDeps[auxDep.Id] = auxDep
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return auxDeps, nil
}

func genAuxiliaryDeploymentsFilter(deploymentId string, filter models_handler_database.AuxiliaryDeploymentsFilter) (string, []any) {
	fc := []string{"dep_id = ?"}
	val := []any{deploymentId}
	if len(filter.Labels) > 0 {
		var tc int
		var str string
		for n, v := range filter.Labels {
			val = append(val, n, v)
			if tc == 0 {
				str = fmt.Sprintf("SELECT t%d.* FROM (SELECT aux_dep_id FROM aux_dep_labels WHERE name = ? AND value = ?) t%d", tc, tc)
			} else {
				str += fmt.Sprintf(" INNER JOIN (SELECT aux_dep_id FROM aux_dep_labels WHERE name = ? AND value = ?) t%d ON t%d.aux_dep_id = t%d.aux_dep_id", tc, tc-1, tc)
			}
			tc += 1
		}
		fc = append(fc, "id IN ("+str+")")
	}
	if len(filter.Ids) > 0 {
		ids := helper_slices.RemoveDuplicates(filter.Ids)
		fc = append(fc, "id IN ("+genQuestionMarks(len(ids))+")")
		for _, id := range ids {
			val = append(val, id)
		}
	}
	if filter.Enabled < 0 {
		fc = append(fc, "enabled = ?")
		val = append(val, false)
	}
	if filter.Enabled > 0 {
		fc = append(fc, "enabled = ?")
		val = append(val, true)
	}
	if filter.Image != "" {
		fc = append(fc, "image = ?")
		val = append(val, filter.Image)
	}
	if len(fc) > 0 {
		return " WHERE " + strings.Join(fc, " AND "), val
	}
	return "", nil
}
