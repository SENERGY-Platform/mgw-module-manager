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

package schema

import _ "embed"

//go:embed modules.sql
var modules []byte

//go:embed deployments.sql
var deployments []byte

//go:embed aux_deployments.sql
var auxDeployments []byte

//go:embed dep_advertisements.sql
var depAdvertisements []byte

var Init = migration{
	modules,
	deployments,
	auxDeployments,
	depAdvertisements,
}
