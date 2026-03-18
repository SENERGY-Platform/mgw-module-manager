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
	"maps"
	"os"
	"path"
	"strings"

	helper_naming "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/naming"
	models_handler_deployment "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/deployment"
	models_handler_module "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/module"
	models_handler_storage "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/storage"
)

func getDefaultFiles(module models_handler_module.Module) (map[string][]byte, error) {
	files := make(map[string][]byte)
	var errs []string
	for reference, file := range module.Files {
		if file.Source != "" {
			b, err := fileToBytes(module.FileSystem, file.Source)
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
	module models_handler_module.Module,
	defaultData defaultDataCollection,
	userInputs models_handler_deployment.UserInput,
	deploymentId string,
) map[string]models_handler_storage.DeploymentFile {
	files := make(map[string]models_handler_storage.DeploymentFile)
	for reference := range module.Files {
		data, ok := userInputs.Files[reference]
		if !ok || len(data) == 0 {
			continue
		}
		defaultData, ok := defaultData.Files[reference]
		if ok && bytes.Equal(data, defaultData) {
			continue
		}
		files[reference] = models_handler_storage.DeploymentFile{
			DeploymentId: deploymentId,
			Reference:    reference,
			Data:         data,
		}
	}
	return files
}

func mergeFiles(
	defaultData defaultDataCollection,
	userData userDataCollection,
) map[string][]byte {
	files := make(map[string][]byte)
	maps.Copy(files, defaultData.Files)
	for reference, file := range userData.Files {
		files[reference] = file.Data
	}
	return files
}

func checkFiles(
	module models_handler_module.Module,
	files map[string][]byte,
) error {
	var errs []string
	for reference, moduleFile := range module.Files {
		_, ok := files[reference]
		if !ok && moduleFile.Required {
			errs = append(errs, fmt.Sprintf("file %s required", reference))
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "\n"))
	}
	return nil
}

func getProvidedFileGroups(
	module models_handler_module.Module,
	userInputs models_handler_deployment.UserInput,
	deploymentId string,
) map[string]models_handler_storage.DeploymentFileGroup {
	fileGroups := make(map[string]models_handler_storage.DeploymentFileGroup)
	for reference := range module.FileGroups {
		fg, ok := userInputs.FileGroups[reference]
		if !ok {
			continue
		}
		id := helper_naming.GenHash(deploymentId, reference)
		var files []models_handler_storage.DeploymentFileGroupFile
		for path, input := range fg {
			files = append(files, models_handler_storage.DeploymentFileGroupFile{
				Path:   path,
				Format: input.Format,
				Data:   input.Data,
			})
		}
		fileGroups[reference] = models_handler_storage.DeploymentFileGroup{
			Id:           id,
			DeploymentId: deploymentId,
			Reference:    reference,
			Files:        files,
		}
	}
	return fileGroups
}

func (h *Handler) createFilesDir(deployment extendedDeployment) error {
	return os.Mkdir(path.Join(h.config.WorkDirPath, deployment.FilesDirName), dirPerm)
}

func (h *Handler) removeFilesDir(deployment extendedDeployment) error {
	return os.RemoveAll(path.Join(h.config.WorkDirPath, deployment.FilesDirName))
}

func (h *Handler) createFileGroups(deployment extendedDeployment, userData userDataCollection) (map[string][]fileGroupMount, error) {
	fileNames := make(map[string][]fileGroupMount)
	for reference, fileGroup := range userData.FileGroups {
		for _, file := range fileGroup.Files {
			fileName := helper_naming.GenHash(fileGroup.Id, file.Path)
			err := writeToFile(file.Data, path.Join(h.config.WorkDirPath, deployment.FilesDirName, fileName))
			if err != nil {
				return nil, err
			}
			fileNames[reference] = append(fileNames[reference], fileGroupMount{
				FileName: fileName,
				Path:     file.Path,
			})
		}
	}
	return fileNames, nil
}

func (h *Handler) createFiles(deployment extendedDeployment, files map[string][]byte) (map[string]string, error) {
	mounts := make(map[string]string)
	for reference, data := range files {
		fileName := helper_naming.GenHash(deployment.Id, reference)
		err := writeToFile(data, path.Join(h.config.WorkDirPath, deployment.FilesDirName, fileName))
		if err != nil {
			return nil, err
		}
		mounts[reference] = fileName
	}
	return mounts, nil
}

func writeToFile(data []byte, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.Write(data)
	if err != nil {
		return err
	}
	return nil
}
