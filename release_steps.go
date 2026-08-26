package rellog

type releaseStepKind uint8

const (
	releaseStepKindBuiltin releaseStepKind = iota + 1
	releaseStepKindScript
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

type releasePreset interface {
	ID() string
	ReleaseSteps() releasePhases
}

type presetRegistry map[string]releasePreset

var builtinPresets = newPresetRegistry(rustPreset{})

func newPresetRegistry(presets ...releasePreset) presetRegistry {
	registry := make(presetRegistry, len(presets))
	for _, preset := range presets {
		registry[preset.ID()] = preset
	}
	return registry
}

func (presets presetRegistry) resolve(presetIDs []string, preserve preserveConfig, ready readyConfig) releasePhases {
	var phases releasePhases
	for _, presetID := range presetIDs {
		preset, ok := presets[presetID]
		if !ok {
			continue
		}
		presetSteps := preset.ReleaseSteps()
		phases.PreservePreview = append(phases.PreservePreview, presetSteps.PreservePreview...)
		phases.PreserveRun = append(phases.PreserveRun, presetSteps.PreserveRun...)
		phases.ReadyVerify = append(phases.ReadyVerify, presetSteps.ReadyVerify...)
	}

	phases.PreservePreview = overrideReleaseStepsWithScripts(phases.PreservePreview, preserve.PreviewScripts)
	phases.PreserveRun = overrideReleaseStepsWithScripts(phases.PreserveRun, preserve.RunScripts)
	phases.ReadyVerify = overrideReleaseStepsWithScripts(phases.ReadyVerify, ready.VerifyScripts)
	return phases
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
