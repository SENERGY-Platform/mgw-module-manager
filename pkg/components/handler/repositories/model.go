package repositories

import (
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
)

type Repository struct {
	Handler  repositoryHandler
	Priority int
}

type moduleWrapper struct {
	pkg_models.RepositoryModule
	FSysRef string
}
