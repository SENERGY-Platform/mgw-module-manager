package modfile

import (
	"errors"
	"io/fs"
	"regexp"

	"github.com/SENERGY-Platform/mgw-modfile-lib"
	helper_file_sys "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/file_sys"
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
)

var regExp = regexp.MustCompile(`^Modfile\.(?:yml|yaml)$`)

func GetModule(fSys fs.FS) (pkg_models.ModuleLibModule, error) {
	mfPath, err := helper_file_sys.FindFile(fSys, regExp.MatchString)
	if err != nil {
		return pkg_models.ModuleLibModule{}, err
	}
	if mfPath == "" {
		return pkg_models.ModuleLibModule{}, errors.New("modfile not found")
	}
	file, err := fSys.Open(mfPath)
	if err != nil {
		return pkg_models.ModuleLibModule{}, err
	}
	defer file.Close()
	return modfile_lib.Decode(file)
}
