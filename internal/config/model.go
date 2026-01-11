package config

type Model struct {
	HTTP              HTTPConfig `yaml:"HTTP"`
	Repo              RepoConfig `yaml:"Repo"`
	AccrualSystem     HTTPConfig `yaml:"AccrualSystem"`
	PasswordCryptoKey string     `yaml:"PasswordCryptoKey"`
}

type HTTPConfig struct {
	Host        string
	SecretToken string
}

type RepoConfig struct {
	PsqlConfig
}

type PsqlConfig struct {
	PsqlConnString string
}
