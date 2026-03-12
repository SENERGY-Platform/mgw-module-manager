/*
 * Copyright 2023 InfAI (CC SES)
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

package dir_fs

import (
	"errors"
	"io"
	"io/fs"
	"os"
)

type DirFS string

func New(path string) (DirFS, error) {
	_, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	return DirFS(path), nil
}

func (d DirFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrInvalid}
	}
	f, err := os.Open(string(d) + "/" + name)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (d DirFS) Stat(name string) (fs.FileInfo, error) {
	if !fs.ValidPath(name) {
		return nil, &os.PathError{Op: "stat", Path: name, Err: os.ErrInvalid}
	}
	f, err := os.Stat(string(d) + "/" + name)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func (d DirFS) Sub(name string) (DirFS, error) {
	if !fs.ValidPath(name) {
		return "", &os.PathError{Op: "sub", Path: name, Err: os.ErrInvalid}
	}
	if name == "." {
		return d, nil
	}
	return New(string(d) + "/" + name)
}

func (d DirFS) Path() string {
	return string(d)
}

func Copy(src DirFS, dst string) error {
	srcStat, err := src.Stat(".")
	if err != nil {
		return err
	}
	dirEntries, err := fs.ReadDir(src, ".")
	if err != nil {
		return err
	}
	err = os.MkdirAll(dst, srcStat.Mode())
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			os.RemoveAll(dst)
		}
	}()
	for _, entry := range dirEntries {
		var entryInfo fs.FileInfo
		entryInfo, err = entry.Info()
		if err != nil {
			return err
		}
		dstEntryPath := dst + "/" + entryInfo.Name()
		if entryInfo.IsDir() {
			err = os.MkdirAll(dstEntryPath, entryInfo.Mode())
			if err != nil {
				return err
			}
			var srcSubDir DirFS
			srcSubDir, err = src.Sub(entryInfo.Name())
			if err != nil {
				return err
			}
			err = Copy(srcSubDir, dstEntryPath)
			if err != nil {
				return err
			}
		} else if entryInfo.Mode().IsRegular() {
			var i int64
			i, err = copyFile(entryInfo.Name(), src, dstEntryPath)
			if err != nil {
				return err
			}
			if i != entryInfo.Size() {
				return errors.New("error writing to file")
			}
		}
	}
	return nil
}

func copyFile(name string, src DirFS, dst string) (int64, error) {
	srcFile, err := src.Open(name)
	if err != nil {
		return 0, err
	}
	defer srcFile.Close()
	dstFile, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer dstFile.Close()
	return io.Copy(dstFile, srcFile)
}
