package modfile

import (
	"errors"
	"github.com/SENERGY-Platform/mgw-modfile-lib/modfile"
	"github.com/SENERGY-Platform/mgw-modfile-lib/v1/v1dec"
	"github.com/SENERGY-Platform/mgw-modfile-lib/v1/v1gen"
	module_lib "github.com/SENERGY-Platform/mgw-module-lib/model"
	helper_file_sys "github.com/SENERGY-Platform/mgw-module-manager/pkg/components/helper/file_sys"
	"gopkg.in/yaml.v3"
	"io"
	"io/fs"
	"regexp"
)

func init() {
	mfDecoders.Add(v1dec.GetDecoder)
	mfGenerators.Add(v1gen.GetGenerator)
}

var mfDecoders = make(modfile.Decoders)
var mfGenerators = make(modfile.Generators)

var RegExp = regexp.MustCompile(`^Modfile\.(?:yml|yaml)$`)

func open(fSys fs.FS) (fs.File, error) {
	mfPath, err := helper_file_sys.FindFile(fSys, RegExp.MatchString)
	if err != nil {
		return nil, err
	}
	if mfPath == "" {
		return nil, errors.New("modfile not found")
	}
	file, err := fSys.Open(mfPath)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func decode(r io.Reader) (*modfile.MfWrapper, error) {
	mf := modfile.New(mfDecoders, mfGenerators)
	err := yaml.NewDecoder(r).Decode(&mf)
	if err != nil {
		return nil, err
	}
	return mf, nil
}

func GetModule(fSys fs.FS) (module_lib.Module, error) {
	file, err := open(fSys)
	if err != nil {
		return module_lib.Module{}, err
	}
	defer file.Close()
	mf, err := decode(file)
	if err != nil {
		return module_lib.Module{}, err
	}
	mod, err := mf.GetModule()
	if err != nil {
		return module_lib.Module{}, err
	}
	if mod == nil {
		return module_lib.Module{}, errors.New("nil module")
	}
	return *mod, nil
}
