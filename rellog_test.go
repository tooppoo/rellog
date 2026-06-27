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

	setup := func(env *testscript.Env) error {
		if rawOrigin == "" {
			return nil
		}
		if err := exec.Command("git", "init", env.WorkDir).Run(); err != nil {
			return err
		}
		return exec.Command("git", "-C", env.WorkDir, "remote", "add", "origin", rawOrigin).Run()
	}

	directories := []string{
		"e2e/add-empty",
		"e2e/add",
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
				Dir:   dir,
				Setup: setup,
			})
		})
	}
}

func TestMain(m *testing.M) {
	testscript.Main(m, map[string]func(){
		"rellog": rellog.Main,
	})
}
