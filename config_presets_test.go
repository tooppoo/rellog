package rellog

import (
	"strings"
	"testing"

	kdl "github.com/njreid/gokdl2"
	"github.com/njreid/gokdl2/document"
)

func TestValidateUsePresetsConfig(t *testing.T) {
	tests := []struct {
		name     string
		children string
		wantCode string
	}{
		{name: "omitted"},
		{name: "rust", children: `use-presets "rust"`},
		{name: "duplicate", children: `use-presets "rust"; use-presets "rust"`, wantCode: "error[rellog.use-presets.duplicate]"},
		{name: "unknown", children: `use-presets "go"`, wantCode: "error[rellog.use-presets.unknown]"},
		{name: "empty id", children: `use-presets ""`, wantCode: "error[rellog.use-presets.unknown]"},
		{name: "without argument", children: "use-presets", wantCode: "error[rellog.use-presets.argument_count]"},
		{name: "multiple arguments", children: `use-presets "rust" "go"`, wantCode: "error[rellog.use-presets.argument_count]"},
		{name: "property", children: `use-presets "rust" enabled=true`, wantCode: "error[rellog.use-presets.unknown_property]"},
		{name: "children", children: `use-presets "rust" { option "value" }`, wantCode: "error[rellog.use-presets.unexpected_children]"},
		{name: "non-string", children: "use-presets true", wantCode: "error[rellog.use-presets.type]"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rellogNode := parseUsePresetsTestRellogNode(t, test.children)
			errs := validateUsePresetsConfig(rellogNode)
			if test.wantCode == "" {
				if len(errs) > 0 {
					t.Fatalf("validateUsePresetsConfig() errors = %#v", errs)
				}
				return
			}
			if len(errs) != 1 {
				t.Fatalf("validateUsePresetsConfig() error count = %d, want 1: %#v", len(errs), errs)
			}
			if errs[0].Code != test.wantCode {
				t.Fatalf("validateUsePresetsConfig() code = %q, want %q", errs[0].Code, test.wantCode)
			}
		})
	}
}

func TestDecodeUsePresetsConfigPreservesDeclarationOrder(t *testing.T) {
	rellogNode := parseUsePresetsTestRellogNode(t, `use-presets "first"; use-presets "second"`)

	got := decodeUsePresetsConfig(rellogNode)
	assertStringSliceEqual(t, "preset ids", got, []string{"first", "second"})
}

func TestValidateRellogConfigAcceptsUsePresets(t *testing.T) {
	doc := parseUsePresetsTestConfig(t, `use-presets "rust"`)

	if errs := validateRellogConfig(doc); len(errs) > 0 {
		t.Fatalf("validateRellogConfig() errors = %#v", errs)
	}
}

func TestValidateRellogConfigRejectsInvalidUsePresets(t *testing.T) {
	doc := parseUsePresetsTestConfig(t, `use-presets "go"`)

	errs := validateRellogConfig(doc)
	if len(errs) != 1 {
		t.Fatalf("validateRellogConfig() error count = %d, want 1: %#v", len(errs), errs)
	}
	if errs[0].Code != "error[rellog.use-presets.unknown]" {
		t.Fatalf("validateRellogConfig() code = %q, want %q", errs[0].Code, "error[rellog.use-presets.unknown]")
	}
}

func parseUsePresetsTestRellogNode(t *testing.T, children string) *document.Node {
	t.Helper()
	doc, err := kdl.Parse(strings.NewReader("rellog config-version=1 {\n" + children + "\n}"))
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	return doc.Nodes[0]
}

func parseUsePresetsTestConfig(t *testing.T, children string) *document.Document {
	t.Helper()
	config := `rellog config-version=1 {
		paths {
			changelog "testdata-missing/CHANGELOG.md"
			entries "testdata-missing/entries"
			release-notes "testdata-missing/release-notes"
		}
		entries {
			kinds { kind "added" }
			targets { target "rellog" }
		}
		` + children + `
	}`
	doc, err := kdl.Parse(strings.NewReader(config))
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	return doc
}
