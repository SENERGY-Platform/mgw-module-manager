package modfile

import (
	"errors"
	"io/fs"
	"regexp"

	"github.com/SENERGY-Platform/mgw-modfile-lib"
	module_lib "github.com/SENERGY-Platform/mgw-module-lib/model"
	helper_file_sys "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/file_sys"
)

var regExp = regexp.MustCompile(`^Modfile\.(?:yml|yaml)$`)

func GetModule(fSys fs.FS) (module_lib.Module, error) {
	mfPath, err := helper_file_sys.FindFile(fSys, regExp.MatchString)
	if err != nil {
		return module_lib.Module{}, err
	}
	if mfPath == "" {
		return module_lib.Module{}, errors.New("modfile not found")
	}
	file, err := fSys.Open(mfPath)
	if err != nil {
		return module_lib.Module{}, err
	}
	defer file.Close()
	return modfile_lib.Decode(file)
}
