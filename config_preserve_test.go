package rellog

import (
	"strings"
	"testing"

	kdl "github.com/njreid/gokdl2"
	"github.com/njreid/gokdl2/document"
)

func TestValidatePreserveConfig(t *testing.T) {
	tests := []struct {
		name     string
		children string
		wantCode string
	}{
		{name: "omitted"},
		{name: "empty", children: "preserve {}"},
		{name: "preview and run scripts", children: `preserve {
			preview { script "scripts/preview.sh"; script "scripts/report.sh" }
			run { script "scripts/run.sh" }
		}`},
		{name: "duplicate preserve", children: "preserve {}; preserve {}", wantCode: "error[rellog.preserve.duplicate]"},
		{name: "preserve argument", children: `preserve "value" {}`, wantCode: "error[rellog.preserve.argument_count]"},
		{name: "preserve property", children: `preserve mode="strict" {}`, wantCode: "error[rellog.preserve.unknown_property]"},
		{name: "unknown preserve child", children: "preserve { verify {} }", wantCode: "error[rellog.preserve.unknown_node]"},
		{name: "duplicate preview", children: "preserve { preview {}; preview {} }", wantCode: "error[rellog.preserve.preview.duplicate]"},
		{name: "duplicate run", children: "preserve { run {}; run {} }", wantCode: "error[rellog.preserve.run.duplicate]"},
		{name: "phase argument", children: `preserve { preview "value" {} }`, wantCode: "error[rellog.preserve.preview.argument_count]"},
		{name: "phase property", children: `preserve { preview mode="strict" {} }`, wantCode: "error[rellog.preserve.preview.unknown_property]"},
		{name: "unknown phase child", children: "preserve { preview { command \"scripts/preview.sh\" } }", wantCode: "error[rellog.preserve.preview.unknown_node]"},
		{name: "script without argument", children: "preserve { preview { script } }", wantCode: "error[rellog.preserve.preview.script.argument_count]"},
		{name: "script with multiple arguments", children: `preserve { preview { script "one.sh" "two.sh" } }`, wantCode: "error[rellog.preserve.preview.script.argument_count]"},
		{name: "script property", children: `preserve { run { script "scripts/run.sh" mode="strict" } }`, wantCode: "error[rellog.preserve.run.script.unknown_property]"},
		{name: "script children", children: `preserve { run { script "scripts/run.sh" { nested } } }`, wantCode: "error[rellog.preserve.run.script.unexpected_children]"},
		{name: "script non-string", children: "preserve { run { script true } }", wantCode: "error[rellog.preserve.run.script.type]"},
		{name: "script non-canonical path", children: `preserve { run { script "../scripts/run.sh" } }`, wantCode: "error[rellog.preserve.run.script.path]"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rellogNode := parsePreserveTestRellogNode(t, test.children)
			errs := validatePreserveConfig(rellogNode)
			if test.wantCode == "" {
				if len(errs) > 0 {
					t.Fatalf("validatePreserveConfig() errors = %#v", errs)
				}
				return
			}
			if len(errs) != 1 {
				t.Fatalf("validatePreserveConfig() error count = %d, want 1: %#v", len(errs), errs)
			}
			if errs[0].Code != test.wantCode {
				t.Fatalf("validatePreserveConfig() code = %q, want %q", errs[0].Code, test.wantCode)
			}
		})
	}
}

func TestDecodePreserveConfigPreservesScriptOrder(t *testing.T) {
	rellogNode := parsePreserveTestRellogNode(t, `preserve {
		preview {
			script "scripts/preview-first.sh"
			script "scripts/preview-second.sh"
		}
		run {
			script "scripts/run-first.sh"
			script "scripts/run-second.sh"
		}
	}`)

	got := decodePreserveConfig(rellogNode)
	assertStringSliceEqual(t, "PreviewScripts", got.PreviewScripts, []string{"scripts/preview-first.sh", "scripts/preview-second.sh"})
	assertStringSliceEqual(t, "RunScripts", got.RunScripts, []string{"scripts/run-first.sh", "scripts/run-second.sh"})
}

func TestValidateRellogConfigAcceptsPreserve(t *testing.T) {
	doc, err := kdl.Parse(strings.NewReader(`rellog config-version=1 {
		paths {
			changelog "testdata-missing/CHANGELOG.md"
			entries "testdata-missing/entries"
			release-notes "testdata-missing/release-notes"
		}
		entries {
			kinds { kind "added" }
			targets { target "rellog" }
		}
		preserve { preview { script "scripts/preview.sh" } }
	}`))
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}

	if errs := validateRellogConfig(doc); len(errs) > 0 {
		t.Fatalf("validateRellogConfig() errors = %#v", errs)
	}
}

func parsePreserveTestRellogNode(t *testing.T, children string) *document.Node {
	t.Helper()
	doc, err := kdl.Parse(strings.NewReader("rellog config-version=1 {\n" + children + "\n}"))
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	return doc.Nodes[0]
}

func assertStringSliceEqual(t *testing.T, field string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %#v, want %#v", field, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s = %#v, want %#v", field, got, want)
		}
	}
}
