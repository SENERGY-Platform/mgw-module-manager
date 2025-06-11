package mod_repos_hdl

import (
	"github.com/SENERGY-Platform/mgw-module-manager/pkg/models"
)

type moduleVariant struct {
	models.RepoModuleVariant
	FSysRef string
}
