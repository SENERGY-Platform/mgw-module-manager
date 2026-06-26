package repositories

import (
	pkg_models "github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
)

type moduleWrapper struct {
	pkg_models.RepositoryModule
	RepoType string
	FSysRef  string
}
