package github

type Channel struct {
	Name      string
	Reference string
	Priority  int
	Blacklist []string
}
