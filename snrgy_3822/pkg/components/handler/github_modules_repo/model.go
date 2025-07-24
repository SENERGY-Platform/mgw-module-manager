package github_modules_repo

type Channel struct {
	Name      string
	Reference string
	Priority  int
	Blacklist []string
}
