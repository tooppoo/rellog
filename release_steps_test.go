package rellog

import (
	"slices"
	"testing"
)

func TestBuiltinRustPresetDefinesEveryReleasePhase(t *testing.T) {
	got := builtinPresets.resolve([]string{"rust"}, preserveConfig{}, readyConfig{})

	assertReleaseStepsEqual(t, "PreservePreview", got.PreservePreview, []releaseStep{
		{Kind: releaseStepKindBuiltin, Value: "rust-preview"},
	})
	assertReleaseStepsEqual(t, "PreserveRun", got.PreserveRun, []releaseStep{
		{Kind: releaseStepKindBuiltin, Value: "rust-run"},
	})
	assertReleaseStepsEqual(t, "ReadyVerify", got.ReadyVerify, []releaseStep{
		{Kind: releaseStepKindBuiltin, Value: "rust-verify"},
	})
}

func TestPresetRegistryResolvePreservesPresetDeclarationOrder(t *testing.T) {
	presets := newPresetRegistry(
		testPreset{
			id: "first",
			steps: releasePhases{
				PreservePreview: []releaseStep{
					{Kind: releaseStepKindBuiltin, Value: "first-preview-a"},
					{Kind: releaseStepKindBuiltin, Value: "first-preview-b"},
				},
			},
		},
		testPreset{
			id: "second",
			steps: releasePhases{
				PreservePreview: []releaseStep{{Kind: releaseStepKindBuiltin, Value: "second-preview"}},
			},
		},
	)

	got := presets.resolve([]string{"second", "first"}, preserveConfig{}, readyConfig{})

	assertReleaseStepsEqual(t, "PreservePreview", got.PreservePreview, []releaseStep{
		{Kind: releaseStepKindBuiltin, Value: "second-preview"},
		{Kind: releaseStepKindBuiltin, Value: "first-preview-a"},
		{Kind: releaseStepKindBuiltin, Value: "first-preview-b"},
	})
}

func TestPresetRegistryResolveReplacesOnlyPhasesWithExplicitScripts(t *testing.T) {
	tests := []struct {
		name     string
		preserve preserveConfig
		ready    readyConfig
		want     releasePhases
	}{
		{
			name: "preserve preview",
			preserve: preserveConfig{
				PreviewScripts: []string{"scripts/preview-first.sh", "scripts/preview-second.sh"},
			},
			want: releasePhases{
				PreservePreview: []releaseStep{
					{Kind: releaseStepKindScript, Value: "scripts/preview-first.sh"},
					{Kind: releaseStepKindScript, Value: "scripts/preview-second.sh"},
				},
				PreserveRun: []releaseStep{{Kind: releaseStepKindBuiltin, Value: "rust-run"}},
				ReadyVerify: []releaseStep{{Kind: releaseStepKindBuiltin, Value: "rust-verify"}},
			},
		},
		{
			name: "preserve run",
			preserve: preserveConfig{
				RunScripts: []string{"scripts/run.sh"},
			},
			want: releasePhases{
				PreservePreview: []releaseStep{{Kind: releaseStepKindBuiltin, Value: "rust-preview"}},
				PreserveRun:     []releaseStep{{Kind: releaseStepKindScript, Value: "scripts/run.sh"}},
				ReadyVerify:     []releaseStep{{Kind: releaseStepKindBuiltin, Value: "rust-verify"}},
			},
		},
		{
			name:  "ready verify",
			ready: readyConfig{VerifyScripts: []string{"scripts/verify.sh"}},
			want: releasePhases{
				PreservePreview: []releaseStep{{Kind: releaseStepKindBuiltin, Value: "rust-preview"}},
				PreserveRun:     []releaseStep{{Kind: releaseStepKindBuiltin, Value: "rust-run"}},
				ReadyVerify:     []releaseStep{{Kind: releaseStepKindScript, Value: "scripts/verify.sh"}},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := builtinPresets.resolve([]string{"rust"}, test.preserve, test.ready)

			assertReleaseStepsEqual(t, "PreservePreview", got.PreservePreview, test.want.PreservePreview)
			assertReleaseStepsEqual(t, "PreserveRun", got.PreserveRun, test.want.PreserveRun)
			assertReleaseStepsEqual(t, "ReadyVerify", got.ReadyVerify, test.want.ReadyVerify)
		})
	}
}

func TestPresetRegistryResolveUsesScriptsWithoutSelectedPresets(t *testing.T) {
	preserve := preserveConfig{
		RunScripts: []string{"scripts/run.sh"},
	}

	got := builtinPresets.resolve(nil, preserve, readyConfig{})

	assertReleaseStepsEqual(t, "PreservePreview", got.PreservePreview, nil)
	assertReleaseStepsEqual(t, "PreserveRun", got.PreserveRun, []releaseStep{
		{Kind: releaseStepKindScript, Value: "scripts/run.sh"},
	})
	assertReleaseStepsEqual(t, "ReadyVerify", got.ReadyVerify, nil)
}

func assertReleaseStepsEqual(t *testing.T, field string, got, want []releaseStep) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("%s = %#v, want %#v", field, got, want)
	}
}

type testPreset struct {
	id    string
	steps releasePhases
}

func (preset testPreset) ID() string {
	return preset.id
}

func (preset testPreset) ReleaseSteps() releasePhases {
	return preset.steps
}
