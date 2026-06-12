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

package host_dir

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"sync"

	module_lib_validation "github.com/SENERGY-Platform/mgw-module-lib/validation"
	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	handler_repositories "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/repositories"
	helper_errors "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/errors"
	helper_modfile "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/modfile"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/constants/slog_keys"
)

const repoType = "host-dir"
const sourceName = "localhost"
const channelName = "default"

type Handler struct {
	workdirPath string
	priority    int
	fileSystems map[string]fs.FS
	mu          sync.RWMutex
}

func New(workdirPath string, priority int) *Handler {
	return &Handler{
		workdirPath: workdirPath,
		priority:    priority,
	}
}

func (h *Handler) Type() string {
	return repoType
}

func (h *Handler) Priority() int {
	return h.priority
}

func (h *Handler) Source() string {
	return sourceName
}

func (h *Handler) Channels() []lib_models.RepositoryChannel {
	return []lib_models.RepositoryChannel{
		{
			Name:     channelName,
			Priority: 0,
		},
	}
}

func (h *Handler) Refresh(_ context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	dirEntries, err := os.ReadDir(h.workdirPath)
	if err != nil {
		return err
	}
	var errs []error
	for _, dirEntry := range dirEntries {
		if !dirEntry.IsDir() {
			continue
		}
		err = validateModule(os.DirFS(path.Join(h.workdirPath, dirEntry.Name())))
		if err != nil {
			errs = append(errs, fmt.Errorf("'%s' %w", dirEntry.Name(), err))
		}
	}
	if len(errs) > 0 {
		return helper_errors.Join(errs...)
	}
	return nil
}

func (h *Handler) GetFileSystemsMap(ctx context.Context, channel string) (map[string]fs.FS, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if channel != channelName {
		return nil, errors.New(fmt.Sprintf("channel '%s' not defined", channel))
	}
	dirEntries, err := os.ReadDir(h.workdirPath)
	if err != nil {
		return nil, err
	}
	fsMap := make(map[string]fs.FS)
	for _, dirEntry := range dirEntries {
		if !dirEntry.IsDir() {
			continue
		}
		dirFs := os.DirFS(path.Join(h.workdirPath, dirEntry.Name()))
		err = validateModule(dirFs)
		if err != nil {
			logger.ErrorContext(ctx, "get file systems map", slog_keys.DirName, dirEntry.Name(), slog_keys.Error, err)
			continue
		}
		fsMap[dirEntry.Name()] = dirFs
	}
	return fsMap, nil
}

func (h *Handler) GetFileSystem(_ context.Context, channel, fsRef string) (fs.FS, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if channel != channelName {
		return nil, errors.New(fmt.Sprintf("channel '%s' not defined", channel))
	}
	dirEntries, err := os.ReadDir(h.workdirPath)
	if err != nil {
		return nil, err
	}
	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() && dirEntry.Name() == fsRef {
			dirFs := os.DirFS(path.Join(h.workdirPath, dirEntry.Name()))
			err = validateModule(dirFs)
			if err != nil {
				return nil, fmt.Errorf("'%s' %w", dirEntry.Name(), err)
			}
			return dirFs, nil
		}
	}
	return nil, errors.New("reference not found")
}

func (h *Handler) Init() error {
	return os.MkdirAll(h.workdirPath, 0775)
}

func (h *Handler) RepositoryType() string {
	return repoType
}

func (h *Handler) GetRepositories(_ context.Context) (map[string]handler_repositories.Repository, error) {
	return map[string]handler_repositories.Repository{sourceName: h}, nil
}

func (h *Handler) GetRepository(_ context.Context, source string) (handler_repositories.Repository, error) {
	if source != sourceName {
		return nil, errors.New(fmt.Sprintf("source '%s' not defined", source))
	}
	return h, nil
}

func (h *Handler) CreateRepository(_ context.Context, _ []byte) error {
	return errors.New("not supported")
}

func (h *Handler) DeleteRepository(_ context.Context, _ string) error {
	return errors.New("not supported")
}

func validateModule(dirFs fs.FS) error {
	mod, err := helper_modfile.GetModule(dirFs)
	if err != nil {
		return err
	}
	err = module_lib_validation.Validate(mod)
	if err != nil {
		return err
	}
	return nil
}
