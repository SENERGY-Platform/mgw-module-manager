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

package clients

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/SENERGY-Platform/mgw-module-manager/lib/errors"
)

type ErrHttpResponse struct {
	err        error
	statusCode int
	header     http.Header
	body       []byte
}

func (e *ErrHttpResponse) Error() string {
	if e.err != nil {
		return fmt.Sprintf("%s | read body: %s", http.StatusText(e.statusCode), e.err.Error())
	}
	if len(e.body) == 0 {
		return http.StatusText(e.statusCode)
	}
	return http.StatusText(e.statusCode) + " | " + string(e.body)
}

func (e *ErrHttpResponse) Unwrap() error {
	return e.err
}

func (e *ErrHttpResponse) StatusCode() int {
	return e.statusCode
}

// HeaderValue gets the first value associated with the given key. If
// there are no values associated with the key, HeaderValue returns "".
func (e *ErrHttpResponse) HeaderValue(key string) string {
	return e.header.Get(key)
}

// HeaderValues returns all values associated with the given key.
func (e *ErrHttpResponse) HeaderValues(key string) []string {
	return e.header.Values(key)
}

func (e *ErrHttpResponse) Body() []byte {
	return bytes.Clone(e.body)
}

func wrapError(err error, errCode string) error {
	switch errCode {
	case "001":
		err = errors.Wrap[errors.ErrNotFound](err)
	case "002":
		err = errors.Wrap[errors.ErrExists](err)
	case "003":
		err = errors.Wrap[errors.ErrInvalidInput](err)
	case "004":
		err = errors.Wrap[errors.ErrActiveJob](err)
	}
	return err
}
