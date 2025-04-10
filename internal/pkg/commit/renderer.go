package commit

import (
	"strings"
)

func NewRenderer(template string) *Renderer {
	return &Renderer{
		template: template,
	}
}

type Renderer struct {
	template   string
	lastRender string
}

func (c *Renderer) String() string {
	return c.lastRender
}

func (c *Renderer) Linebreaks() int {
	count := 0
	r := '\n'
	for _, x := range c.lastRender {
		if x == r {
			count++
		}
	}
	return count
}

type TemplateValue struct {
	Key    string
	Value  any
	Format string
}

func (c *Renderer) Render(data []TemplateValue) string {
	s := strings.TrimPrefix(c.template, "\n")
	for _, cd := range data {
		fv := ""
		switch v := cd.Value.(type) {
		case bool:
			if v {
				fv = "yes"
			} else {
				fv = "no"
			}

		case string:
			fv = v

		default:
			panic("unsupported type")
		}

		if len(fv) > 0 {
			fv = strings.ReplaceAll(cd.Format, "{{value}}", fv)
		}
		s = strings.ReplaceAll(s, "{{"+cd.Key+"}}", fv)
	}
	c.lastRender = strings.TrimSpace(s)
	return c.String()
}
