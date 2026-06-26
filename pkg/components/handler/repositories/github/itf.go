package github

import (
	"context"
	"io"

	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/repositories/github/client"
)

type gitHubClient interface {
	GetLastCommit(ctx context.Context, owner, repo, ref string) (client.GitCommit, error)
	GetRepoTarGzArchive(ctx context.Context, owner, repo, ref string) (io.ReadCloser, error)
}
