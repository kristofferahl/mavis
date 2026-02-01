package config

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/kristofferahl/mavis/internal/pkg/commit"
)

type Field struct {
	ref huh.Field

	Type        string           `yaml:"type" json:"type"`
	Title       string           `yaml:"title" json:"title"`
	Description string           `yaml:"description,omitempty" json:"description,omitempty"`
	Required    bool             `yaml:"required" json:"required"`
	Placeholder string           `yaml:"placeholder,omitempty" json:"placeholder,omitempty"`
	Default     interface{}      `yaml:"default,omitempty" json:"default,omitempty"`
	Formatting  []FormattingRule `yaml:"format,omitempty" json:"format,omitempty"`
	Options     []SelectOption   `yaml:"options,omitempty" json:"options,omitempty"`
}

type SelectOption struct {
	Key   string `yaml:"key,omitempty" json:"key,omitempty"`
	Value string `yaml:"value" json:"value"`
}

func (f *Field) SetRef(ref huh.Field) {
	f.ref = ref
}

func (f *Field) TemplateValues() (values []commit.TemplateValue) {
	for _, rule := range f.Formatting {
		val := fmt.Sprintf("%v", f.ref.GetValue())
		if rule.When == "" || val == rule.When {
			values = append(values, commit.TemplateValue{
				Key:    rule.Key,
				Value:  f.ref.GetValue(),
				Format: rule.Format,
			})
		}
	}
	return
}

// TemplateValuesFrom returns template values using the provided value instead of the huh.Field reference
func (f *Field) TemplateValuesFrom(value interface{}) (values []commit.TemplateValue) {
	for _, rule := range f.Formatting {
		val := fmt.Sprintf("%v", value)
		if rule.When == "" || val == rule.When {
			values = append(values, commit.TemplateValue{
				Key:    rule.Key,
				Value:  value,
				Format: rule.Format,
			})
		}
	}
	return
}

type FormattingRule struct {
	Key    string `yaml:"key" json:"key"`
	Format string `yaml:"format" json:"format"`
	When   string `yaml:"when,omitempty" json:"when,omitempty"`
}
