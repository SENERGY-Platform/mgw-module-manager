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

package service

import (
	"context"
	"testing"
	"time"

	lib_models "github.com/SENERGY-Platform/mgw-module-manager/lib/models"
	handler_jobs "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/jobs"
)

type refreshTestRepositoriesHandler struct {
	repositoriesHandler
	calls chan lib_models.RepositoriesRefreshFilter
}

func (m *refreshTestRepositoriesHandler) RefreshRepositories(_ context.Context, filter lib_models.RepositoriesRefreshFilter) ([]lib_models.RepositoryResult, error) {
	m.calls <- filter
	return nil, nil
}

func newRefreshTestService() (*Service, *refreshTestRepositoriesHandler) {
	repoHdl := &refreshTestRepositoriesHandler{calls: make(chan lib_models.RepositoriesRefreshFilter, 4)}
	srv := New(repoHdl, nil, nil, nil, nil, nil, nil, handler_jobs.New(context.Background(), handler_jobs.Config{}), nil)
	return srv, repoHdl
}

func TestService_refreshRepositoriesIfIdle(t *testing.T) {
	t.Run("idle starts a refresh of all repositories", func(t *testing.T) {
		srv, repoHdl := newRefreshTestService()
		job, skipped, err := srv.refreshRepositoriesIfIdle(context.Background())
		if err != nil || skipped || job.Id == "" {
			t.Fatalf("expected started job, got job=%+v skipped=%v err=%v", job, skipped, err)
		}
		select {
		case filter := <-repoHdl.calls:
			if len(filter.Sources) != 0 {
				t.Errorf("expected empty filter, got %+v", filter)
			}
		case <-time.After(time.Second):
			t.Fatal("repositories handler was not called")
		}
	})

	t.Run("pending change request skips and is kept", func(t *testing.T) {
		srv, repoHdl := newRefreshTestService()
		cr := &modulesChangeRequest{Remove: []string{"mod"}}
		srv.changeRequest = cr
		_, skipped, err := srv.refreshRepositoriesIfIdle(context.Background())
		if err != nil || !skipped {
			t.Fatalf("expected skip without error, got skipped=%v err=%v", skipped, err)
		}
		if srv.changeRequest != cr {
			t.Error("change request was discarded")
		}
		if len(repoHdl.calls) != 0 {
			t.Error("repositories handler must not be called")
		}
	})

	t.Run("active repository job skips", func(t *testing.T) {
		srv, repoHdl := newRefreshTestService()
		job, err := srv.jobsHandler.CreateSlotJob(repositoryJobSlotNum, "test")
		if err != nil {
			t.Fatal(err)
		}
		defer job.Done()
		_, skipped, err := srv.refreshRepositoriesIfIdle(context.Background())
		if err != nil || !skipped {
			t.Fatalf("expected skip without error, got skipped=%v err=%v", skipped, err)
		}
		if len(repoHdl.calls) != 0 {
			t.Error("repositories handler must not be called")
		}
	})
}

func TestService_RefreshRepositories_DiscardsChangeRequest(t *testing.T) {
	srv, _ := newRefreshTestService()
	srv.changeRequest = &modulesChangeRequest{Remove: []string{"mod"}}
	if _, err := srv.RefreshRepositories(context.Background(), lib_models.RepositoriesRefreshFilter{}); err != nil {
		t.Fatal(err)
	}
	if srv.changeRequest != nil {
		t.Error("public refresh must keep discarding the change request")
	}
}

func TestService_RefreshRepositoriesLoop(t *testing.T) {
	srv, repoHdl := newRefreshTestService()
	ctx, cf := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		srv.RefreshRepositoriesLoop(ctx, time.Millisecond, time.Hour, time.Hour)
		close(done)
	}()
	select {
	case <-repoHdl.calls:
	case <-time.After(time.Second):
		t.Fatal("loop did not refresh after startup delay")
	}
	cf()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("loop did not stop on context cancel")
	}
}

func TestService_RefreshRepositoriesLoop_RetriesSkippedRun(t *testing.T) {
	srv, repoHdl := newRefreshTestService()
	srv.changeRequest = &modulesChangeRequest{Remove: []string{"mod"}}
	ctx, cf := context.WithCancel(context.Background())
	defer cf()
	go srv.RefreshRepositoriesLoop(ctx, time.Millisecond, time.Hour, 20*time.Millisecond)
	time.Sleep(10 * time.Millisecond)
	srv.mu.Lock()
	srv.changeRequest = nil
	srv.mu.Unlock()
	select {
	case <-repoHdl.calls:
	case <-time.After(time.Second):
		t.Fatal("skipped run was not retried before the interval")
	}
}
