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

package mod_storage_hdl

import (
	"encoding/json"
	"errors"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"io"
	"io/fs"
	"os"
	"path"
	"sync"
	"time"
)

const indexFile = "index"

type item struct {
	ID       string    `json:"id"`
	Dir      string    `json:"dir"`
	ModFile  string    `json:"modfile"`
	Indirect bool      `json:"indirect"`
	Added    time.Time `json:"added"`
	Updated  time.Time `json:"updated"`
}

type indexHandler struct {
	index map[string]item
	path  string
	mu    sync.RWMutex
}

func newIndexHandler(pth string) *indexHandler {
	return &indexHandler{path: path.Join(pth, indexFile)}
}

func (h *indexHandler) Init() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := os.Stat(h.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			file, err := os.Create(h.path)
			if err != nil {
				return err
			}
			defer file.Close()
		} else {
			return err
		}
	}
	return h.read()
}

func (h *indexHandler) List() map[string]item {
	h.mu.RLock()
	defer h.mu.RUnlock()
	index := make(map[string]item)
	for key, val := range h.index {
		index[key] = val
	}
	return index
}

func (h *indexHandler) Get(id string) (item, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	i, ok := h.index[id]
	if !ok {
		return item{}, model.NewNotFoundError(errors.New("not found"))
	}
	return i, nil
}

func (h *indexHandler) Add(id, dir, modFile string, indirect bool) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	_, ok := h.index[id]
	if ok {
		return model.NewInvalidInputError(errors.New("already exists"))
	}
	t := time.Now().UTC()
	h.index[id] = item{
		ID:       id,
		Dir:      dir,
		ModFile:  modFile,
		Indirect: indirect,
		Added:    t,
		Updated:  t,
	}
	return h.write()
}

func (h *indexHandler) Delete(id string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	_, ok := h.index[id]
	if ok {
		delete(h.index, id)
		return h.write()
	}
	return nil
}

func (h *indexHandler) read() error {
	file, err := os.Open(h.path)
	if err != nil {
		return err
	}
	defer file.Close()
	h.index = make(map[string]item)
	jd := json.NewDecoder(file)
	for {
		var i item
		if err := jd.Decode(&i); err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		h.index[i.ID] = i
	}
	return nil
}

func (h *indexHandler) write() error {
	tmpPth := h.path + "_tmp"
	file, err := os.OpenFile(tmpPth, os.O_CREATE|os.O_TRUNC|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	je := json.NewEncoder(file)
	je.SetIndent("", "")
	for _, i := range h.index {
		err := je.Encode(i)
		if err != nil {
			return err
		}
	}
	err = os.Remove(h.path)
	if err != nil {
		return err
	}
	err = os.Rename(tmpPth, h.path)
	if err != nil {
		return err
	}
	return nil
}
