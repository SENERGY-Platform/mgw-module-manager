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

package service

type RepoModule struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	Desc         string         `json:"description"`
	Version      string         `json:"version"`
	Repositories []Repository   `json:"repositories"`
	Installed    *ModuleVariant `json:"installed"`
}

type Repository struct {
	Source   string    `json:"source"`
	Default  bool      `json:"default"`
	Channels []Channel `json:"channels"`
}

type Channel struct {
	Name    string `json:"name"`
	Default bool   `json:"default"`
	Version string `json:"version"`
}

type ModuleVariant struct {
	Source  string `json:"source"`
	Channel string `json:"channel"`
	Version string `json:"version"`
}
