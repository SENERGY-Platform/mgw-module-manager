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
	"context"
	"fmt"
	"github.com/SENERGY-Platform/mgw-module-manager/handler"
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"github.com/SENERGY-Platform/mgw-module-manager/util/dir_fs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"time"
)

type Handler struct {
	wrkSpcPath  string
	httpTimeout time.Duration
}

func New(workspacePath string, httpTimeout time.Duration) *Handler {
	return &Handler{
		wrkSpcPath:  workspacePath,
		httpTimeout: httpTimeout,
	}
}

func (h *Handler) InitWorkspace(perm fs.FileMode) error {
	if !path.IsAbs(h.wrkSpcPath) {
		return fmt.Errorf("workspace path must be absolute")
	}
	if err := os.MkdirAll(h.wrkSpcPath, perm); err != nil {
		return err
	}
	return nil
}

func (h *Handler) Get(ctx context.Context, mID string) (handler.ModRepo, error) {
	rPth, err := os.MkdirTemp(h.wrkSpcPath, "repo_")
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	defer func() {
		if err != nil {
			os.RemoveAll(rPth)
		}
	}()
	var cPth string
	cPth, err = os.MkdirTemp(rPth, "clone_")
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	defer os.RemoveAll(cPth)
	rPath, mPath := parseModID(mID)
	ctxWt, cf := context.WithTimeout(ctx, h.httpTimeout)
	defer cf()
	var repo *git.Repository
	repo, err = git.PlainCloneContext(ctxWt, cPth, false, &git.CloneOptions{
		URL:               "https://" + rPath + ".git",
		NoCheckout:        true,
		RecurseSubmodules: git.NoRecurseSubmodules,
		Tags:              git.AllTags,
	})
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	var versions map[string]dir_fs.DirFS
	versions, err = getVersions(repo, rPth, cPth, mPath)
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	return &modRepo{
		versions: versions,
		path:     rPth,
	}, nil
}

func parseModID(mID string) (repo string, pth string) {
	c := strings.Count(mID, "/")
	if c > 2 {
		i := strings.LastIndex(mID, "/")
		repo = mID[:i]
		pth = mID[i+1:]
	} else {
		repo = mID
	}
	return
}

func storeVersion(wt *git.Worktree, hash plumbing.Hash, rPath, cPath, mPath string) (dir_fs.DirFS, error) {
	if err := wt.Checkout(&git.CheckoutOptions{
		Hash:  hash,
		Force: true,
	}); err != nil {
		return "", err
	}
	vPth, err := os.MkdirTemp(rPath, "ver_")
	if err != nil {
		return "", err
	}
	err = util.CopyDir(path.Join(cPath, mPath), vPth)
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

func getVersions(repo *git.Repository, rPath, cPath, mPath string) (map[string]dir_fs.DirFS, error) {
	wt, err := repo.Worktree()
	if err != nil {
		return nil, err
	}
	iter, err := repo.Tags()
	if err != nil {
		return nil, err
	}
	defer iter.Close()
	versions := make(map[string]dir_fs.DirFS)
	for {
		ref, err := iter.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		tag := ref.Name().Short()
		var ver string
		if mPath != "" {
			if strings.Contains(tag, mPath) {
				ver = strings.Replace(tag, mPath+"/", "", 1)
			} else {
				continue
			}
		} else {
			if strings.Count(tag, "/") == 0 {
				ver = tag
			} else {
				continue
			}
		}
		vDir, err := storeVersion(wt, ref.Hash(), rPath, cPath, mPath)
		if err != nil {
			return nil, err
		}
		versions[ver] = vDir
	}
	return versions, nil
}
