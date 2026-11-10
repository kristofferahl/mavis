package commit

import "testing"

const sectionedTemplate = `{{type}}{{scope}}{{breaking_glyph}}: {{description}}

{{breaking_body}}{{body}}

{{ticket}}
`

func sectionedValues(breaking bool, body, ticket string) []TemplateValue {
	glyph, breakingBody := "", ""
	if breaking {
		glyph, breakingBody = "!", "BREAKING CHANGE: "
	}
	return []TemplateValue{
		{Key: "type", Value: "feat", Format: "{{value}}"},
		{Key: "scope", Value: "", Format: "({{value}})"},
		{Key: "breaking_glyph", Value: glyph, Format: "{{value}}"},
		{Key: "description", Value: "x", Format: "{{value}}"},
		{Key: "breaking_body", Value: breakingBody, Format: "{{value}}"},
		{Key: "body", Value: body, Format: "{{value}}"},
		{Key: "ticket", Value: ticket, Format: "Refs: {{value}}"},
	}
}

func TestRender_OptionalSections(t *testing.T) {
	tests := []struct {
		name     string
		breaking bool
		body     string
		ticket   string
		want     string
	}{
		{"all optional sections empty renders only the subject", false, "", "", "feat: x"},
		{"body only is separated by one blank line", false, "b", "", "feat: x\n\nb"},
		{"ticket only is separated by one blank line", false, "", "T-1", "feat: x\n\nRefs: T-1"},
		{"body and ticket each separated by one blank line", false, "b", "T-1", "feat: x\n\nb\n\nRefs: T-1"},
		{"breaking change prefixes the body", true, "b", "T-1", "feat!: x\n\nBREAKING CHANGE: b\n\nRefs: T-1"},
		{"breaking change without body drops the trailing space", true, "", "T-1", "feat!: x\n\nBREAKING CHANGE:\n\nRefs: T-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewRenderer(sectionedTemplate).Render(sectionedValues(tt.breaking, tt.body, tt.ticket))
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRender_BlankLineNormalisation(t *testing.T) {
	tests := []struct {
		name     string
		template string
		body     string
		want     string
	}{
		{"whitespace-only lines count as blank", "a\n  \n\t\n{{body}}", "b", "a\n\nb"},
		{"three or more blank lines in a body collapse to one", "a\n\n{{body}}", "b1\n\n\n\nb2", "a\n\nb1\n\nb2"},
		{"a single blank line is preserved", "a\n\n{{body}}", "b", "a\n\nb"},
		{"leading and trailing blank lines are trimmed", "\n\n  \n{{body}}\n\n \n", "b", "b"},
		{"trailing whitespace is stripped from every line", "a  \n{{body}}\t", "b ", "a\nb"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewRenderer(tt.template).Render([]TemplateValue{{Key: "body", Value: tt.body, Format: "{{value}}"}})
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
