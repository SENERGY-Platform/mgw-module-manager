package repositories

import (
	"errors"

	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
)

var ErrNotFound = errors.New("not found")

type Repository struct {
	Handler  repositoryHandler
	Priority int
}

type moduleWrapper struct {
	pkg_models.RepositoryModule
	FSysRef string
}
