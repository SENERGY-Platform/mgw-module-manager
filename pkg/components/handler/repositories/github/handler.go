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

package github

import (
	"context"
	"encoding/json"
	"os"
	"path"
	"strings"
	"sync"

	lib_errors "github.com/SENERGY-Platform/mgw-module-manager/lib/errors"
	handler_repositories "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/repositories"
)

const (
	reposDir   = "repositories"
	sourcesDir = "sources"
)

type Handler struct {
	repositories map[string]*Repository
	gitHubClient gitHubClient
	workdirPath  string
	mu           sync.RWMutex
}

func New(gitHubClient gitHubClient, workdirPath string) *Handler {
	return &Handler{
		gitHubClient: gitHubClient,
		workdirPath:  workdirPath,
	}
}

func (h *Handler) RepositoryType() string {
	return gitHubCom
}

func (h *Handler) Init(_ context.Context) error {
	err := os.MkdirAll(path.Join(h.workdirPath, sourcesDir), 0775)
	if err != nil {
		return err
	}
	dirEntries, err := os.ReadDir(path.Join(h.workdirPath, sourcesDir))
	if err != nil {
		return err
	}
	h.repositories = make(map[string]*Repository)
	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			continue
		}
		source, err := readSourceFile(path.Join(h.workdirPath, sourcesDir, dirEntry.Name()))
		if err != nil {
			return err
		}
		h.repositories[getSourceString(source)] = newRepository(
			h.gitHubClient,
			source,
			path.Join(h.workdirPath, reposDir, getFsName(source)),
		)
	}
	return nil
}

func (h *Handler) GetRepositories(_ context.Context) (map[string]handler_repositories.Repository, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	tmp := make(map[string]handler_repositories.Repository)
	for src, repo := range h.repositories {
		tmp[src] = repo
	}
	return tmp, nil
}

func (h *Handler) GetRepository(_ context.Context, source string) (handler_repositories.Repository, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	repo, ok := h.repositories[source]
	if !ok {
		return nil, lib_errors.New[lib_errors.ErrNotFound]("source not found")
	}
	return repo, nil
}

func (h *Handler) CreateRepository(_ context.Context, data []byte) (handler_repositories.Repository, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	var src Source
	err := json.Unmarshal(data, &src)
	if err != nil {
		return nil, err
	}
	srcString := getSourceString(src)
	_, ok := h.repositories[srcString]
	if !ok {
		return nil, lib_errors.New[lib_errors.ErrExists]("source already exists")
	}
	fsName := getFsName(src)
	err = writeSourceFile(path.Join(h.workdirPath, sourcesDir, fsName), src)
	if err != nil {
		return nil, err
	}
	repo := newRepository(h.gitHubClient, src, path.Join(h.workdirPath, reposDir, fsName))
	h.repositories[srcString] = repo
	return repo, nil
}

func (h *Handler) DeleteRepository(_ context.Context, source string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	repo, ok := h.repositories[source]
	if !ok {
		return nil
	}
	fsName := getFsName(repo.source)
	err := os.RemoveAll(path.Join(h.workdirPath, reposDir, fsName))
	if err != nil {
		return err
	}
	err = os.RemoveAll(path.Join(h.workdirPath, sourcesDir, fsName))
	if err != nil {
		return err
	}
	delete(h.repositories, source)
	return nil
}

func readSourceFile(p string) (Source, error) {
	file, err := os.Open(p)
	if err != nil {
		return Source{}, err
	}
	defer file.Close()
	jd := json.NewDecoder(file)
	var src Source
	err = jd.Decode(&src)
	if err != nil {
		return Source{}, err
	}
	return src, nil
}

func writeSourceFile(p string, src Source) error {
	file, err := os.Create(p)
	if err != nil {
		return err
	}
	defer file.Close()
	je := json.NewEncoder(file)
	je.SetIndent("", "\t")
	return je.Encode(src)
}

func getSourceString(src Source) string {
	return path.Join(gitHubCom, src.Owner, src.Repository)
}

func getFsName(src Source) string {
	return strings.Replace(strings.Replace(src.Owner+"_"+src.Repository+"_"+src.Reference, "/", "_", -1), ".", "_", -1)
}
