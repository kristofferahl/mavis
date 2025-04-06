package config

type Config struct {
	Theme string `yaml:"theme"`
}

func New() Config {
	return Config{
		Theme: "",
	}
}
