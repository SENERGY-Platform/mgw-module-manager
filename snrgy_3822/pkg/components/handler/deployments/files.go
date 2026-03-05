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

package deployments

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"

	models_external "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
	models_handler_deployment "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/deployment"
	models_handler_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/storage"
)

func getDefaultFiles(moduleFiles map[string]models_external.ModuleFile, moduleFS fs.FS) (map[string][]byte, error) {
	files := make(map[string][]byte)
	var errs []string
	for reference, file := range moduleFiles {
		if file.Source != "" {
			b, err := fileToBytes(moduleFS, file.Source)
			if err != nil {
				errs = append(errs, err.Error())
				continue
			}
			files[reference] = b
		}
	}
	if len(errs) > 0 {
		return nil, errors.New(strings.Join(errs, "\n")) // TODO
	}
	return files, nil
}

func fileToBytes(fSys fs.FS, path string) ([]byte, error) {
	f, err := fSys.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

func getProvidedFiles(
	moduleFiles map[string]models_external.ModuleFile,
	defaultFiles map[string][]byte, userInputs map[string][]byte,
	deploymentId string,
) (map[string]models_handler_storage.DeploymentFile, error) {
	files := make(map[string]models_handler_storage.DeploymentFile)
	var errs []string
	for reference, file := range moduleFiles {
		defaultData, defaultOK := defaultFiles[reference]
		data, ok := userInputs[reference]
		if !ok {
			if file.Required && !defaultOK {
				errs = append(errs, fmt.Sprintf("missing required file '%s'", reference))
			}
			continue
		}
		if defaultOK && bytes.Equal(data, defaultData) {
			continue
		}
		files[reference] = models_handler_storage.DeploymentFile{
			DeploymentId: deploymentId,
			Reference:    reference,
			Data:         data,
		}
	}
	if len(errs) > 0 {
		return nil, errors.New(strings.Join(errs, "\n")) // TODO
	}
	return files, nil
}

func getProvidedFileGroups(moduleFileGroups map[string]struct{}, userInputs map[string]map[string]models_handler_deployment.FileGroupUserInput, deploymentId string) map[string]models_handler_storage.DeploymentFileGroup {
	fileGroups := make(map[string]models_handler_storage.DeploymentFileGroup)
	for reference := range moduleFileGroups {
		fg, ok := userInputs[reference]
		if !ok {
			continue
		}
		var files []models_handler_storage.DeploymentFileGroupFile
		for path, input := range fg {
			files = append(files, models_handler_storage.DeploymentFileGroupFile{
				Path:   path,
				Format: input.Format,
				Data:   input.Data,
			})
		}
		fileGroups[reference] = models_handler_storage.DeploymentFileGroup{
			Id:           deploymentId + "_" + reference,
			DeploymentId: deploymentId,
			Reference:    reference,
			Files:        files,
		}
	}
	return fileGroups
}
