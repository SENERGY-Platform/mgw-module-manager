/*
 * Copyright 2025 InfAI (CC SES)
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

package github_modules_repo

import (
	"context"
	github_clt2 "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/github_modules_repo/github_clt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestHandler_Init(t *testing.T) {
	tempDir := t.TempDir()
	h := New(nil, time.Second, tempDir, "test_owner", "test_repo", []Channel{
		{
			Name: "test_channel",
		},
	})
	err := h.Init()
	if err != nil {
		t.Error(err)
	}
	_, err = os.Stat(filepath.Join(tempDir, "github_com_test_owner_test_repo/test_channel"))
	if err != nil {
		t.Error(err)
	}
}

func TestHandler_Source(t *testing.T) {
	h := New(nil, time.Second, "", "test_owner", "test_repo", nil)
	if h.Source() != "github.com/test_owner/test_repo" {
		t.Errorf("expect github.com/test_owner/test_repo, got %s", h.Source())
	}
}

func TestHandler_Channels(t *testing.T) {
	h := New(nil, time.Second, "", "test_owner", "test_repo", []Channel{
		{
			Name: "test_channel",
		},
	})
	a := []string{"test_channel"}
	b := h.Channels()
	if !reflect.DeepEqual(a, b) {
		t.Errorf("expect %v, got %v", a, b)
	}
}

func TestHandler_DefaultChannel(t *testing.T) {
	h := New(nil, time.Second, "", "test_owner", "test_repo", []Channel{
		{
			Name:    "test_channel_1",
			Default: true,
		},
		{
			Name: "test_channel_2",
		},
	})
	if h.DefaultChannel() != "test_channel_1" {
		t.Errorf("expect test_channel_1, got %s", h.DefaultChannel())
	}
}

func TestHandler_FileSystemsMap(t *testing.T) {
	h := New(nil, time.Second, "./test", "test_owner", "test_repo", []Channel{
		{
			Name:      "test_channel",
			Reference: "test_ref",
			Blacklist: []string{"test_dir"},
		},
	})
	a := map[string]fs.FS{
		"test_mod_1": os.DirFS("test/github_com_test_owner_test_repo/test_channel/sha_ref/mods/test_mod_1"),
		"test_mod_2": os.DirFS("test/github_com_test_owner_test_repo/test_channel/sha_ref/mods/test_mod_2"),
	}
	b, err := h.FileSystemsMap(context.Background(), "test_channel")
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Errorf("expected %v, got %v", a, b)
	}
	t.Run("no repo file", func(t *testing.T) {
		h2 := New(nil, time.Second, "./test", "test_owner", "test_repo_2", []Channel{
			{
				Name:      "test_channel",
				Reference: "test_ref",
			},
		})
		fsMap, err := h2.FileSystemsMap(context.Background(), "test_channel")
		if err != nil {
			t.Error(err)
		}
		if len(fsMap) > 0 {
			t.Error("expect empty map")
		}
	})
	t.Run("error", func(t *testing.T) {
		_, err = h.FileSystemsMap(context.Background(), "test")
		if err == nil {
			t.Error("expected error")
		}
	})
}

func TestHandler_FileSystem(t *testing.T) {
	h := New(nil, time.Second, "./test", "test_owner", "test_repo", []Channel{
		{
			Name:      "test_channel",
			Reference: "test_ref",
			Blacklist: []string{"test_dir"},
		},
	})
	a := os.DirFS("test/github_com_test_owner_test_repo/test_channel/sha_ref/mods/test_mod_1")
	b, err := h.FileSystem(context.Background(), "test_channel", "test_mod_1")
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Errorf("expected %v, got %v", a, b)
	}
	t.Run("error", func(t *testing.T) {
		t.Run("fs ref does not exist", func(t *testing.T) {
			_, err = h.FileSystem(context.Background(), "test_channel", "test")
			if err == nil {
				t.Error("expected error")
			}
		})
		t.Run("channel does not exist", func(t *testing.T) {
			_, err = h.FileSystem(context.Background(), "test", "test_mod_1")
			if err == nil {
				t.Error("expected error")
			}
		})
	})
}

func TestHandler_Refresh(t *testing.T) {
	mockClient := &gitHubClientMock{
		Commits: map[string]map[string]map[string]github_clt2.GitCommit{
			"test_owner": {
				"test_repo": {
					"test_ref": {
						Sha:  "1234",
						Date: time.Now(),
					},
				},
			},
		},
		Archives: map[string]map[string]map[string]string{
			"test_owner": {
				"test_repo": {
					"1234": "./test/test.tar.gz",
				},
			},
		},
	}
	tempDir := t.TempDir()
	h := New(mockClient, time.Second, tempDir, "test_owner", "test_repo", []Channel{
		{
			Name:      "test_channel",
			Reference: "test_ref",
		},
	})
	err := h.Refresh(context.Background())
	if err != nil {
		t.Error(err)
	}
	rf, err := readRepoFile(path.Join(tempDir, "github_com_test_owner_test_repo/test_channel"))
	if err != nil {
		t.Error(err)
	}
	if rf.GitCommit.Sha != "1234" {
		t.Errorf("expect 1234, got %s", rf.GitCommit.Sha)
	}
	if rf.Path != "1234/test" {
		t.Errorf("expect test, got %s", rf.Path)
	}
	_, err = os.Stat(path.Join(tempDir, "github_com_test_owner_test_repo/test_channel/1234/test/test_mod/Modfile.yml"))
	if err != nil {
		t.Error(err)
	}
	t.Run("refresh existing", func(t *testing.T) {
		mockClient.Commits = map[string]map[string]map[string]github_clt2.GitCommit{
			"test_owner": {
				"test_repo": {
					"test_ref": {
						Sha:  "5678",
						Date: time.Now(),
					},
				},
			},
		}
		mockClient.Archives = map[string]map[string]map[string]string{
			"test_owner": {
				"test_repo": {
					"5678": "./test/test.tar.gz",
				},
			},
		}
		err = h.Refresh(context.Background())
		if err != nil {
			t.Error(err)
		}
		rf, err = readRepoFile(path.Join(tempDir, "github_com_test_owner_test_repo/test_channel"))
		if err != nil {
			t.Error(err)
		}
		if rf.GitCommit.Sha != "5678" {
			t.Errorf("expect 5678, got %s", rf.GitCommit.Sha)
		}
		if rf.Path != "5678/test" {
			t.Errorf("expect test, got %s", rf.Path)
		}
		_, err = os.Stat(path.Join(tempDir, "github_com_test_owner_test_repo/test_channel/5678/test/test_mod/Modfile.yml"))
		if err != nil {
			t.Error(err)
		}
		_, err = os.Stat(path.Join(tempDir, "github_com_test_owner_test_repo/test_channel/1234"))
		if err != nil {
			if !os.IsNotExist(err) {
				t.Error(err)
			}
		} else {
			t.Error("expected error")
		}
	})
}

type gitHubClientMock struct {
	Err      error
	Commits  map[string]map[string]map[string]github_clt2.GitCommit
	Archives map[string]map[string]map[string]string
}

func (m *gitHubClientMock) GetLastCommit(ctx context.Context, owner, repo, ref string) (github_clt2.GitCommit, error) {
	if m.Err != nil {
		return github_clt2.GitCommit{}, m.Err
	}
	repos, ok := m.Commits[owner]
	if !ok {
		return github_clt2.GitCommit{}, github_clt2.NewResponseError(http.StatusNotFound, "commit: owner not found")
	}
	refs, ok := repos[repo]
	if !ok {
		return github_clt2.GitCommit{}, github_clt2.NewResponseError(http.StatusNotFound, "commit: repo not found")
	}
	commit, ok := refs[ref]
	if !ok {
		return github_clt2.GitCommit{}, github_clt2.NewResponseError(http.StatusNotFound, "commit: ref not found")
	}
	return commit, nil
}

func (m *gitHubClientMock) GetRepoTarGzArchive(ctx context.Context, owner, repo, ref string) (io.ReadCloser, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	repos, ok := m.Archives[owner]
	if !ok {
		return nil, github_clt2.NewResponseError(http.StatusNotFound, "archive: owner not found")
	}
	refs, ok := repos[repo]
	if !ok {
		return nil, github_clt2.NewResponseError(http.StatusNotFound, "archive: repo not found")
	}
	filePath, ok := refs[ref]
	if !ok {
		return nil, github_clt2.NewResponseError(http.StatusNotFound, "archive: ref not found")
	}
	return os.Open(filePath)
}
