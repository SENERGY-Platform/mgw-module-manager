package github

type Source struct {
	Owner      string    `json:"owner"`
	Repository string    `json:"repository"`
	Reference  string    `json:"reference"`
	Priority   int       `json:"priority"`
	Channels   []Channel `json:"channels"`
}

type Channel struct {
	Name      string   `json:"name"`
	Priority  int      `json:"priority"`
	Blacklist []string `json:"blacklist"`
}
