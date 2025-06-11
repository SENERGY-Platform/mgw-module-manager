package fs_util

import (
	"io"
	"io/fs"
	"os"
	"path"
)

func CopyFile(fSys fs.FS, dstPath, srcPath string) error {
	src, err := fSys.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()
	fileInfo, err := src.Stat()
	if err != nil {
		return err
	}
	dst, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fileInfo.Mode())
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	if err != nil {
		return err
	}
	return nil
}

func CopyAll(fSys fs.FS, dstPath string) error {
	return fs.WalkDir(fSys, ".", func(currentPath string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if dirEntry.IsDir() {
			fileInfo, err := dirEntry.Info()
			if err != nil {
				return err
			}
			err = os.Mkdir(path.Join(dstPath, currentPath), fileInfo.Mode())
			if err != nil && !os.IsExist(err) {
				return err
			}
		} else {
			return CopyFile(fSys, path.Join(dstPath, currentPath), currentPath)
		}
		return nil
	})
}

func FindFile(fSys fs.FS, match func(v string) bool) (string, error) {
	var filePath string
	err := fs.WalkDir(fSys, ".", func(currentPath string, dirEntry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !dirEntry.IsDir() && match(dirEntry.Name()) {
			filePath = currentPath
			return fs.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return filePath, nil
}
