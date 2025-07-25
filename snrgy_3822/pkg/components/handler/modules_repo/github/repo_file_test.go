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
	"encoding/json"
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/components/handler/modules_repo/github/client"
	"os"
	"path"
	"reflect"
	"testing"
)

func Test_readRepoFile(t *testing.T) {
	a := repoFile{
		GitCommit: client.GitCommit{
			Sha: "test_sha",
		},
		Path: "test_source",
	}
	tempDir := t.TempDir()
	err := createTestFile(a, path.Join(tempDir, repoFileName))
	if err != nil {
		t.Fatal(err)
	}
	b, err := readRepoFile(tempDir)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(b, a) {
		t.Errorf("excpected %v, got %v", a, b)
	}
	t.Run("does not exist", func(t *testing.T) {
		b, err = readRepoFile("test")
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(b, repoFile{}) {
			t.Errorf("excpected %v, got %v", a, b)
		}
	})
}

func Test_writeRepoFile(t *testing.T) {
	a := repoFile{
		GitCommit: client.GitCommit{
			Sha: "test_sha",
		},
		Path: "test_source",
	}
	tempDir := t.TempDir()
	err := writeRepoFile(tempDir, a)
	if err != nil {
		t.Error(err)
	}
	b, err := readTestFile(path.Join(tempDir, repoFileName))
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(b, a) {
		t.Errorf("excpected %v, got %v", a, b)
	}
	t.Run("file exists", func(t *testing.T) {
		aNew := a
		aNew.GitCommit.Sha = "test_sha2"
		err = writeRepoFile(tempDir, aNew)
		if err != nil {
			t.Error(err)
		}
		bNew, err := readTestFile(path.Join(tempDir, repoFileName))
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(bNew, aNew) {
			t.Errorf("excpected %v, got %v", aNew, bNew)
		}
		b, err = readTestFile(path.Join(tempDir, bkRepoFileName))
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(b, a) {
			t.Errorf("excpected %v, got %v", a, b)
		}
	})
}

func createTestFile(a repoFile, p string) error {
	file, err := os.Create(p)
	if err != nil {
		return err
	}
	defer file.Close()
	err = json.NewEncoder(file).Encode(a)
	if err != nil {
		return err
	}
	return nil
}

func readTestFile(p string) (repoFile, error) {
	file, err := os.Open(p)
	if err != nil {
		return repoFile{}, err
	}
	defer file.Close()
	var a repoFile
	err = json.NewDecoder(file).Decode(&a)
	if err != nil {
		return repoFile{}, err
	}
	return a, nil
}
