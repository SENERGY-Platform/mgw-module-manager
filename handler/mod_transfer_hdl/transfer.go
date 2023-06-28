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
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"io"
	"io/fs"
	"os"
	"path"
	"regexp"
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
	// [REMINDER] clean workspace
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
	var versions map[string]plumbing.Hash
	versions, err = getVersions(repo, mPath)
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	var wt *git.Worktree
	wt, err = repo.Worktree()
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	return &modRepo{
		versions: versions,
		gitWt:    wt,
		gitPath:  cPth,
		modPath:  mPath,
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

func getVersions(repo *git.Repository, mPath string) (map[string]plumbing.Hash, error) {
	iter, err := repo.Tags()
	if err != nil {
		return nil, err
	}
	defer iter.Close()
	var regex *regexp.Regexp
	if mPath != "" {
		regex, err = regexp.Compile(`^` + strings.Replace(mPath, "/", "\\/", 0) + `\/(v[0-9]+(?:\.[0-9]+){0,2})$`)
		if err != nil {
			return nil, err
		}
	}
	versions := make(map[string]plumbing.Hash)
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
		if regex != nil {
			matches := regex.FindStringSubmatch(tag)
			if len(matches) == 2 {
				ver = matches[1]
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
		versions[ver] = ref.Hash()
	}
	return versions, nil
}
