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

package archive

import (
	"os"
	"path"
	"testing"
)

func TestExtractTarGz(t *testing.T) {
	file, err := os.Open("./test/test.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	tempDir := t.TempDir()
	rootDir, err := ExtractTarGz(file, tempDir)
	if err != nil {
		t.Error()
	}
	if rootDir != "test" {
		t.Errorf("expected %s got %s", "test", rootDir)
	}
	_, err = os.Stat(path.Join(tempDir, rootDir, "test_mod/Modfile.yml"))
	if err != nil {
		t.Error(err)
	}
	t.Run("error", func(t *testing.T) {
		invalidFile, err := os.Open("archive_test.go")
		if err != nil {
			t.Fatal(err)
		}
		defer invalidFile.Close()
		_, err = ExtractTarGz(invalidFile, tempDir)
		if err == nil {
			t.Error("expected error")
		}
	})
}
