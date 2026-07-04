package rellog

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateEntrySchema(t *testing.T) {
	tests := []struct {
		name string
		json string
		want []string
	}{
		{
			name: "valid normal entry",
			json: `{"kind":"changed","targets":["cli"],"links":["https://example.com/issue/1"],"body":"Body"}`,
		},
		{
			name: "unknown field",
			json: `{"kind":"changed","targets":["cli"],"links":[],"body":"Body","extra":true}`,
			want: []string{"error[entry.unknown_field]"},
		},
		{
			name: "missing normal fields",
			json: `{}`,
			want: []string{
				"error[entry.kind.missing]",
				"error[entry.targets.missing]",
				"error[entry.links.missing]",
				"error[entry.body.missing]",
			},
		},
		{
			name: "normal field type errors",
			json: `{"kind":"changed","targets":[1],"links":[1],"body":null}`,
			want: []string{
				"error[entry.targets.invalid]",
				"error[entry.links.invalid]",
				"error[entry.body.invalid]",
			},
		},
		{
			name: "normal scalar targets and links",
			json: `{"kind":"changed","targets":"cli","links":"https://example.com","body":"Body"}`,
			want: []string{
				"error[entry.targets.invalid]",
				"error[entry.links.invalid]",
			},
		},
		{
			name: "normal invalid link and body",
			json: `{"kind":"changed","targets":["cli"],"links":["ftp://example.com"],"body":"<!-- rellog:reserved -->"}`,
			want: []string{
				"error[entry.links.invalid]",
				"error[entry.body.reserved_marker]",
			},
		},
		{
			name: "empty entry is valid with empty arrays and body",
			json: `{"kind":"empty","targets":[],"links":[],"body":""}`,
		},
		{
			name: "empty entry rejects fields",
			json: `{"kind":"empty","targets":["cli"],"links":["https://example.com"],"body":null}`,
			want: []string{
				"error[entry.body.invalid]",
				"error[entry.empty.targets.invalid]",
				"error[entry.empty.links.invalid]",
			},
		},
		{
			name: "empty entry requires structural fields",
			json: `{"kind":"empty"}`,
			want: []string{
				"error[entry.targets.missing]",
				"error[entry.links.missing]",
				"error[entry.body.missing]",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, err := parseEntryJSON([]byte(tt.json))
			if err != nil {
				t.Fatalf("parseEntryJSON() error = %v", err)
			}
			errs := validateEntrySchema(e)
			if len(errs) != len(tt.want) {
				t.Fatalf("got %d errors %#v, want %d %#v", len(errs), errs, len(tt.want), tt.want)
			}
			for i, want := range tt.want {
				if errs[i].Code != want {
					t.Fatalf("error %d code = %q, want %q; all errors: %#v", i, errs[i].Code, want, errs)
				}
			}
		})
	}
}

func TestValidateConsumedCacheDir(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		dir := makeConsumedCacheFixture(t, "v1.0.0",
			`{"schemaVersion":1,"releaseId":"v1.0.0","entries":[{"filename":"entry.json"}]}`,
			map[string]string{"entry.json": validEntryJSON()})
		if err := validateConsumedCacheDir(dir, "v1.0.0"); err != nil {
			t.Fatalf("validateConsumedCacheDir() error = %v", err)
		}
	})

	tests := []struct {
		name      string
		manifest  string
		entries   map[string]string
		releaseID string
		want      string
	}{
		{
			name:      "invalid manifest json",
			manifest:  `{`,
			entries:   map[string]string{},
			releaseID: "v1.0.0",
			want:      "invalid manifest JSON",
		},
		{
			name:      "unexpected schema version",
			manifest:  `{"schemaVersion":2,"releaseId":"v1.0.0","entries":[]}`,
			entries:   map[string]string{},
			releaseID: "v1.0.0",
			want:      "unexpected schema version",
		},
		{
			name:      "release mismatch",
			manifest:  `{"schemaVersion":1,"releaseId":"v2.0.0","entries":[]}`,
			entries:   map[string]string{},
			releaseID: "v1.0.0",
			want:      "release ID mismatch",
		},
		{
			name:      "duplicate filename",
			manifest:  `{"schemaVersion":1,"releaseId":"v1.0.0","entries":[{"filename":"entry.json"},{"filename":"entry.json"}]}`,
			entries:   map[string]string{"entry.json": validEntryJSON()},
			releaseID: "v1.0.0",
			want:      "duplicate filename",
		},
		{
			name:      "missing file",
			manifest:  `{"schemaVersion":1,"releaseId":"v1.0.0","entries":[{"filename":"missing.json"}]}`,
			entries:   map[string]string{},
			releaseID: "v1.0.0",
			want:      "manifest entry missing file",
		},
		{
			name:      "orphan file",
			manifest:  `{"schemaVersion":1,"releaseId":"v1.0.0","entries":[]}`,
			entries:   map[string]string{"orphan.json": validEntryJSON()},
			releaseID: "v1.0.0",
			want:      "orphan entry file",
		},
		{
			name:      "invalid copied entry",
			manifest:  `{"schemaVersion":1,"releaseId":"v1.0.0","entries":[{"filename":"entry.json"}]}`,
			entries:   map[string]string{"entry.json": `{"kind":"changed","targets":["cli"],"links":[],"body":null}`},
			releaseID: "v1.0.0",
			want:      "schema error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := makeConsumedCacheFixture(t, tt.releaseID, tt.manifest, tt.entries)
			err := validateConsumedCacheDir(dir, tt.releaseID)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("validateConsumedCacheDir() error = %v, want substring %q", err, tt.want)
			}
		})
	}

	t.Run("missing manifest", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, "entries"), 0755); err != nil {
			t.Fatal(err)
		}
		err := validateConsumedCacheDir(dir, "v1.0.0")
		if err == nil || !strings.Contains(err.Error(), "cannot read manifest") {
			t.Fatalf("validateConsumedCacheDir() error = %v", err)
		}
	})
}

func TestBuildConsumedCacheTemp(t *testing.T) {
	withTempWorkingDir(t)
	if err := os.MkdirAll(entriesDir(), 0755); err != nil {
		t.Fatal(err)
	}
	entryPath := filepath.Join(entriesDir(), "entry.json")
	if err := os.WriteFile(entryPath, []byte(validEntryJSON()), 0644); err != nil {
		t.Fatal(err)
	}

	tempDir, err := buildConsumedCacheTemp("v1.0.0", []entryFile{{name: "entry.json", path: entryPath}})
	if err != nil {
		t.Fatalf("buildConsumedCacheTemp() error = %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	if _, err := os.Stat(filepath.Join(tempDir, "manifest.json")); err != nil {
		t.Fatalf("manifest was not written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tempDir, "entries", "entry.json")); err != nil {
		t.Fatalf("entry copy was not written: %v", err)
	}

	_, err = buildConsumedCacheTemp("v1.0.0", []entryFile{{name: "missing.json", path: filepath.Join(entriesDir(), "missing.json")}})
	if err == nil {
		t.Fatal("buildConsumedCacheTemp() with missing entry file succeeded")
	}
}

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	if err := writeFileAtomic(path, []byte("first"), 0644); err != nil {
		t.Fatalf("writeFileAtomic() create error = %v", err)
	}
	if err := writeFileAtomic(path, []byte("second"), 0644); err != nil {
		t.Fatalf("writeFileAtomic() replace error = %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "second" {
		t.Fatalf("file content = %q, want second", got)
	}

	if err := writeFileAtomic(filepath.Join(dir, "missing", "file.txt"), []byte("x"), 0644); err == nil {
		t.Fatal("writeFileAtomic() with missing parent succeeded")
	}
}

func TestMergeChangelog(t *testing.T) {
	tests := []struct {
		name     string
		new      string
		existing string
		want     string
	}{
		{
			name: "empty changelog",
			new:  "## v1.0.0\n\n- Entry\n",
			want: "# CHANGELOG\n\n## v1.0.0\n\n- Entry\n",
		},
		{
			name:     "canonical heading",
			new:      "## v1.0.0\n",
			existing: "# CHANGELOG\n\n## v0.9.0\n",
			want:     "# CHANGELOG\n\n## v1.0.0\n\n## v0.9.0\n",
		},
		{
			name:     "no canonical heading",
			new:      "## v1.0.0\n",
			existing: "Existing\n",
			want:     "## v1.0.0\n\nExisting\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mergeChangelog(tt.new, tt.existing); got != tt.want {
				t.Fatalf("mergeChangelog() = %q, want %q", got, tt.want)
			}
		})
	}
}

func makeConsumedCacheFixture(t *testing.T, _, manifest string, entries map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "entries"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	for name, data := range entries {
		if err := os.WriteFile(filepath.Join(dir, "entries", name), []byte(data), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func validEntryJSON() string {
	return `{"kind":"changed","targets":["cli"],"links":[],"body":"Body"}`
}

func TestExtractChangelogSection(t *testing.T) {
	t.Run("finds section with next heading", func(t *testing.T) {
		content := "# CHANGELOG\n\n## v1.1.0\n\n### changed\n\nBody1\n\n## v1.0.0\n\n### added\n\nBody0\n"
		before, section, after, found, err := extractChangelogSection(content, "v1.1.0")
		if err != nil {
			t.Fatalf("extractChangelogSection() error = %v", err)
		}
		if !found {
			t.Fatal("expected found = true")
		}
		if want := "# CHANGELOG\n\n"; before != want {
			t.Errorf("before = %q, want %q", before, want)
		}
		if want := "## v1.1.0\n\n### changed\n\nBody1\n"; section != want {
			t.Errorf("section = %q, want %q", section, want)
		}
		if want := "## v1.0.0\n\n### added\n\nBody0\n"; after != want {
			t.Errorf("after = %q, want %q", after, want)
		}
		if got := spliceSection(before, section, after); got != content {
			t.Errorf("splice roundtrip = %q, want %q", got, content)
		}
	})

	t.Run("last section has no trailing separator", func(t *testing.T) {
		content := "# CHANGELOG\n\n## v1.0.0\n\n### added\n\nBody0\n"
		before, section, after, found, err := extractChangelogSection(content, "v1.0.0")
		if err != nil {
			t.Fatalf("extractChangelogSection() error = %v", err)
		}
		if !found {
			t.Fatal("expected found = true")
		}
		if after != "" {
			t.Errorf("after = %q, want empty", after)
		}
		if want := "## v1.0.0\n\n### added\n\nBody0\n"; section != want {
			t.Errorf("section = %q, want %q", section, want)
		}
		if got := spliceSection(before, section, after); got != content {
			t.Errorf("splice roundtrip = %q, want %q", got, content)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, _, _, found, err := extractChangelogSection("# CHANGELOG\n\n## v1.0.0\n", "v2.0.0")
		if err != nil {
			t.Fatalf("extractChangelogSection() error = %v", err)
		}
		if found {
			t.Fatal("expected found = false")
		}
	})

	t.Run("ignores heading inside body marker range", func(t *testing.T) {
		content := "## v1.0.0\n\n### changed\n\n#### Details\n\n<!-- rellog:body:start -->\n## v1.0.0\n<!-- rellog:body:end -->\n"
		_, section, after, found, err := extractChangelogSection(content, "v1.0.0")
		if err != nil {
			t.Fatalf("extractChangelogSection() error = %v", err)
		}
		if !found {
			t.Fatal("expected found = true")
		}
		if after != "" || section != content {
			t.Errorf("section = %q, after = %q; want whole content treated as one section", section, after)
		}
	})

	t.Run("malformed marker range is an error", func(t *testing.T) {
		content := "## v1.0.0\n\n<!-- rellog:body:start -->\nBody\n"
		_, _, _, _, err := extractChangelogSection(content, "v1.0.0")
		if err == nil {
			t.Fatal("expected error for unterminated body marker")
		}
	})
}

func TestParseKindSections(t *testing.T) {
	content := "## v1.0.0\n\n### changed\n\n#### Details\n\nBody\n\n### fixed\n\n#### Details\n\nBody2\n"
	sections, err := parseKindSections(content)
	if err != nil {
		t.Fatalf("parseKindSections() error = %v", err)
	}
	if len(sections) != 2 {
		t.Fatalf("got %d sections, want 2: %#v", len(sections), sections)
	}
	if sections[0].title != "changed" || sections[1].title != "fixed" {
		t.Fatalf("titles = %q, %q; want changed, fixed", sections[0].title, sections[1].title)
	}
	// sections[0].end stops one byte before "### fixed" (excluding the blank
	// line that is the *next* heading's own leading separator, not this
	// section's trailing content).
	if want := strings.Index(content, "### fixed") - 1; sections[0].end != want {
		t.Errorf("sections[0].end = %d, want %d", sections[0].end, want)
	}
	if sections[1].end != len(content) {
		t.Errorf("sections[1].end = %d, want %d (end of content)", sections[1].end, len(content))
	}

	t.Run("malformed marker range is an error", func(t *testing.T) {
		_, err := parseKindSections("## v1.0.0\n\n<!-- rellog:body:end -->\n")
		if err == nil {
			t.Fatal("expected error for unmatched body marker end")
		}
	})
}

func TestApplyKindInsertions(t *testing.T) {
	cfg := entryValidationConfig{}

	t.Run("insert into existing kind section matches full regenerate", func(t *testing.T) {
		content := renderReleaseNote("v1.0.0", []entry{{Kind: "changed", Body: "First"}}, cfg)
		plan := []kindInsertion{{title: "changed", entries: []entry{{Kind: "changed", Body: "Second"}}}}
		got, err := applyKindInsertions(content, plan)
		if err != nil {
			t.Fatalf("applyKindInsertions() error = %v", err)
		}
		want := renderReleaseNote("v1.0.0", []entry{
			{Kind: "changed", Body: "First"},
			{Kind: "changed", Body: "Second"},
		}, cfg)
		if got != want {
			t.Errorf("got:\n%q\nwant:\n%q", got, want)
		}
	})

	t.Run("append new kind section matches full regenerate", func(t *testing.T) {
		content := renderReleaseNote("v1.0.0", []entry{{Kind: "changed", Body: "First"}}, cfg)
		plan := []kindInsertion{{title: "fixed", entries: []entry{{Kind: "fixed", Body: "Bug"}}}}
		got, err := applyKindInsertions(content, plan)
		if err != nil {
			t.Fatalf("applyKindInsertions() error = %v", err)
		}
		want := renderReleaseNote("v1.0.0", []entry{
			{Kind: "changed", Body: "First"},
			{Kind: "fixed", Body: "Bug"},
		}, cfg)
		if got != want {
			t.Errorf("got:\n%q\nwant:\n%q", got, want)
		}
	})

	t.Run("malformed marker range is an error", func(t *testing.T) {
		_, err := applyKindInsertions("## v1.0.0\n\n<!-- rellog:body:start -->\n", nil)
		if err == nil {
			t.Fatal("expected error for unterminated body marker")
		}
	})
}

func TestKindTitle(t *testing.T) {
	cfg := entryValidationConfig{kindTitles: map[string]string{"fixed": "Bug Fixes"}}
	if got := kindTitle("fixed", cfg); got != "Bug Fixes" {
		t.Errorf("kindTitle(fixed) = %q, want %q", got, "Bug Fixes")
	}
	if got := kindTitle("changed", cfg); got != "changed" {
		t.Errorf("kindTitle(changed) = %q, want %q (default to id)", got, "changed")
	}
}

func TestReadEntryValidationConfigKindTitle(t *testing.T) {
	withTempWorkingDir(t)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Fatal(err)
	}
	config := "rellog config-version=1 {\n" +
		"  paths {\n" +
		"    changelog \"CHANGELOG.md\"\n" +
		"    entries \".rellog/entries\"\n" +
		"    release-notes \".rellog/release-notes\"\n" +
		"  }\n" +
		"  entries {\n" +
		"    target-policy \"allow-unknown\"\n" +
		"    kinds {\n" +
		"      kind \"fixed\" title=\"  Bug Fixes  \"\n" +
		"      kind \"changed\"\n" +
		"    }\n" +
		"  }\n" +
		"}\n"
	if err := os.WriteFile(configFile(), []byte(config), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := readEntryValidationConfig()
	if err != nil {
		t.Fatalf("readEntryValidationConfig() error = %v", err)
	}
	// The surrounding whitespace in the KDL value must not leak into the
	// stored title, or it would leak into rendered "### <title>" headings.
	if got, want := cfg.kindTitles["fixed"], "Bug Fixes"; got != want {
		t.Errorf("kindTitles[fixed] = %q, want %q", got, want)
	}
	if _, ok := cfg.kindTitles["changed"]; ok {
		t.Errorf("kindTitles[changed] should be absent (no title configured), got %q", cfg.kindTitles["changed"])
	}
}

func TestCheckMarkersBalanced(t *testing.T) {
	if err := checkMarkersBalanced("no markers here\n"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := checkMarkersBalanced(bodyMarkerStart + "\nBody\n" + bodyMarkerEnd + "\n"); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if err := checkMarkersBalanced(bodyMarkerStart + "\nBody\n"); err == nil {
		t.Error("expected error for unterminated marker")
	}
	if err := checkMarkersBalanced(bodyMarkerEnd + "\n"); err == nil {
		t.Error("expected error for unmatched end marker")
	}
}

func TestIsEmptyReleaseContent(t *testing.T) {
	content := "## v1.0.0\n\nNo changelog-worthy changes.\n"
	if !isEmptyReleaseContent(content, "v1.0.0") {
		t.Error("expected content to be recognized as the empty-release template")
	}
	if isEmptyReleaseContent(content, "v2.0.0") {
		t.Error("expected mismatch for a different release id")
	}
	if isEmptyReleaseContent("## v1.0.0\n\n### changed\n", "v1.0.0") {
		t.Error("expected normal content not to be recognized as empty")
	}
}

func TestRenderAmendReleaseContent(t *testing.T) {
	cfg := entryValidationConfig{}
	t.Run("all empty entries render the fixed template", func(t *testing.T) {
		got := renderAmendReleaseContent("v1.0.0", []entry{{Kind: "empty"}, {Kind: "empty"}}, cfg)
		want := "## v1.0.0\n\nNo changelog-worthy changes.\n"
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
	t.Run("normal entries render via renderReleaseNote", func(t *testing.T) {
		entries := []entry{{Kind: "changed", Body: "Body"}}
		got := renderAmendReleaseContent("v1.0.0", entries, cfg)
		want := renderReleaseNote("v1.0.0", entries, cfg)
		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestBuildKindInsertionPlan(t *testing.T) {
	entries := []entryFile{
		{name: "a", e: entry{Kind: "fixed", Body: "F1"}},
		{name: "b", e: entry{Kind: "changed", Body: "C1"}},
		{name: "c", e: entry{Kind: "fixed", Body: "F2"}},
		{name: "d", e: entry{Kind: "empty"}},
	}
	plan := buildKindInsertionPlan(entries, entryValidationConfig{})
	if len(plan) != 2 {
		t.Fatalf("got %d kinds, want 2: %#v", len(plan), plan)
	}
	if plan[0].title != "fixed" || plan[1].title != "changed" {
		t.Fatalf("kind order = %q, %q; want fixed, changed (first-seen order)", plan[0].title, plan[1].title)
	}
	if len(plan[0].entries) != 2 {
		t.Fatalf("fixed entries = %d, want 2", len(plan[0].entries))
	}
	if plan[0].entries[0].Body != "F1" || plan[0].entries[1].Body != "F2" {
		t.Fatalf("fixed entries out of filename order: %#v", plan[0].entries)
	}
}

// TestAmendExitCodes locks in the amend exit-code mapping documented in
// docs/commands.md: each failure path must return the specific documented
// exitError.Code, not just a generic non-nil error.
func TestAmendExitCodes(t *testing.T) {
	exitCode := func(t *testing.T, err error) int {
		t.Helper()
		var ee *exitError
		if !errors.As(err, &ee) {
			t.Fatalf("error = %v (%T), want *exitError", err, err)
		}
		return ee.Code
	}

	t.Run("invalid release id", func(t *testing.T) {
		withTempWorkingDir(t)
		if err := initRellog(); err != nil {
			t.Fatal(err)
		}
		err := amendRelease(amendOptions{Version: "../v1.0.0", DryRun: true})
		if got, want := exitCode(t, err), ExitInvalidArgument; got != want {
			t.Errorf("exit code = %d, want %d (ExitInvalidArgument)", got, want)
		}
	})

	t.Run("not initialized", func(t *testing.T) {
		withTempWorkingDir(t)
		err := amendRelease(amendOptions{Version: "v1.0.0", DryRun: true})
		if got, want := exitCode(t, err), ExitNotInitialized; got != want {
			t.Errorf("exit code = %d, want %d (ExitNotInitialized)", got, want)
		}
	})

	t.Run("release note missing", func(t *testing.T) {
		withTempWorkingDir(t)
		if err := initRellog(); err != nil {
			t.Fatal(err)
		}
		err := amendRelease(amendOptions{Version: "v1.0.0", DryRun: true})
		if got, want := exitCode(t, err), ExitReleaseNotFound; got != want {
			t.Errorf("exit code = %d, want %d (ExitReleaseNotFound)", got, want)
		}
	})

	t.Run("empty baseline plus normal pending is a conflict", func(t *testing.T) {
		withTempWorkingDir(t)
		if err := initRellog(); err != nil {
			t.Fatal(err)
		}
		writeAmendFixtureEntry(t, "20260626T000000.000000001Z.json", `{"kind":"empty","targets":[],"links":[],"body":"No changelog-worthy changes."}`)
		if err := prepareRelease(prepareOptions{Version: "v1.0.0"}); err != nil {
			t.Fatal(err)
		}
		writeAmendFixtureEntry(t, "20260627T000000.000000001Z.json", validEntryJSON())
		err := amendRelease(amendOptions{Version: "v1.0.0", DryRun: false})
		if got, want := exitCode(t, err), ExitEntryConflict; got != want {
			t.Errorf("exit code = %d, want %d (ExitEntryConflict)", got, want)
		}
	})

	// Regression: baselineIsEmpty must not be computed as len(consumedEntries)
	// == 1, since a successful empty+empty amend grows the consumed cache
	// past one entry while the release stays empty (see allKindEmpty).
	t.Run("empty baseline still conflicts with normal pending after a prior empty+empty merge", func(t *testing.T) {
		withTempWorkingDir(t)
		if err := initRellog(); err != nil {
			t.Fatal(err)
		}
		writeAmendFixtureEntry(t, "20260626T000000.000000001Z.json", `{"kind":"empty","targets":[],"links":[],"body":"No changelog-worthy changes."}`)
		if err := prepareRelease(prepareOptions{Version: "v1.0.0"}); err != nil {
			t.Fatal(err)
		}
		writeAmendFixtureEntry(t, "20260627T000000.000000001Z.json", `{"kind":"empty","targets":[],"links":[],"body":"No changelog-worthy changes."}`)
		if err := amendRelease(amendOptions{Version: "v1.0.0", DryRun: false}); err != nil {
			t.Fatalf("first empty+empty amend failed: %v", err)
		}
		if n := len(mustReadManifestEntries(t, "v1.0.0")); n != 2 {
			t.Fatalf("consumed manifest has %d entries, want 2 (precondition for this regression)", n)
		}
		writeAmendFixtureEntry(t, "20260628T000000.000000001Z.json", validEntryJSON())
		err := amendRelease(amendOptions{Version: "v1.0.0", DryRun: false})
		if got, want := exitCode(t, err), ExitEntryConflict; got != want {
			t.Errorf("exit code = %d, want %d (ExitEntryConflict); a normal entry must not merge into an empty release", got, want)
		}
	})

	t.Run("malformed body marker", func(t *testing.T) {
		withTempWorkingDir(t)
		if err := initRellog(); err != nil {
			t.Fatal(err)
		}
		writeAmendFixtureEntry(t, "20260626T000000.000000001Z.json", validEntryJSON())
		if err := prepareRelease(prepareOptions{Version: "v1.0.0"}); err != nil {
			t.Fatal(err)
		}
		if err := os.RemoveAll(consumedDir("v1.0.0")); err != nil {
			t.Fatal(err)
		}
		releaseNotePath := filepath.Join(releaseNotesDir(), "v1.0.0.md")
		if err := os.WriteFile(releaseNotePath, []byte("## v1.0.0\n\n"+bodyMarkerStart+"\nBody\n"), 0644); err != nil {
			t.Fatal(err)
		}
		writeAmendFixtureEntry(t, "20260627T000000.000000001Z.json", validEntryJSON())
		err := amendRelease(amendOptions{Version: "v1.0.0", DryRun: false})
		if got, want := exitCode(t, err), ExitCheckFailed; got != want {
			t.Errorf("exit code = %d, want %d (ExitCheckFailed)", got, want)
		}
	})

	t.Run("regenerate mismatch after hand edit", func(t *testing.T) {
		withTempWorkingDir(t)
		if err := initRellog(); err != nil {
			t.Fatal(err)
		}
		writeAmendFixtureEntry(t, "20260626T000000.000000001Z.json", validEntryJSON())
		if err := prepareRelease(prepareOptions{Version: "v1.0.0"}); err != nil {
			t.Fatal(err)
		}
		releaseNotePath := filepath.Join(releaseNotesDir(), "v1.0.0.md")
		data, err := os.ReadFile(releaseNotePath)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(releaseNotePath, append(data, []byte("hand edit\n")...), 0644); err != nil {
			t.Fatal(err)
		}
		writeAmendFixtureEntry(t, "20260627T000000.000000001Z.json", validEntryJSON())
		err = amendRelease(amendOptions{Version: "v1.0.0", DryRun: false})
		if got, want := exitCode(t, err), ExitCheckFailed; got != want {
			t.Errorf("exit code = %d, want %d (ExitCheckFailed)", got, want)
		}
	})
}

func writeConsumeConfig(t *testing.T, onFailCreate string) {
	t.Helper()
	content := "rellog config-version=1 {\n" +
		"  paths {\n" +
		"    changelog \"CHANGELOG.md\"\n" +
		"    entries \".rellog/entries\"\n" +
		"    release-notes \".rellog/release-notes\"\n" +
		"  }\n" +
		"  entries {\n" +
		"    target-policy \"allow-unknown\"\n" +
		"    kinds {\n" +
		"      kind \"changed\"\n" +
		"    }\n" +
		"  }\n" +
		"  consume {\n" +
		"    on-fail-create \"" + onFailCreate + "\"\n" +
		"  }\n" +
		"}\n"
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configFile(), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestPlanConsumedCacheUpdate(t *testing.T) {
	missingEntry := func() []entryFile {
		return []entryFile{{name: "missing.json", path: filepath.Join(entriesDir(), "missing.json")}}
	}

	t.Run("build failure under error policy aborts", func(t *testing.T) {
		withTempWorkingDir(t)
		if err := initRellog(); err != nil {
			t.Fatal(err)
		}
		_, warning, abort := planConsumedCacheUpdate("v1.0.0", missingEntry())
		if abort == nil {
			t.Fatal("expected abort error")
		}
		if warning != nil {
			t.Errorf("expected no warning, got %v", warning)
		}
	})

	t.Run("build failure under warn policy returns warning only", func(t *testing.T) {
		withTempWorkingDir(t)
		if err := initRellog(); err != nil {
			t.Fatal(err)
		}
		writeConsumeConfig(t, "warn")
		_, warning, abort := planConsumedCacheUpdate("v1.0.0", missingEntry())
		if abort != nil {
			t.Fatalf("unexpected abort: %v", abort)
		}
		if warning == nil {
			t.Fatal("expected warning")
		}
	})

	t.Run("build failure under ignore policy is silent", func(t *testing.T) {
		withTempWorkingDir(t)
		if err := initRellog(); err != nil {
			t.Fatal(err)
		}
		writeConsumeConfig(t, "ignore")
		plan, warning, abort := planConsumedCacheUpdate("v1.0.0", missingEntry())
		if abort != nil || warning != nil {
			t.Fatalf("expected no abort/warning, got abort=%v warning=%v", abort, warning)
		}
		if plan.tempDir != "" {
			t.Fatal("expected empty plan on build failure")
		}
	})

	t.Run("successful build returns a committable plan", func(t *testing.T) {
		withTempWorkingDir(t)
		if err := initRellog(); err != nil {
			t.Fatal(err)
		}
		writeAmendFixtureEntry(t, "entry.json", validEntryJSON())
		plan, warning, abort := planConsumedCacheUpdate("v1.0.0", []entryFile{{name: "entry.json", path: filepath.Join(entriesDir(), "entry.json")}})
		if abort != nil || warning != nil {
			t.Fatalf("unexpected abort=%v warning=%v", abort, warning)
		}
		if plan.tempDir == "" {
			t.Fatal("expected non-empty temp dir")
		}
		defer func() { _ = os.RemoveAll(plan.tempDir) }()
	})

	t.Run("preflight failure when a consumed path segment is a file", func(t *testing.T) {
		withTempWorkingDir(t)
		if err := initRellog(); err != nil {
			t.Fatal(err)
		}
		writeAmendFixtureEntry(t, "entry.json", validEntryJSON())
		if err := os.MkdirAll(filepath.Join(baseDir, "consumed"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(baseDir, "consumed", "cli"), []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
		_, warning, abort := planConsumedCacheUpdate("cli/v1.0.0", []entryFile{{name: "entry.json", path: filepath.Join(entriesDir(), "entry.json")}})
		if abort == nil {
			t.Fatal("expected abort error")
		}
		if warning != nil {
			t.Errorf("expected no warning, got %v", warning)
		}
	})
}

func TestCommitConsumedCacheUpdate(t *testing.T) {
	t.Run("empty plan is a no-op", func(t *testing.T) {
		withTempWorkingDir(t)
		warning, abort := commitConsumedCacheUpdate("v1.0.0", consumedCachePlan{})
		if warning != nil || abort != nil {
			t.Fatalf("expected no-op, got warning=%v abort=%v", warning, abort)
		}
	})

	t.Run("successful commit renames temp dir into place", func(t *testing.T) {
		withTempWorkingDir(t)
		if err := initRellog(); err != nil {
			t.Fatal(err)
		}
		writeAmendFixtureEntry(t, "entry.json", validEntryJSON())
		plan, _, abort := planConsumedCacheUpdate("v1.0.0", []entryFile{{name: "entry.json", path: filepath.Join(entriesDir(), "entry.json")}})
		if abort != nil {
			t.Fatal(abort)
		}
		warning, abort2 := commitConsumedCacheUpdate("v1.0.0", plan)
		if warning != nil || abort2 != nil {
			t.Fatalf("unexpected warning=%v abort=%v", warning, abort2)
		}
		if _, err := os.Stat(consumedDir("v1.0.0")); err != nil {
			t.Fatalf("expected committed dir: %v", err)
		}
	})

	t.Run("commit failure applies the policy", func(t *testing.T) {
		for _, tc := range []struct {
			policy    string
			wantWarn  bool
			wantAbort bool
		}{
			{"error", false, true},
			{"warn", true, false},
			{"ignore", false, false},
		} {
			t.Run(tc.policy, func(t *testing.T) {
				withTempWorkingDir(t)
				if err := os.MkdirAll(filepath.Join(baseDir, "consumed"), 0755); err != nil {
					t.Fatal(err)
				}
				// Make the parent directory component a file so MkdirAll(Dir(finalDir))
				// fails during commit, forcing a deterministic commit failure.
				if err := os.WriteFile(filepath.Join(baseDir, "consumed", "cli"), []byte("x"), 0644); err != nil {
					t.Fatal(err)
				}
				plan := consumedCachePlan{tempDir: t.TempDir(), policy: tc.policy}
				warning, abort := commitConsumedCacheUpdate("cli/v1.0.0", plan)
				if tc.wantAbort && abort == nil {
					t.Fatal("expected abort")
				}
				if !tc.wantAbort && abort != nil {
					t.Fatalf("unexpected abort: %v", abort)
				}
				if tc.wantWarn && warning == nil {
					t.Fatal("expected warning")
				}
				if !tc.wantWarn && warning != nil {
					t.Fatalf("unexpected warning: %v", warning)
				}
			})
		}
	})

	t.Run("failed rename-in restores the pre-existing cache instead of leaving it absent", func(t *testing.T) {
		withTempWorkingDir(t)
		finalDir := consumedDir("v1.0.0")
		if err := os.MkdirAll(filepath.Join(finalDir, "entries"), 0755); err != nil {
			t.Fatal(err)
		}
		oldManifest := []byte(`{"schemaVersion":1,"releaseId":"v1.0.0","entries":[]}`)
		if err := os.WriteFile(filepath.Join(finalDir, "manifest.json"), oldManifest, 0644); err != nil {
			t.Fatal(err)
		}

		// A tempDir that does not exist makes the final os.Rename fail after the
		// old cache has already been renamed aside, exercising the restore path.
		plan := consumedCachePlan{tempDir: filepath.Join(t.TempDir(), "does-not-exist"), policy: "error"}
		_, abort := commitConsumedCacheUpdate("v1.0.0", plan)
		if abort == nil {
			t.Fatal("expected abort error from a missing temp dir")
		}

		got, err := os.ReadFile(filepath.Join(finalDir, "manifest.json"))
		if err != nil {
			t.Fatalf("consumed cache was not restored after failed commit: %v", err)
		}
		if string(got) != string(oldManifest) {
			t.Errorf("restored manifest = %q, want original %q", got, oldManifest)
		}
		if _, err := os.Stat(finalDir + ".amend-bak"); !os.IsNotExist(err) {
			t.Errorf("backup dir should have been restored and removed, stat err = %v", err)
		}
	})
}

func mustReadManifestEntries(t *testing.T, releaseID string) []consumedManifestEntry {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(consumedDir(releaseID), "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest consumedManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	return manifest.Entries
}

func writeAmendFixtureEntry(t *testing.T, filename, json string) {
	t.Helper()
	if err := os.MkdirAll(entriesDir(), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(entriesDir(), filename), []byte(json), 0644); err != nil {
		t.Fatal(err)
	}
}

func withTempWorkingDir(t *testing.T) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(wd); err != nil {
			t.Fatal(err)
		}
	})
}
