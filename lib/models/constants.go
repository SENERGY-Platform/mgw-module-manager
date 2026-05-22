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

package models

const (
	ActionInstall = "install"
	ActionChange  = "change"
	ActionRemove  = "remove"
)

type DeploymentState = int

const (
	DeploymentHealthy DeploymentState = iota + 1
	DeploymentUnhealthy
)

const (
	HttpHeaderCoreId    = "X-Core-Id"
	HttpHeaderManagerId = "X-Manager-Id"
	HttpHeaderRuntimeId = "X-Runtime-Id"
	HttpHeaderRequestId = "X-Request-Id"
	HttpHeaderErrorCode = "X-Err-Code"
	HttpHeaderApiVer    = "X-Version"
	HttpHeaderSrvName   = "X-Service"
)
