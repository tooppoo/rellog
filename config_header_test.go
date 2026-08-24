package rellog

import (
	"strings"
	"testing"

	kdl "github.com/njreid/gokdl2"
)

func TestValidateRellogNodeHeader(t *testing.T) {
	tests := []struct {
		name       string
		rellogHead string
		wantCode   string
	}{
		{name: "version 1", rellogHead: "config-version=1"},
		{name: "missing", wantCode: "error[rellog.config-version.missing]"},
		{name: "unsupported", rellogHead: "config-version=2", wantCode: "error[rellog.config-version.unsupported]"},
		{name: "string", rellogHead: `config-version="1"`, wantCode: "error[rellog.config-version.unsupported]"},
		{name: "unknown property", rellogHead: "config-version=1 mode=release", wantCode: "error[rellog.unknown_property]"},
		{name: "argument", rellogHead: `"value" config-version=1`, wantCode: "error[rellog.argument_count]"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			doc, err := kdl.Parse(strings.NewReader("rellog " + test.rellogHead + " {}"))
			if err != nil {
				t.Fatalf("parse config: %v", err)
			}
			errs := validateRellogNodeHeader(doc.Nodes[0])
			if test.wantCode == "" {
				if len(errs) > 0 {
					t.Fatalf("validateRellogNodeHeader() errors = %#v", errs)
				}
				return
			}
			if len(errs) != 1 {
				t.Fatalf("validateRellogNodeHeader() error count = %d, want 1: %#v", len(errs), errs)
			}
			if errs[0].Code != test.wantCode {
				t.Fatalf("validateRellogNodeHeader() code = %q, want %q", errs[0].Code, test.wantCode)
			}
		})
	}
}
