package rellog_test

import (
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
	"github.com/tooppoo/rellog"
)

func TestE2E(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "e2e",
	})
}
func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"rellog": rellog.Main,
	})
}
