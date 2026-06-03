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

package github

import (
	"context"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"reflect"
	"testing"
	"time"

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/repositories/github/client"
)

func TestHandler_Source(t *testing.T) {
	r := newRepository(
		nil,
		Source{
			Owner:      "test_owner",
			Repository: "test_repo",
			Reference:  "test_ref",
		},
		"",
	)
	if r.Source() != "github.com/test_owner/test_repo" {
		t.Errorf("expect github.com/test_owner/test_repo, got %s", r.Source())
	}
}

func TestHandler_Channels(t *testing.T) {
	r := newRepository(
		nil,
		Source{
			Channels: []Channel{
				{
					Name:     "test_channel",
					Priority: 1,
				},
			},
		},
		"",
	)
	a := []lib_models.RepositoryChannel{{Name: "test_channel", Priority: 1}}
	b := r.Channels()
	if !reflect.DeepEqual(a, b) {
		t.Errorf("expect %v, got %v", a, b)
	}
}

func TestHandler_FileSystemsMap(t *testing.T) {
	r := newRepository(
		nil,
		Source{
			Owner:      "test_owner",
			Repository: "test_repo",
			Reference:  "test_ref",
			Channels: []Channel{
				{
					Name:      "test_channel",
					Blacklist: []string{"test_dir"},
				},
			},
		},
		"./test/repo_1",
	)
	a := map[string]fs.FS{
		"test_mod_1": os.DirFS("test/repo_1/sha_ref/mods/test_channel/test_mod_1"),
		"test_mod_2": os.DirFS("test/repo_1/sha_ref/mods/test_channel/test_mod_2"),
	}
	b, err := r.GetFileSystemsMap(context.Background(), "test_channel")
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Errorf("expected %v, got %v", a, b)
	}
	t.Run("no repo file", func(t *testing.T) {
		r2 := newRepository(
			nil,
			Source{
				Owner:      "test_owner",
				Repository: "test_repo_2",
				Reference:  "test_ref",
				Priority:   0,
				Channels: []Channel{
					{
						Name: "test_channel",
					},
				},
			},
			"./test/repo_2",
		)
		fsMap, err := r2.GetFileSystemsMap(context.Background(), "test_channel")
		if err != nil {
			t.Error(err)
		}
		if len(fsMap) > 0 {
			t.Error("expect empty map")
		}
	})
	t.Run("error", func(t *testing.T) {
		_, err = r.GetFileSystemsMap(context.Background(), "test")
		if err == nil {
			t.Error("expected error")
		}
	})
}

func TestHandler_FileSystem(t *testing.T) {
	r := newRepository(
		nil,
		Source{
			Owner:      "test_owner",
			Repository: "test_repo",
			Reference:  "test_ref",
			Channels: []Channel{
				{
					Name:      "test_channel",
					Blacklist: []string{"test_dir"},
				},
			},
		},
		"./test/repo_1",
	)
	a := os.DirFS("test/repo_1/sha_ref/mods/test_channel/test_mod_1")
	b, err := r.GetFileSystem(context.Background(), "test_channel", "test_mod_1")
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Errorf("expected %v, got %v", a, b)
	}
	t.Run("error", func(t *testing.T) {
		t.Run("fs ref does not exist", func(t *testing.T) {
			_, err = r.GetFileSystem(context.Background(), "test_channel", "test")
			if err == nil {
				t.Error("expected error")
			}
		})
		t.Run("channel does not exist", func(t *testing.T) {
			_, err = r.GetFileSystem(context.Background(), "test", "test_mod_1")
			if err == nil {
				t.Error("expected error")
			}
		})
	})
}

func TestHandler_Refresh(t *testing.T) {
	mockClient := &gitHubClientMock{
		Commits: map[string]map[string]map[string]client.GitCommit{
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
	r := newRepository(
		mockClient,
		Source{
			Owner:      "test_owner",
			Repository: "test_repo",
			Reference:  "test_ref",
			Channels: []Channel{
				{
					Name:      "test_channel",
					Blacklist: []string{"test_dir"},
				},
			},
		},
		path.Join(tempDir, "repo"),
	)
	err := r.Refresh(context.Background())
	if err != nil {
		t.Error(err)
	}
	rf, err := readRepoFile(path.Join(tempDir, "repo"))
	if err != nil {
		t.Error(err)
	}
	if rf.GitCommit.Sha != "1234" {
		t.Errorf("expect 1234, got %s", rf.GitCommit.Sha)
	}
	if rf.Path != "1234/test" {
		t.Errorf("expect test, got %s", rf.Path)
	}
	_, err = os.Stat(path.Join(tempDir, "repo/1234/test/test_channel/test_mod/Modfile.yml"))
	if err != nil {
		t.Error(err)
	}
	t.Run("refresh existing", func(t *testing.T) {
		mockClient.Commits = map[string]map[string]map[string]client.GitCommit{
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
		err = r.Refresh(context.Background())
		if err != nil {
			t.Error(err)
		}
		rf, err = readRepoFile(path.Join(tempDir, "repo"))
		if err != nil {
			t.Error(err)
		}
		if rf.GitCommit.Sha != "5678" {
			t.Errorf("expect 5678, got %s", rf.GitCommit.Sha)
		}
		if rf.Path != "5678/test" {
			t.Errorf("expect test, got %s", rf.Path)
		}
		_, err = os.Stat(path.Join(tempDir, "repo/5678/test/test_channel/test_mod/Modfile.yml"))
		if err != nil {
			t.Error(err)
		}
		_, err = os.Stat(path.Join(tempDir, "repo/1234"))
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
	Commits  map[string]map[string]map[string]client.GitCommit
	Archives map[string]map[string]map[string]string
}

func (m *gitHubClientMock) GetLastCommit(ctx context.Context, owner, repo, ref string) (client.GitCommit, error) {
	if m.Err != nil {
		return client.GitCommit{}, m.Err
	}
	repos, ok := m.Commits[owner]
	if !ok {
		return client.GitCommit{}, client.NewResponseError(http.StatusNotFound, "commit: owner not found")
	}
	refs, ok := repos[repo]
	if !ok {
		return client.GitCommit{}, client.NewResponseError(http.StatusNotFound, "commit: repo not found")
	}
	commit, ok := refs[ref]
	if !ok {
		return client.GitCommit{}, client.NewResponseError(http.StatusNotFound, "commit: ref not found")
	}
	return commit, nil
}

func (m *gitHubClientMock) GetRepoTarGzArchive(ctx context.Context, owner, repo, ref string) (io.ReadCloser, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	repos, ok := m.Archives[owner]
	if !ok {
		return nil, client.NewResponseError(http.StatusNotFound, "archive: owner not found")
	}
	refs, ok := repos[repo]
	if !ok {
		return nil, client.NewResponseError(http.StatusNotFound, "archive: repo not found")
	}
	filePath, ok := refs[ref]
	if !ok {
		return nil, client.NewResponseError(http.StatusNotFound, "archive: ref not found")
	}
	return os.Open(filePath)
}
