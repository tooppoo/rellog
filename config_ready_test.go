package rellog

import (
	"strings"
	"testing"

	kdl "github.com/njreid/gokdl2"
	"github.com/njreid/gokdl2/document"
)

func TestValidateReadyConfig(t *testing.T) {
	tests := []struct {
		name     string
		children string
		wantCode string
	}{
		{name: "omitted"},
		{name: "empty", children: "ready {}"},
		{name: "repeatable verify scripts", children: `ready {
			verify { script "scripts/verify.sh"; script "scripts/report.sh" }
		}`},
		{name: "duplicate ready", children: "ready {}; ready {}", wantCode: "error[rellog.ready.duplicate]"},
		{name: "ready argument", children: `ready "value" {}`, wantCode: "error[rellog.ready.argument_count]"},
		{name: "ready property", children: `ready mode="strict" {}`, wantCode: "error[rellog.ready.unknown_property]"},
		{name: "unknown ready child", children: "ready { check {} }", wantCode: "error[rellog.ready.unknown_node]"},
		{name: "duplicate verify", children: "ready { verify {}; verify {} }", wantCode: "error[rellog.ready.verify.duplicate]"},
		{name: "verify argument", children: `ready { verify "value" {} }`, wantCode: "error[rellog.ready.verify.argument_count]"},
		{name: "verify property", children: `ready { verify mode="strict" {} }`, wantCode: "error[rellog.ready.verify.unknown_property]"},
		{name: "unknown verify child", children: `ready { verify { command "scripts/verify.sh" } }`, wantCode: "error[rellog.ready.verify.unknown_node]"},
		{name: "script without argument", children: "ready { verify { script } }", wantCode: "error[rellog.ready.verify.script.argument_count]"},
		{name: "script with multiple arguments", children: `ready { verify { script "one.sh" "two.sh" } }`, wantCode: "error[rellog.ready.verify.script.argument_count]"},
		{name: "script property", children: `ready { verify { script "scripts/verify.sh" mode="strict" } }`, wantCode: "error[rellog.ready.verify.script.unknown_property]"},
		{name: "script children", children: `ready { verify { script "scripts/verify.sh" { nested } } }`, wantCode: "error[rellog.ready.verify.script.unexpected_children]"},
		{name: "script non-string", children: "ready { verify { script true } }", wantCode: "error[rellog.ready.verify.script.type]"},
		{name: "script non-canonical path", children: `ready { verify { script "../scripts/verify.sh" } }`, wantCode: "error[rellog.ready.verify.script.path]"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rellogNode := parseReadyTestRellogNode(t, test.children)
			errs := validateReadyConfig(rellogNode)
			if test.wantCode == "" {
				if len(errs) > 0 {
					t.Fatalf("validateReadyConfig() errors = %#v", errs)
				}
				return
			}
			if len(errs) != 1 {
				t.Fatalf("validateReadyConfig() error count = %d, want 1: %#v", len(errs), errs)
			}
			if errs[0].Code != test.wantCode {
				t.Fatalf("validateReadyConfig() code = %q, want %q", errs[0].Code, test.wantCode)
			}
		})
	}
}

func TestDecodeReadyConfigPreservesScriptOrder(t *testing.T) {
	rellogNode := parseReadyTestRellogNode(t, `ready {
		verify {
			script "scripts/verify-first.sh"
			script "scripts/verify-second.sh"
		}
	}`)

	got := decodeReadyConfig(rellogNode)
	assertStringSliceEqual(t, "VerifyScripts", got.VerifyScripts, []string{"scripts/verify-first.sh", "scripts/verify-second.sh"})
}

func TestValidateRellogConfigAcceptsReadyVerify(t *testing.T) {
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
		ready { verify { script "scripts/verify.sh" } }
	}`))
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}

	if errs := validateRellogConfig(doc); len(errs) > 0 {
		t.Fatalf("validateRellogConfig() errors = %#v", errs)
	}
}

func parseReadyTestRellogNode(t *testing.T, children string) *document.Node {
	t.Helper()
	doc, err := kdl.Parse(strings.NewReader("rellog config-version=1 {\n" + children + "\n}"))
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	return doc.Nodes[0]
}
