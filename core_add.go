package rellog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type addOptions struct {
	Kind    string
	Targets []string
	Body    string
	Issues  []int
	PRs     []int
}

func addEntry(opts addOptions) error {
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return &exitError{ExitNotInitialized, "run `rellog init` first"}
	}
	if info, err := os.Stat(entriesDir()); err == nil && !info.IsDir() {
		return &exitError{ExitInvalidStructure, entriesDir() + " is not a directory"}
	}
	files, err := os.ReadDir(entriesDir())
	if err != nil {
		return err
	}
	count := 0
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".md") {
			count++
		}
	}

	e := entry{
		Kind:    opts.Kind,
		Targets: opts.Targets,
		Issues:  opts.Issues,
		PRs:     opts.PRs,
		Body:    opts.Body,
	}
	filename := fmt.Sprintf("%04d.md", count+1)
	return os.WriteFile(filepath.Join(entriesDir(), filename), []byte(formatEntry(e)), 0644)
}
