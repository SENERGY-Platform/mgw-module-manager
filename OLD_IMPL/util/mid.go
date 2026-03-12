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
	"bytes"
	"github.com/google/uuid"
	"os"
)

func GetManagerID(pth, val string) (string, error) {
	if val != "" {
		return val, nil
	}
	file, err := os.OpenFile(pth, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return "", err
	}
	defer file.Close()
	buf := new(bytes.Buffer)
	n, err := buf.ReadFrom(file)
	if err != nil {
		return "", err
	}
	var id string
	if n != 0 {
		id = buf.String()
	} else {
		uid, err := uuid.NewRandom()
		if err != nil {
			return "", err
		}
		id = uid.String()
		_, err = file.Write([]byte(id))
		if err != nil {
			return "", err
		}
	}
	return id, nil
}
