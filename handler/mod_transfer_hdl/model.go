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
	"github.com/SENERGY-Platform/mgw-module-lib/validation/sem_ver"
	"github.com/SENERGY-Platform/mgw-module-manager/util/dir_fs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"os"
	"path"
	"slices"
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
	slices.SortStableFunc(versions, func(a, b string) int {
		res, _ := sem_ver.CompareSemVer(a, b)
		return res
	})
	return versions
}

func (r *modRepo) Get(ver string) (dir_fs.DirFS, error) {
	hash, ok := r.versions[ver]
	if !ok {
		return "", errors.New("not found")
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
	modDir, err := dir_fs.New(path.Join(r.gitPath, r.modPath))
	if err != nil {
		return "", err
	}
	err = dir_fs.Copy(modDir, vPth)
	if err != nil {
		return "", err
	}
	os.RemoveAll(path.Join(vPth, ".git"))
	verDir, err := dir_fs.New(vPth)
	if err != nil {
		return "", err
	}
	return verDir, nil
}

func (r *modRepo) Remove() error {
	return os.RemoveAll(r.path)
}
