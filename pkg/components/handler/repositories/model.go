package handler_repositories

import (
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models/handler/repositories"
)

type Repository struct {
	Handler  repositoryHandler
	Priority int
}

type moduleWrapper struct {
	models_handler_repositories.Module
	FSysRef string
}
