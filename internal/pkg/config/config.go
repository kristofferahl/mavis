package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/log"
	yaml "gopkg.in/yaml.v3"
)

type Config struct {
	path string

	Include []Include `yaml:"include,omitempty" json:"include,omitempty"`

	Theme    string `yaml:"theme" json:"theme"`
	Chip     string `yaml:"chip,omitempty" json:"chip,omitempty"`
	Template string `yaml:"template" json:"template"`

	Fields []*Field `yaml:"fields" json:"fields"`
}

func New(path string) *Config {
	c := Config{
		path: path,

		Theme: "charm",
		Chip:  "",

		Template: `
{{type}}{{scope}}{{breaking_glyph}}: {{description}}

{{breaking_body}}{{body}}`,

		Fields: make([]*Field, 0),
	}

	c.Fields = append(c.Fields, &Field{
		Type:    "select",
		Title:   "type of commit",
		Default: "feat",
		Formatting: []FormattingRule{
			{
				Key:    "type",
				Format: "{{value}}",
			},
		},
		Options: []SelectOption{
			{
				Key:   "feat",
				Value: "feat",
			},
			{
				Key:   "fix",
				Value: "fix",
			},
			{
				Key:   "chore",
				Value: "chore",
			},
		},
	})
	c.Fields = append(c.Fields, &Field{
		Type:        "input",
		Title:       "scope of the commit",
		Description: "noun describing a section of the codebase",
		Placeholder: "e.g. api, ui, app etc.",
		Formatting: []FormattingRule{
			{
				Key:    "scope",
				Format: "({{value}})",
			},
		},
	})
	c.Fields = append(c.Fields, &Field{
		Type:        "input",
		Title:       "summary of the change",
		Description: "a short description of the change",
		Placeholder: "e.g. add config file",
		Required:    true,
		Formatting: []FormattingRule{
			{
				Key:    "description",
				Format: "{{value}}",
			},
		},
	})
	c.Fields = append(c.Fields, &Field{
		Type:        "confirm",
		Title:       "breaking change?",
		Description: "if yes, describe the breaking change in detail",
		Formatting: []FormattingRule{
			{
				Key:    "breaking_glyph",
				Format: "!",
				When:   "true",
			},
			{
				Key:    "breaking_glyph",
				Format: "",
				When:   "false",
			},
			{
				Key:    "breaking_body",
				Format: "BREAKING CHANGE: ",
				When:   "true",
			},
			{
				Key:    "breaking_body",
				Format: "",
				When:   "false",
			},
		},
	})
	c.Fields = append(c.Fields, &Field{
		Type:        "text",
		Title:       "describe the change in detail (optional)",
		Description: "what is the motivation for this change",
		Formatting: []FormattingRule{
			{
				Key:    "body",
				Format: "{{value}}",
			},
		},
	})

	return &c
}

func (c *Config) Exists() bool {
	if _, err := os.Stat(c.path); os.IsNotExist(err) {
		return false
	}
	return true
}

func (c *Config) Read() error {
	return c.read(c.path)
}

func (c *Config) read(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file, %w", err)
	}
	if err := yaml.Unmarshal(b, c); err != nil {
		return fmt.Errorf("failed to unmarshal config, %w", err)
	}

	pwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory, %w", err)
	}

	if c.path == path {
		log.Debug("allowing includes from root config")
		for _, i := range c.Include {
			if i.Match(pwd) {
				log.Debug("including config", "condition", i.When, "path", i.Path)
				includePath, err := resolvePath(i.Path)
				if err != nil {
					return fmt.Errorf("failed to resolve include path, %w", err)
				}
				if err := c.read(includePath); err != nil {
					return fmt.Errorf("include failed, %w", err)
				}
			} else {
				log.Debug("skipping include", "condition", i.When, "path", i.Path)
			}
		}
	} else {
		log.Debug("skipping includes, only allowed from root config", "path", path)
	}

	return nil
}

func (c *Config) Write() error {
	s, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config, %w", err)
	}
	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config dir, %w", err)
	}
	if err := os.WriteFile(c.path, s, 0644); err != nil {
		return fmt.Errorf("failed to write config file, %w", err)
	}
	return nil
}
