package rellog

type releaseStepKind uint8

const (
	releaseStepKindBuiltin releaseStepKind = iota + 1
	releaseStepKindScript
)

const (
	builtinStepRustPreview = "rust-preview"
	builtinStepRustRun     = "rust-run"
	builtinStepRustVerify  = "rust-verify"
)

type releaseStep struct {
	Kind  releaseStepKind
	Value string
}

type releasePhases struct {
	PreservePreview []releaseStep
	PreserveRun     []releaseStep
	ReadyVerify     []releaseStep
}

type presetDefinition struct {
	PreservePreview []string
	PreserveRun     []string
	ReadyVerify     []string
}

type presetRegistry map[string]presetDefinition

var builtinPresets = presetRegistry{
	"rust": {
		PreservePreview: []string{builtinStepRustPreview},
		PreserveRun:     []string{builtinStepRustRun},
		ReadyVerify:     []string{builtinStepRustVerify},
	},
}

func (presets presetRegistry) resolve(presetIDs []string, preserve preserveConfig, ready readyConfig) releasePhases {
	var phases releasePhases
	for _, presetID := range presetIDs {
		preset, ok := presets[presetID]
		if !ok {
			continue
		}
		phases.PreservePreview = appendBuiltinReleaseSteps(phases.PreservePreview, preset.PreservePreview)
		phases.PreserveRun = appendBuiltinReleaseSteps(phases.PreserveRun, preset.PreserveRun)
		phases.ReadyVerify = appendBuiltinReleaseSteps(phases.ReadyVerify, preset.ReadyVerify)
	}

	phases.PreservePreview = overrideReleaseStepsWithScripts(phases.PreservePreview, preserve.PreviewScripts)
	phases.PreserveRun = overrideReleaseStepsWithScripts(phases.PreserveRun, preserve.RunScripts)
	phases.ReadyVerify = overrideReleaseStepsWithScripts(phases.ReadyVerify, ready.VerifyScripts)
	return phases
}

func appendBuiltinReleaseSteps(steps []releaseStep, builtinIDs []string) []releaseStep {
	for _, builtinID := range builtinIDs {
		steps = append(steps, releaseStep{Kind: releaseStepKindBuiltin, Value: builtinID})
	}
	return steps
}

func overrideReleaseStepsWithScripts(steps []releaseStep, scriptPaths []string) []releaseStep {
	if len(scriptPaths) == 0 {
		return steps
	}

	scriptSteps := make([]releaseStep, 0, len(scriptPaths))
	for _, scriptPath := range scriptPaths {
		scriptSteps = append(scriptSteps, releaseStep{Kind: releaseStepKindScript, Value: scriptPath})
	}
	return scriptSteps
}
