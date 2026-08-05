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

package test_client

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	lib_clients "github.com/SENERGY-Platform/mgw-module-manager/lib/clients"
)

const clientConfFile = "config.json"

var client *Client
var depAdvClient *lib_clients.ClientDeploymentAdvertisements
var auxDepClient *lib_clients.ClientAuxiliaryDeployments

func init() {
	conf, err := getClientConf()
	if err != nil {
		panic(err)
	}
	client = &Client{httpClient: http.DefaultClient, baseUrl: conf.BaseUrl, cookie: conf.Cookie}
	depAdvClient = lib_clients.NewClientDeploymentAdvertisements(client, conf.BaseUrl)
	auxDepClient = lib_clients.NewClientAuxiliaryDeployments(client, conf.BaseUrl)
}

type clientConf struct {
	BaseUrl string `json:"base_url"`
	Cookie  string `json:"cookie"`
}

func getClientConf() (clientConf, error) {
	c := clientConf{
		BaseUrl: "http://localhost:8080/core/api/module-manager",
	}
	file, err := os.Open(clientConfFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return clientConf{}, err
		}
		f, err := os.Create(clientConfFile)
		if err != nil {
			return clientConf{}, err
		}
		defer f.Close()
		je := json.NewEncoder(f)
		je.SetIndent("", "\t")
		err = je.Encode(c)
		if err != nil {
			return clientConf{}, err
		}
		return c, nil
	}
	defer file.Close()
	err = json.NewDecoder(file).Decode(&c)
	if err != nil {
		panic(err)
	}
	return c, nil
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	httpClient
	baseUrl string
	cookie  string
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if c.cookie != "" {
		req.Header.Set("Cookie", c.cookie)
	}
	return c.httpClient.Do(req)
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
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return errors.New(string(b))
	}
	return nil
}

func queryJoinStrings(sl []string) string {
	tmp := make([]string, len(sl))
	for i, s := range sl {
		tmp[i] = url.QueryEscape(s)
	}
	return strings.Join(tmp, ",")
}

var urlPathParamRegex = regexp.MustCompile(":[^/]+")

func getUrlRelPath(template string, params ...string) string {
	placeholders := urlPathParamRegex.FindAllString(template, -1)
	if len(placeholders) == 0 {
		return template
	}
	placeholders = placeholders[:len(params)]
	for i, placeholder := range placeholders {
		template = strings.Replace(template, placeholder, url.PathEscape(params[i]), 1)
	}
	return template
}

func writeToJson(n string, v any) {
	entries, err := os.ReadDir("outputs")
	if err != nil {
		if !os.IsNotExist(err) {
			panic(err)
		}
		err = os.Mkdir("outputs", 0755)
		if err != nil {
			panic(err)
		}
	}
	var c int64
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		parts := strings.SplitN(entry.Name(), "_", 2)
		if len(parts) != 2 {
			continue
		}
		i, err := strconv.ParseInt(strings.TrimPrefix(strings.TrimPrefix(parts[0], "0"), "0"), 10, 64)
		if err != nil {
			panic(err)
		}
		if i > c {
			c = i
		}
	}
	f, err := os.Create(path.Join("outputs", strings.Replace(n, "Test", fmt.Sprintf("%03d_", c+1), 1)+".json"))
	if err != nil {
		panic(err)
	}
	e := json.NewEncoder(f)
	e.SetIndent("", "\t")
	err = e.Encode(v)
	if err != nil {
		panic(err)
	}
}
