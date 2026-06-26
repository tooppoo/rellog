package rellog

// Exit codes returned by rellog commands.
const (
	ExitNotInitialized   = 1 // rellog has not been initialized (run rellog init first)
	ExitInvalidStructure = 2 // rellog directory structure is invalid (expected directory is a file)
	ExitCheckFailed      = 3 // rellog check found validation errors
	ExitReleaseNotFound  = 4 // required release-note file does not exist
)

type exitError struct {
	Code int
	Msg  string
}

func (e *exitError) Error() string { return e.Msg }
