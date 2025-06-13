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

package fs_util

import (
	"os"
	"path"
	"testing"
)

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()
	err := CopyFile(os.DirFS("./test"), path.Join(tmpDir, "f1_c"), "f1")
	if err != nil {
		t.Error(err)
	}
	b, err := os.ReadFile(path.Join(tmpDir, "f1_c"))
	if err != nil {
		t.Error(err)
	}
	if string(b) != "file 1" {
		t.Errorf("expected: %s, got: %s", "file 1", string(b))
	}
	t.Run("error", func(t *testing.T) {
		t.Run("invalid fs", func(t *testing.T) {
			err = CopyFile(os.DirFS("does_not_exist"), path.Join(tmpDir, "f1_c"), "f1")
			if err == nil {
				t.Error("expected error")
			}
		})
		t.Run("invalid destination", func(t *testing.T) {
			err = CopyFile(os.DirFS("./test"), path.Join("does_not_exist", "f1_c"), "f1")
			if err == nil {
				t.Error("expected error")
			}
		})
		t.Run("source does not exist", func(t *testing.T) {
			err = CopyFile(os.DirFS("./test"), path.Join(tmpDir, "f1_c"), "f3")
			if err == nil {
				t.Error("expected error")
			}
		})
	})
}

func TestCopyAll(t *testing.T) {
	tmpDir := t.TempDir()
	err := CopyAll(os.DirFS("./test"), path.Join(tmpDir, "test_c"))
	if err != nil {
		t.Error(err)
	}
	b, err := os.ReadFile(path.Join(tmpDir, "test_c", "f1"))
	if err != nil {
		t.Error(err)
	}
	if string(b) != "file 1" {
		t.Errorf("expected: %s, got: %s", "file 1", string(b))
	}
	b, err = os.ReadFile(path.Join(tmpDir, "test_c", "f2"))
	if err != nil {
		t.Error(err)
	}
	if string(b) != "file 2" {
		t.Errorf("expected: %s, got: %s", "file 2", string(b))
	}
	t.Run("error", func(t *testing.T) {
		t.Run("invalid fs", func(t *testing.T) {
			err = CopyAll(os.DirFS("does_not_exist"), path.Join(tmpDir, "test_c"))
			if err == nil {
				t.Error("expected error")
			}
		})
		t.Run("invalid destination", func(t *testing.T) {
			err = CopyAll(os.DirFS("./test"), "does_not_exist/test_c")
			if err == nil {
				t.Error("expected error")
			}
		})
	})
}

func TestFindFile(t *testing.T) {
	t.Run("exists", func(t *testing.T) {
		filePath, err := FindFile(os.DirFS("./test"), func(v string) bool {
			return v == "f1"
		})
		if err != nil {
			t.Error(err)
		}
		if filePath != "f1" {
			t.Errorf("expected: %s, got: %s", "f1", filePath)
		}
	})
	t.Run("does not exist", func(t *testing.T) {
		filePath, err := FindFile(os.DirFS("./test"), func(v string) bool {
			return v == "f3"
		})
		if err != nil {
			t.Error(err)
		}
		if filePath != "" {
			t.Error("should be empty")
		}
	})
	t.Run("error invalid fs", func(t *testing.T) {
		_, err := FindFile(os.DirFS("does_not_exist"), func(v string) bool {
			return true
		})
		if err == nil {
			t.Error("expected error")
		}
	})
}
