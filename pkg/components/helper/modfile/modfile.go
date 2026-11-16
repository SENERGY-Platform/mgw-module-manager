package modfile

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"regexp"

	"github.com/SENERGY-Platform/mgw-modfile-lib"
	helper_errors "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/errors"
	helper_file_sys "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/file_sys"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
	external_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/external"
)

var regExp = regexp.MustCompile(`^Modfile\.(?:yml|yaml)$`)

func GetModule(fSys fs.FS) (external_models.ModuleLibModule, error) {
	mfPath, err := helper_file_sys.FindFile(fSys, regExp.MatchString)
	if err != nil {
		return external_models.ModuleLibModule{}, err
	}
	if mfPath == "" {
		return external_models.ModuleLibModule{}, errors.New("modfile not found")
	}
	file, err := fSys.Open(mfPath)
	if err != nil {
		return external_models.ModuleLibModule{}, err
	}
	defer file.Close()
	return modfile_lib.Decode(file)
}

// GetFiles reads the default data of the module files that declare a source from the module file system.
func GetFiles(modFS fs.FS, modFiles map[string]external_models.ModuleLibFile) (map[string]pkg_models.ModuleFile, error) {
	if len(modFiles) == 0 {
		return nil, nil
	}
	files := make(map[string]pkg_models.ModuleFile)
	var errs []error
	for reference, modLibFile := range modFiles {
		file := pkg_models.ModuleFile{
			ModuleLibFile: modLibFile,
		}
		if modLibFile.Source != "" {
			b, err := fileToBytes(modFS, modLibFile.Source)
			if err != nil {
				errs = append(errs, fmt.Errorf("'%s' %w", reference, err))
				continue
			}
			file.DefaultData = b
		}
		files[reference] = file
	}
	if len(errs) > 0 {
		return nil, helper_errors.Join(errs...)
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
