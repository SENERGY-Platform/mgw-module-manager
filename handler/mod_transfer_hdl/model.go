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

package mod_transfer_hdl

import (
	"errors"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"github.com/SENERGY-Platform/mgw-module-manager/util/dir_fs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"os"
	"path"
	"sort"
)

type modRepo struct {
	versions map[string]plumbing.Hash
	gitWt    *git.Worktree
	gitPath  string
	modPath  string
	path     string
}

func (r *modRepo) Versions() []string {
	var versions []string
	for ver := range r.versions {
		versions = append(versions, ver)
	}
	sort.Strings(versions)
	return versions
}

func (r *modRepo) Get(ver string) (dir_fs.DirFS, error) {
	hash, ok := r.versions[ver]
	if !ok {
		return "", errors.New("version not found")
	}
	if err := r.gitWt.Checkout(&git.CheckoutOptions{
		Hash:  hash,
		Force: true,
	}); err != nil {
		return "", err
	}
	vPth, err := os.MkdirTemp(r.path, "ver_")
	if err != nil {
		return "", err
	}
	err = util.CopyDir(path.Join(r.gitPath, r.modPath), vPth)
	if err != nil {
		return "", err
	}
	os.RemoveAll(path.Join(vPth, ".git"))
	dir, err := dir_fs.New(vPth)
	if err != nil {
		return "", err
	}
	return dir, nil
}

func (r *modRepo) Remove() error {
	return os.RemoveAll(r.path)
}
