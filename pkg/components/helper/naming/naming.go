/*
 * Copyright 2026 InfAI (CC SES)
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

package naming

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"slices"
	"strconv"
	"time"

	"github.com/google/uuid"
)

const prefix = "mgw"

var CoreId string
var ManagerId string
var ModuleContainerNetwork string

func NewContainerName(prefix string) (string, error) {
	newUUID, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s-%s-%s", prefix, CoreId, prefix, GenHash(newUUID.String())), nil
}

func NewContainerAlias(arg ...string) string {
	return fmt.Sprintf("%s-%s-%s", prefix, CoreId, GenHash(arg...))
}

func NewVolumeName(prefix string, arg ...string) string {
	return fmt.Sprintf("%s_%s_%s_%s", prefix, CoreId, prefix, GenHash(arg...))
}

func GenHash(str ...string) string {
	hash := sha1.New()
	for _, s := range str {
		hash.Write([]byte(s))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func SetManagerID(pth, val string) error {
	if val != "" {
		ManagerId = val
		return nil
	}
	file, err := os.OpenFile(pth, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	buf := new(bytes.Buffer)
	n, err := buf.ReadFrom(file)
	if err != nil {
		return err
	}
	var id string
	if n != 0 {
		id = buf.String()
	} else {
		newUUID, err := uuid.NewV7()
		if err != nil {
			return err
		}
		id = newUUID.String()
		_, err = file.Write([]byte(id))
		if err != nil {
			return err
		}
	}
	ManagerId = id
	return nil
}

func GetRuntimeId() string {
	time.Sleep(time.Millisecond * 2)
	b := []byte(strconv.FormatInt(time.Now().UnixMilli(), 10))
	slices.Reverse(b)
	return base64.RawStdEncoding.EncodeToString(b)
}
