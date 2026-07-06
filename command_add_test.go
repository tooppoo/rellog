package rellog

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRealAddFlagsChanged(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{"no flags", nil, false},
		{"debug-datetime only", []string{"--debug-datetime", "20260101T000000.000000000Z"}, false},
		{"kind only", []string{"--kind", "changed"}, true},
		{"target only", []string{"--target", "cli"}, true},
		{"body only", []string{"--body", "Body"}, true},
		{"link only", []string{"--link", "https://example.com"}, true},
		{"link with debug-datetime", []string{"--link", "https://example.com", "--debug-datetime", "20260101T000000.000000000Z"}, true},
		{"all flags", []string{"--kind", "changed", "--target", "cli", "--body", "Body", "--link", "https://example.com"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := cmdAdd()
			if err := cmd.Flags().Parse(tt.args); err != nil {
				t.Fatalf("parse flags: %v", err)
			}
			if got := realAddFlagsChanged(cmd); got != tt.want {
				t.Errorf("realAddFlagsChanged() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldUseAddForm(t *testing.T) {
	pipeFile := func(t *testing.T) *os.File {
		t.Helper()
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("os.Pipe: %v", err)
		}
		t.Cleanup(func() {
			_ = r.Close()
			_ = w.Close()
		})
		return r
	}

	stubIsTerminal := func(t *testing.T, fn func(*os.File) bool) {
		t.Helper()
		orig := checkIsTerminal
		checkIsTerminal = fn
		t.Cleanup(func() { checkIsTerminal = orig })
	}

	t.Run("both terminals", func(t *testing.T) {
		stubIsTerminal(t, func(*os.File) bool { return true })
		if !shouldUseAddForm(pipeFile(t), pipeFile(t)) {
			t.Error("want true when both in and out are terminals")
		}
	})

	t.Run("stdin not a terminal", func(t *testing.T) {
		nonTTY := pipeFile(t)
		stubIsTerminal(t, func(f *os.File) bool { return f != nonTTY })
		if shouldUseAddForm(nonTTY, pipeFile(t)) {
			t.Error("want false when stdin is not a terminal")
		}
	})

	t.Run("stdout not a terminal", func(t *testing.T) {
		nonTTY := pipeFile(t)
		stubIsTerminal(t, func(f *os.File) bool { return f != nonTTY })
		if shouldUseAddForm(pipeFile(t), nonTTY) {
			t.Error("want false when stdout is not a terminal")
		}
	})

	t.Run("neither is a terminal", func(t *testing.T) {
		stubIsTerminal(t, func(*os.File) bool { return false })
		if shouldUseAddForm(pipeFile(t), pipeFile(t)) {
			t.Error("want false when neither is a terminal")
		}
	})

	t.Run("non-file reader and writer", func(t *testing.T) {
		stubIsTerminal(t, func(*os.File) bool { return true })
		if shouldUseAddForm(strings.NewReader(""), &bytes.Buffer{}) {
			t.Error("want false for non-*os.File in/out even if the stub reports terminals")
		}
	})

	t.Run("file reader with non-file writer", func(t *testing.T) {
		stubIsTerminal(t, func(*os.File) bool { return true })
		if shouldUseAddForm(pipeFile(t), &bytes.Buffer{}) {
			t.Error("want false when out is not an *os.File")
		}
	})
}

// TestCmdAddLaunchesFormOnTTY exercises the RunE wiring end-to-end: with no
// real add flags and terminal-like stdin/stdout, cmdAdd must route into the
// form path and persist the submitted values through addEntry.
func TestCmdAddLaunchesFormOnTTY(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := initRellog(); err != nil {
		t.Fatalf("initRellog: %v", err)
	}

	orig := checkIsTerminal
	checkIsTerminal = func(*os.File) bool { return true }
	t.Cleanup(func() { checkIsTerminal = orig })

	formRan := false
	stubAddFormProgram(t, func(m addFormModel, _ io.Reader, _ io.Writer) (addFormModel, error) {
		formRan = true
		m.kind.SetValue("changed")
		m.submitted = true
		return m, nil
	})

	pipeEnd := func() *os.File {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("os.Pipe: %v", err)
		}
		t.Cleanup(func() {
			_ = r.Close()
			_ = w.Close()
		})
		return r
	}

	const ts = "20260102T000000.000000000Z"
	cmd := cmdAdd()
	cmd.SetIn(pipeEnd())
	cmd.SetOut(pipeEnd())
	cmd.SetArgs([]string{"--debug-datetime", ts})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !formRan {
		t.Fatal("expected the TTY form path to run")
	}
	if _, err := os.Stat(filepath.Join(entriesDir(), ts+".json")); err != nil {
		t.Fatalf("entry file not written: %v", err)
	}
}

// stubAddFormProgram replaces the Bubble Tea program loop with fn for the
// duration of the test.
func stubAddFormProgram(t *testing.T, fn func(addFormModel, io.Reader, io.Writer) (addFormModel, error)) {
	t.Helper()
	orig := runAddFormProgram
	runAddFormProgram = fn
	t.Cleanup(func() { runAddFormProgram = orig })
}

func TestRunAddForm(t *testing.T) {
	setupInitializedDir := func(t *testing.T) {
		t.Helper()
		t.Chdir(t.TempDir())
		if err := initRellog(); err != nil {
			t.Fatalf("initRellog: %v", err)
		}
	}

	t.Run("not initialized", func(t *testing.T) {
		t.Chdir(t.TempDir())
		stubAddFormProgram(t, func(addFormModel, io.Reader, io.Writer) (addFormModel, error) {
			t.Fatal("the form must not launch before init")
			return addFormModel{}, nil
		})
		err := runAddForm(strings.NewReader(""), &bytes.Buffer{}, "")
		var ee *exitError
		if !errors.As(err, &ee) || ee.Code != ExitNotInitialized {
			t.Fatalf("err = %v, want exitError with ExitNotInitialized", err)
		}
	})

	t.Run("entries path is not a directory", func(t *testing.T) {
		setupInitializedDir(t)
		if err := os.RemoveAll(entriesDir()); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(entriesDir(), []byte(""), 0644); err != nil {
			t.Fatal(err)
		}
		stubAddFormProgram(t, func(addFormModel, io.Reader, io.Writer) (addFormModel, error) {
			t.Fatal("the form must not launch with a broken entries dir")
			return addFormModel{}, nil
		})
		err := runAddForm(strings.NewReader(""), &bytes.Buffer{}, "")
		var ee *exitError
		if !errors.As(err, &ee) || ee.Code != ExitInvalidStructure {
			t.Fatalf("err = %v, want exitError with ExitInvalidStructure", err)
		}
	})

	t.Run("existing empty entry fails before the form launches", func(t *testing.T) {
		setupInitializedDir(t)
		if err := addEmptyEntry(""); err != nil {
			t.Fatalf("addEmptyEntry: %v", err)
		}
		stubAddFormProgram(t, func(addFormModel, io.Reader, io.Writer) (addFormModel, error) {
			t.Fatal("the form must not launch when a submit can never succeed")
			return addFormModel{}, nil
		})
		err := runAddForm(strings.NewReader(""), &bytes.Buffer{}, "")
		var ee *exitError
		if !errors.As(err, &ee) || ee.Code != ExitEntryConflict {
			t.Fatalf("err = %v, want exitError with ExitEntryConflict", err)
		}
	})

	t.Run("cancelled form writes nothing", func(t *testing.T) {
		setupInitializedDir(t)
		stubAddFormProgram(t, func(m addFormModel, _ io.Reader, _ io.Writer) (addFormModel, error) {
			m.cancelled = true
			return m, nil
		})
		if err := runAddForm(strings.NewReader(""), &bytes.Buffer{}, ""); err != nil {
			t.Fatalf("cancel should not error: %v", err)
		}
		files, err := os.ReadDir(entriesDir())
		if err != nil {
			t.Fatal(err)
		}
		if len(files) != 0 {
			t.Fatalf("cancel must not write entries, found %d files", len(files))
		}
	})

	t.Run("submitted form goes through addEntry", func(t *testing.T) {
		setupInitializedDir(t)
		stubAddFormProgram(t, func(m addFormModel, _ io.Reader, _ io.Writer) (addFormModel, error) {
			if len(m.kindCandidates) == 0 {
				t.Error("form model should receive kind candidates from config")
			}
			m.kind.SetValue("changed")
			m.targets.SetValue("cli api")
			m.body.SetValue("From the form")
			m.links.SetValue("https://example.com/1, https://example.com/2")
			m.submitted = true
			return m, nil
		})
		const ts = "20260101T000000.000000000Z"
		if err := runAddForm(strings.NewReader(""), &bytes.Buffer{}, ts); err != nil {
			t.Fatalf("runAddForm: %v", err)
		}
		data, err := os.ReadFile(filepath.Join(entriesDir(), ts+".json"))
		if err != nil {
			t.Fatalf("entry file not written: %v", err)
		}
		e, err := parseEntryJSON(data)
		if err != nil {
			t.Fatal(err)
		}
		if e.Kind != "changed" || e.Body != "From the form" {
			t.Errorf("entry = %+v", e)
		}
		if want := []string{"cli", "api"}; !reflect.DeepEqual(e.Targets, want) {
			t.Errorf("Targets = %v, want %v", e.Targets, want)
		}
		if want := []string{"https://example.com/1", "https://example.com/2"}; !reflect.DeepEqual(e.Links, want) {
			t.Errorf("Links = %v, want %v", e.Links, want)
		}
	})

	t.Run("submitted invalid kind fails via addEntry validation", func(t *testing.T) {
		setupInitializedDir(t)
		stubAddFormProgram(t, func(m addFormModel, _ io.Reader, _ io.Writer) (addFormModel, error) {
			m.kind.SetValue("nope")
			m.submitted = true
			return m, nil
		})
		err := runAddForm(strings.NewReader(""), &bytes.Buffer{}, "")
		var ee *exitError
		if !errors.As(err, &ee) || ee.Code != ExitCheckFailed {
			t.Fatalf("err = %v, want exitError with ExitCheckFailed", err)
		}
	})

	t.Run("program error is propagated", func(t *testing.T) {
		setupInitializedDir(t)
		wantErr := errors.New("terminal exploded")
		stubAddFormProgram(t, func(addFormModel, io.Reader, io.Writer) (addFormModel, error) {
			return addFormModel{}, wantErr
		})
		if err := runAddForm(strings.NewReader(""), &bytes.Buffer{}, ""); !errors.Is(err, wantErr) {
			t.Fatalf("err = %v, want %v", err, wantErr)
		}
	})
}
