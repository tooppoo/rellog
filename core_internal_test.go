package rellog

import (
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
