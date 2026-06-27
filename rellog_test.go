package rellog_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
	"github.com/tooppoo/rellog"
)

func TestE2E(t *testing.T) {
	// Detect the GitHub repository URL from the current git remote.
	// This is passed to each test's workdir via a git repo so that
	// detectGitHubURL() works correctly in the subprocess context.
	out, _ := exec.Command("git", "remote", "get-url", "origin").Output()
	rawOrigin := strings.TrimSpace(string(out))

	testscript.Run(t, testscript.Params{
		Dir: "e2e",
		Setup: func(env *testscript.Env) error {
			if rawOrigin == "" {
				return nil
			}
			if err := exec.Command("git", "init", env.WorkDir).Run(); err != nil {
				return err
			}
			return exec.Command("git", "-C", env.WorkDir, "remote", "add", "origin", rawOrigin).Run()
		},
	})
}

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"rellog": rellog.Main,
	})
}
