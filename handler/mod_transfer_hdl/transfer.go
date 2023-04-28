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
	"github.com/SENERGY-Platform/mgw-module-manager/lib/model"
	"github.com/SENERGY-Platform/mgw-module-manager/util"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"io"
	"os"
	"path"
	"time"
)

type Handler struct {
	wrkSpcPath  string
	httpTimeout time.Duration
}

func New(workspacePath string, httpTimeout time.Duration) *Handler {
	return &Handler{wrkSpcPath: workspacePath, httpTimeout: httpTimeout}
}

func (h *Handler) InitWorkspace() error {
	if err := os.MkdirAll(h.wrkSpcPath, 0777); err != nil {
		return err
	}
	return nil
}

func (h *Handler) ListVersions(ctx context.Context, mID string) ([]string, error) {
	dir, err := os.MkdirTemp(h.wrkSpcPath, "list_")
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	defer os.RemoveAll(dir)
	ctxWt, cf := context.WithTimeout(ctx, h.httpTimeout)
	defer cf()
	repo, err := git.PlainCloneContext(ctxWt, dir, false, &git.CloneOptions{
		URL:               "https://" + mID + ".git",
		NoCheckout:        true,
		RecurseSubmodules: git.NoRecurseSubmodules,
		Tags:              git.AllTags,
	})
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	iter, err := repo.Tags()
	if err != nil {
		return nil, model.NewInternalError(err)
	}
	defer iter.Close()
	var versions []string
	for {
		ref, err := iter.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, model.NewInternalError(err)
		}
		versions = append(versions, ref.Name().Short())
	}
	return versions, nil
}

func (h *Handler) Get(ctx context.Context, mID, ver, sub string) (dir util.DirFS, err error) {
	tDir, err := os.MkdirTemp(h.wrkSpcPath, "clone_")
	if err != nil {
		return "", model.NewInternalError(err)
	}
	defer func() {
		if err != nil {
			os.RemoveAll(tDir)
		}
	}()
	ctxWt, cf := context.WithTimeout(ctx, h.httpTimeout)
	defer cf()
	_, err = git.PlainCloneContext(ctxWt, tDir, false, &git.CloneOptions{
		URL:               "https://" + mID + ".git",
		ReferenceName:     plumbing.NewTagReferenceName(path.Join(sub, ver)),
		SingleBranch:      true,
		Depth:             1,
		RecurseSubmodules: git.NoRecurseSubmodules,
		Tags:              git.NoTags,
	})
	if err != nil {
		return "", model.NewInternalError(err)
	}
	var p string
	if sub == "" {
		err = os.RemoveAll(path.Join(tDir, ".git"))
		if err != nil {
			return "", model.NewInternalError(err)
		}
		p = tDir
	} else {
		p, err = os.MkdirTemp(h.wrkSpcPath, "clone_")
		if err != nil {
			return "", model.NewInternalError(err)
		}
		p = path.Join(tDir, sub)
		err = util.CopyDir(path.Join(tDir, sub), p)
		if err != nil {
			os.RemoveAll(p)
			return "", model.NewInternalError(err)
		}
		os.RemoveAll(tDir)
	}
	dir, err = util.NewDirFS(p)
	if err != nil {
		return "", model.NewInternalError(err)
	}
	return dir, nil
}