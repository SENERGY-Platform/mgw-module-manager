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

package util

import (
	"io/fs"
	"os"
	"path"
)

type DirFS string

func NewDirFS(path string) (DirFS, error) {
	d := DirFS(path)
	_, err := fs.Stat(d, ".")
	if err != nil {
		return "", err
	}
	return d, nil
}

func (d DirFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrInvalid}
	}
	f, err := os.Open(path.Join(string(d), name))
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
	return NewDirFS(path.Join(string(d), name))
}

func (d DirFS) Path() string {
	return string(d)
}
