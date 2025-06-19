package github_mod_repo_hdl

type Channel struct {
	Name      string
	Reference string
	Priority  int
	Blacklist []string
}
