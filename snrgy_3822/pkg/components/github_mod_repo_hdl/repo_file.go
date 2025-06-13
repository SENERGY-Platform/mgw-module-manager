package github_mod_repo_hdl

import (
	"encoding/json"
	"errors"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/github_mod_repo_hdl/github_clt"
	"io"
	"os"
	"path"
)

const (
	repoFileName   = "repo"
	bkRepoFileName = "repo.bk"
)

type repoFile struct {
	GitCommit github_clt.GitCommit `json:"git_commit"`
	Path      string               `json:"path"`
}

func readRepoFile(p string) (repoFile, error) {
	file, err := os.Open(path.Join(p, repoFileName))
	if err != nil {
		if os.IsNotExist(err) {
			return repoFile{}, nil
		}
		return repoFile{}, err
	}
	defer file.Close()
	var repo repoFile
	if err = json.NewDecoder(file).Decode(&repo); err != nil {
		return repoFile{}, err
	}
	return repo, nil
}

func writeRepoFile(pth string, mr repoFile) error {
	rfPath := path.Join(pth, repoFileName)
	var rfBkPath string
	_, err := os.Stat(rfPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else {
		rfBkPath = path.Join(pth, bkRepoFileName)
		if err = copyFile(rfPath, rfBkPath); err != nil {
			return err
		}
	}
	file, err := os.OpenFile(rfPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	defer func() {
		file.Close()
		if err != nil && rfBkPath != "" {
			if e := copyFile(rfBkPath, rfPath); e != nil {
				err = errors.Join(err, e)
			}
		}
	}()
	if err = json.NewEncoder(file).Encode(mr); err != nil {
		return err
	}
	return nil
}

func copyFile(srcPath, targetPath string) error {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	targetFile, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer targetFile.Close()
	_, err = io.Copy(targetFile, srcFile)
	if err != nil {
		return err
	}
	return nil
}
