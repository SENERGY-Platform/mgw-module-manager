package mod_repos_hdl

import (
	models_repo "github.com/SENERGY-Platform/mgw-module-manager/pkg/models/repository"
)

type RepoHandlerWrapper struct {
	RepoHandler
	Priority int
}

type moduleWrapper struct {
	models_repo.Module
	FSysRef string
}
