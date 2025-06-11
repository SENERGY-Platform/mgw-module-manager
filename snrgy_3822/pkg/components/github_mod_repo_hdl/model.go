package github_mod_repo_hdl

type Channel struct {
	Name      string
	Reference string
	Default   bool
	Blacklist []string
}
