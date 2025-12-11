package repositories

import (
	models_handler_repo "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/repository"
)

type Repository struct {
	Handler  repositoryHandler
	Priority int
}

type moduleWrapper struct {
	models_handler_repo.Module
	FSysRef string
}
