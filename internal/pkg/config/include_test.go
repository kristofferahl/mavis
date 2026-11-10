package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInclude_MatchComparesWholePathSegments(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		when string
		path string
		want bool
	}{
		{"exact directory matches", "path:/work/mavis", "/work/mavis", true},
		{"subdirectory matches", "path:/work/mavis", "/work/mavis/internal/pkg", true},
		{"trailing slash in condition still matches the directory", "path:/work/mavis/", "/work/mavis", true},
		{"trailing slash in condition still matches subdirectories", "path:/work/mavis/", "/work/mavis/sub", true},
		{"sibling sharing the name prefix does not match", "path:/work/mavis", "/work/mavis-mcp-changedir", false},
		{"sibling sharing the name prefix does not match with trailing slash", "path:/work/mavis/", "/work/mavis-test", false},
		{"parent directory does not match", "path:/work/mavis", "/work", false},
		{"unrelated directory does not match", "path:/work/mavis", "/other/mavis", false},
		{"home-relative condition matches under home", "path:~/develop/mavis/", filepath.Join(home, "develop", "mavis", "sub"), true},
		{"home-relative condition rejects sibling under home", "path:~/develop/mavis/", filepath.Join(home, "develop", "mavis-test"), false},
		{"root condition matches everything", "path:/", "/anything", true},
		{"empty condition matches everything", "", "/anything", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Include{When: tt.when}.Match(tt.path)
			if got != tt.want {
				t.Errorf("Include{When: %q}.Match(%q) = %v, want %v", tt.when, tt.path, got, tt.want)
			}
		})
	}
}
