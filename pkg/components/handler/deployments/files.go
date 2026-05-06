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
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
)

func getDefaultFiles(moduleFiles map[string]pkg_models.ModuleLibFile, moduleFileSystem fs.FS) (map[string][]byte, error) {
	files := make(map[string][]byte)
	var errs []string
	for reference, file := range moduleFiles {
		if file.Source != "" {
			b, err := fileToBytes(moduleFileSystem, file.Source)
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
	moduleFiles map[string]pkg_models.ModuleLibFile,
	defaultDataFiles map[string][]byte,
	userInputsFiles map[string][]byte,
	deploymentId string,
) map[string]pkg_models.DeploymentFile {
	files := make(map[string]pkg_models.DeploymentFile)
	for reference := range moduleFiles {
		data, ok := userInputsFiles[reference]
		if !ok || len(data) == 0 {
			continue
		}
		defaultData, ok := defaultDataFiles[reference]
		if ok && bytes.Equal(data, defaultData) {
			continue
		}
		files[reference] = pkg_models.DeploymentFile{
			DeploymentId: deploymentId,
			Reference:    reference,
			Data:         data,
		}
	}
	return files
}

func mergeFiles(
	defaultDataFiles map[string][]byte,
	userDataFiles map[string]pkg_models.DeploymentFile,
) map[string][]byte {
	files := make(map[string][]byte)
	maps.Copy(files, defaultDataFiles)
	for reference, file := range userDataFiles {
		files[reference] = file.Data
	}
	return files
}

func checkFiles(
	moduleFiles map[string]pkg_models.ModuleLibFile,
	files map[string][]byte,
) error {
	var errs []string
	for reference, moduleFile := range moduleFiles {
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
	moduleFileGroups map[string]struct{},
	userInputFileGroups map[string]map[string]pkg_models.DeploymentFileGroupUserInput,
	deploymentId string,
) map[string]pkg_models.DeploymentFileGroup {
	fileGroups := make(map[string]pkg_models.DeploymentFileGroup)
	for reference := range moduleFileGroups {
		fg, ok := userInputFileGroups[reference]
		if !ok {
			continue
		}
		id := helper_naming.GenHash(deploymentId, reference)
		var files []pkg_models.DeploymentFileGroupFile
		for pth, input := range fg {
			files = append(files, pkg_models.DeploymentFileGroupFile{
				Path:   pth,
				Format: input.Format,
				Data:   input.Data,
			})
		}
		fileGroups[reference] = pkg_models.DeploymentFileGroup{
			Id:           id,
			DeploymentId: deploymentId,
			Reference:    reference,
			Files:        files,
		}
	}
	return fileGroups
}

func createFilesDir(workDirPath, deploymentFilesDirName string) error {
	return os.Mkdir(path.Join(workDirPath, deploymentFilesDirName), dirPerm)
}

func removeFilesDir(workDirPath, deploymentFilesDirName string) error {
	return os.RemoveAll(path.Join(workDirPath, deploymentFilesDirName))
}

func createFileGroups(
	deploymentFilesDirName string,
	userDataFileGroups map[string]pkg_models.DeploymentFileGroup,
	workDirPath string,
) (map[string][]fileGroupMount, error) {
	fileNames := make(map[string][]fileGroupMount)
	for reference, fileGroup := range userDataFileGroups {
		for _, file := range fileGroup.Files {
			fileName := helper_naming.GenHash(fileGroup.Id, file.Path)
			err := writeToFile(file.Data, path.Join(workDirPath, deploymentFilesDirName, fileName))
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

func createFiles(
	deploymentId string,
	deploymentFilesDirName string,
	files map[string][]byte,
	workDirPath string,
) (map[string]string, error) {
	mounts := make(map[string]string)
	for reference, data := range files {
		fileName := helper_naming.GenHash(deploymentId, reference)
		err := writeToFile(data, path.Join(workDirPath, deploymentFilesDirName, fileName))
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
