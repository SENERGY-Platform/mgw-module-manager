package repositories

import (
	models_repositories "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/repositories"
)

type Repository struct {
	Handler  repositoryHandler
	Priority int
}

type moduleWrapper struct {
	models_repositories.Module
	FSysRef string
}
