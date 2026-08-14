package service

// OpenAIAccountLevelConfig describes one project-specific account sharing tier.
type OpenAIAccountLevelConfig struct {
	Key                string   `json:"key"`
	Label              string   `json:"label"`
	Aliases            []string `json:"aliases"`
	SortOrder          int      `json:"sort_order"`
	Enabled            bool     `json:"enabled"`
	RequiresProxyLogin bool     `json:"requires_proxy_login"`
}
