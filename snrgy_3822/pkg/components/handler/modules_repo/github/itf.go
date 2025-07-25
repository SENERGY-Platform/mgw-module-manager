package github

import (
	"context"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/modules_repo/github/client"
	"io"
)

type GitHubClient interface {
	GetLastCommit(ctx context.Context, owner, repo, ref string) (client.GitCommit, error)
	GetRepoTarGzArchive(ctx context.Context, owner, repo, ref string) (io.ReadCloser, error)
}
