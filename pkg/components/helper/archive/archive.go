package archive

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
)

func ExtractTarGz(rc io.Reader, targetPath string) (string, error) {
	gzipReader, err := gzip.NewReader(rc)
	if err != nil {
		return "", err
	}
	defer gzipReader.Close()
	tarReader := tar.NewReader(gzipReader)
	var rootDir string
	for {
		tarHeader, err := tarReader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
		if tarHeader.Typeflag == tar.TypeDir {
			if rootDir == "" {
				parts := strings.Split(tarHeader.Name, string(os.PathSeparator))
				if len(parts) > 0 {
					rootDir = parts[0]
				}
			}
			if err = os.MkdirAll(path.Join(targetPath, tarHeader.Name), fs.FileMode(tarHeader.Mode)); err != nil {
				return "", err
			}
		}
		if tarHeader.Typeflag == tar.TypeReg {
			if err = writeFile(path.Join(targetPath, tarHeader.Name), tarHeader.Mode, tarReader); err != nil {
				return "", err
			}
		}
	}
	return rootDir, nil
}

func writeFile(name string, mode int64, reader *tar.Reader) error {
	file, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, fs.FileMode(mode))
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, reader)
	if err != nil {
		return err
	}
	return nil
}
