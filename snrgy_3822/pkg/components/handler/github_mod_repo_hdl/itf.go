package github_mod_repo_hdl

import (
	"context"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/github_mod_repo_hdl/github_clt"
	"io"
)

type GitHubClient interface {
	GetLastCommit(ctx context.Context, owner, repo, ref string) (github_clt.GitCommit, error)
	GetRepoTarGzArchive(ctx context.Context, owner, repo, ref string) (io.ReadCloser, error)
}
