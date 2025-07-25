package repositories

import (
	models_repo "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/repository"
)

type Repository struct {
	Handler  repositoryHandler
	Priority int
}

type moduleWrapper struct {
	models_repo.Module
	FSysRef string
}
