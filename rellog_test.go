package rellog_test

import (
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
	"github.com/tooppoo/rellog"
)

func TestE2E(t *testing.T) {
	directories := []string{
		"e2e/add-empty",
		"e2e/add",
		"e2e/amend",
		"e2e/check",
		"e2e/init",
		"e2e/prepare",
		"e2e/ready",
		"e2e/status",
		"e2e/workflow",
	}

	for _, dir := range directories {
		t.Run(dir, func(t *testing.T) {
			testscript.Run(t, testscript.Params{
				Dir: dir,
			})
		})
	}
}

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"rellog": rellog.Main,
	})
}
