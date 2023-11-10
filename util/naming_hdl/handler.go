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

package naming_hdl

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"github.com/google/uuid"
)

var Global *handler

type handler struct {
	coreID string
	prefix string
}

func Init(cID string, prefix string) {
	Global = &handler{
		coreID: cID,
		prefix: prefix,
	}
}

func (h *handler) NewContainerName(subPrefix string) (string, error) {
	newUUID, err := uuid.NewUUID()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s-%s-%s", h.prefix, h.coreID, subPrefix, hex.EncodeToString([]byte(newUUID.String()))), nil
}

func (h *handler) NewContainerAlias(arg ...string) string {
	hash := sha1.New()
	for _, s := range arg {
		hash.Write([]byte(s))
	}
	return fmt.Sprintf("%s-%s-%s", h.prefix, h.coreID, genHash(arg...))
}

func (h *handler) NewVolumeName(arg ...string) string {
	hash := sha1.New()
	for _, s := range arg {
		hash.Write([]byte(s))
	}
	return fmt.Sprintf("%s_%s_%s", h.prefix, h.coreID, genHash(arg...))
}

func genHash(str ...string) string {
	hash := sha1.New()
	for _, s := range str {
		hash.Write([]byte(s))
	}
	return hex.EncodeToString(hash.Sum(nil))
}
