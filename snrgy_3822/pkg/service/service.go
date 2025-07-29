package service

type Service struct {
	reposHdl RepositoriesHandler
	modsHdl  ModulesHandler
}

func New(reposHdl RepositoriesHandler, modsHdl ModulesHandler) *Service {
	return &Service{
		reposHdl: reposHdl,
		modsHdl:  modsHdl,
	}
}
