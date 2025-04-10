package config

type Config struct {
	Theme    string `yaml:"theme" json:"theme"`
	Chip     string `yaml:"chip" json:"chip"`
	Template string `yaml:"template" json:"template"`

	Fields []*Field `yaml:"fields" json:"fields"`
}

func New() Config {
	return Config{
		Theme:  "",
		Chip:   "",
		Fields: make([]*Field, 0),
	}
}
