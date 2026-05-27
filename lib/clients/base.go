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
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/SENERGY-Platform/mgw-module-manager/lib/constants"
)

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func doJson(client httpClient, req *http.Request, v any) error {
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	err = handleResponseErr(res)
	if err != nil {
		return err
	}
	err = json.NewDecoder(res.Body).Decode(v)
	if err != nil {
		_, _ = io.ReadAll(res.Body)
		return err
	}
	return nil
}

func doErr(client httpClient, req *http.Request) error {
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	err = handleResponseErr(res)
	if err != nil {
		return err
	}
	_, _ = io.ReadAll(res.Body)
	return nil
}

func handleResponseErr(resp *http.Response) error {
	if resp.StatusCode >= 400 {
		resErr := &ErrHttpResponse{
			statusCode: resp.StatusCode,
			header:     resp.Header.Clone(),
		}
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			resErr.err = err
		} else {
			resErr.body = b
		}
		return wrapError(resErr, resp.Header.Get(constants.HttpHeaderErrorCode))
	}
	return nil
}

func queryJoinStrings(sl []string, sep string) string {
	tmp := make([]string, len(sl))
	for i, s := range sl {
		tmp[i] = url.QueryEscape(s)
	}
	return strings.Join(tmp, sep)
}

var urlPathParamRegex = regexp.MustCompile(":[^/]+")

func getUrlRelPath(template string, params ...string) string {
	placeholders := urlPathParamRegex.FindAllString(template, -1)
	if len(placeholders) > len(params) {
		placeholders = placeholders[:len(params)]
	}
	for i, placeholder := range placeholders {
		template = strings.Replace(template, placeholder, url.PathEscape(params[i]), 1)
	}
	return template
}
