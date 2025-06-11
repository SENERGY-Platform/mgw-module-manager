package models

type Repository struct {
	Source         string
	Default        bool
	Channels       []string
	DefaultChannel string
}

type RepoModuleVariantBase struct {
	ID      string `json:"id"`
	Source  string `json:"source"`
	Channel string `json:"channel"`
}

type RepoModuleVariant struct {
	RepoModuleVariantBase
	Name    string `json:"name"`
	Desc    string `json:"description"`
	Version string `json:"version"`
}
