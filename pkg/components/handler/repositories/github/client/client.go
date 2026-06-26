package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

const (
	acceptHeaderKey       = "Accept"
	gitHubApiVerHeaderKey = "X-GitHub-Api-Version"
	gitHubApiVer          = "2022-11-28"
	gitHubJsonMediaType   = "application/vnd.github+json"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	httpClient HTTPClient
	baseURL    string
}

func New(httpClient HTTPClient, baseUrl string) *Client {
	return &Client{
		httpClient: httpClient,
		baseURL:    baseUrl,
	}
}

func (c *Client) GetLastCommit(ctx context.Context, owner, repo, ref string) (GitCommit, error) {
	u, err := url.JoinPath(c.baseURL, "repos", owner, repo, "commits", ref)
	if err != nil {
		return GitCommit{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return GitCommit{}, err
	}
	req.Header.Set(acceptHeaderKey, gitHubJsonMediaType)
	req.Header.Set(gitHubApiVerHeaderKey, gitHubApiVer)
	res, err := c.httpClient.Do(req)
	if err != nil {
		return GitCommit{}, err
	}
	defer func() {
		if err != nil {
			_, _ = io.ReadAll(res.Body)
		}
		res.Body.Close()
	}()
	if res.StatusCode >= 400 {
		b, err := io.ReadAll(res.Body)
		if err != nil || len(b) == 0 {
			return GitCommit{}, NewResponseError(res.StatusCode, res.Status)
		}
		return GitCommit{}, NewResponseError(res.StatusCode, string(b))
	}
	var tmp commit
	if err = json.NewDecoder(res.Body).Decode(&tmp); err != nil {
		return GitCommit{}, err
	}
	lastCommit := GitCommit{
		Sha:  tmp.Sha,
		Date: tmp.Commit.Author.Date,
	}
	if tmp.Commit.Committer.Date.After(tmp.Commit.Author.Date) {
		lastCommit.Date = tmp.Commit.Committer.Date
	}
	return lastCommit, nil
}

func (c *Client) GetRepoTarGzArchive(ctx context.Context, owner, repo, ref string) (io.ReadCloser, error) {
	u, err := url.JoinPath(c.baseURL, "repos", owner, repo, "tarball", ref)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set(acceptHeaderKey, gitHubJsonMediaType)
	req.Header.Set(gitHubApiVerHeaderKey, gitHubApiVer)
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 400 {
		defer res.Body.Close()
		b, err := io.ReadAll(res.Body)
		if err != nil || len(b) == 0 {
			return nil, NewResponseError(res.StatusCode, res.Status)
		}
		return nil, NewResponseError(res.StatusCode, string(b))
	}
	return res.Body, nil
}
