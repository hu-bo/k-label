package conf

// Qdrant holds vector database configuration parsed from YAML.
type Qdrant struct {
	BaseURL    string `json:"base_url" yaml:"base_url"`
	Collection string `json:"collection" yaml:"collection"`
	APIKey     string `json:"api_key" yaml:"api_key"`
}
