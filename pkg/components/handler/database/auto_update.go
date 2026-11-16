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

package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	lib_errors "github.com/SENERGY-Platform/mgw-module-manager/lib/errors"
	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants/slog_keys"
)

// The auto-update state is a single row.
const autoUpdateRowId = 1

func (h *Handler) ReadAutoUpdate(ctx context.Context) (pkg_models.AutoUpdate, error) {
	var autoUpdate pkg_models.AutoUpdate
	var lastRunDate sql.NullString
	var lastRun []byte
	err := h.sqlDB.QueryRowContext(
		ctx,
		"SELECT enabled, level, window_start, window_end, time_zone, last_run_date, last_run FROM auto_update WHERE id = ?",
		autoUpdateRowId,
	).Scan(
		&autoUpdate.Enabled,
		&autoUpdate.Level,
		&autoUpdate.WindowStart,
		&autoUpdate.WindowEnd,
		&autoUpdate.TimeZone,
		&lastRunDate,
		&lastRun,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return pkg_models.AutoUpdate{}, lib_errors.New[lib_errors.ErrNotFound]("auto-update settings not found")
		}
		return pkg_models.AutoUpdate{}, err
	}
	autoUpdate.LastRunDate = lastRunDate.String
	autoUpdate.LastRun = decodeAutoUpdateLastRun(ctx, lastRun)
	return autoUpdate, nil
}

// decodeAutoUpdateLastRun returns nil for an undecodable run, it is a report only and must not lock the settings out.
func decodeAutoUpdateLastRun(ctx context.Context, b []byte) *lib_models.AutoUpdateRun {
	if len(b) == 0 {
		return nil
	}
	var run lib_models.AutoUpdateRun
	if err := json.Unmarshal(b, &run); err != nil {
		logger.WarnContext(ctx, "read auto-update, decode last run", slog_keys.Error, err)
		return nil
	}
	return &run
}

// WriteAutoUpdateSettings creates the row if it does not exist and leaves the last run untouched.
func (h *Handler) WriteAutoUpdateSettings(ctx context.Context, settings lib_models.AutoUpdateSettings) error {
	_, err := h.sqlDB.ExecContext(
		ctx,
		"INSERT INTO auto_update (id, enabled, level, window_start, window_end, time_zone) VALUES (?, ?, ?, ?, ?, ?) "+
			"ON DUPLICATE KEY UPDATE enabled = ?, level = ?, window_start = ?, window_end = ?, time_zone = ?",
		autoUpdateRowId,
		settings.Enabled,
		settings.Level,
		settings.WindowStart,
		settings.WindowEnd,
		settings.TimeZone,
		settings.Enabled,
		settings.Level,
		settings.WindowStart,
		settings.WindowEnd,
		settings.TimeZone,
	)
	return err
}

// WriteAutoUpdateRunDate requires the row to exist, it is only written for an enabled auto-update.
func (h *Handler) WriteAutoUpdateRunDate(ctx context.Context, date string) error {
	_, err := h.sqlDB.ExecContext(ctx, "UPDATE auto_update SET last_run_date = ? WHERE id = ?", date, autoUpdateRowId)
	return err
}

// WriteAutoUpdateLastRun requires the row to exist, it is only written for an enabled auto-update.
func (h *Handler) WriteAutoUpdateLastRun(ctx context.Context, run lib_models.AutoUpdateRun) error {
	b, err := json.Marshal(run)
	if err != nil {
		return err
	}
	// Sent as string, MySQL rejects JSON from a value with the binary character set.
	_, err = h.sqlDB.ExecContext(ctx, "UPDATE auto_update SET last_run = ? WHERE id = ?", string(b), autoUpdateRowId)
	return err
}
