package rellog

type rustPreset struct{}

func (rustPreset) ID() string {
	return "rust"
}

func (rustPreset) ReleaseSteps() releasePhases {
	return releasePhases{
		PreservePreview: []releaseStep{{Kind: releaseStepKindBuiltin, Value: "rust-preview"}},
		PreserveRun:     []releaseStep{{Kind: releaseStepKindBuiltin, Value: "rust-run"}},
		ReadyVerify:     []releaseStep{{Kind: releaseStepKindBuiltin, Value: "rust-verify"}},
	}
}
