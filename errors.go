package rellog

// Exit codes returned by rellog commands.
const (
	ExitNotInitialized   = 1 // rellog has not been initialized (run rellog init first)
	ExitInvalidStructure = 2 // rellog directory structure is invalid (expected directory is a file)
	ExitCheckFailed      = 3 // rellog check found validation errors
	ExitReleaseNotFound  = 4 // required release-note file does not exist
	ExitEntryConflict    = 5 // entry conflict: empty and normal entries cannot coexist
	ExitNotGitRepo       = 6 // not a git repository
	ExitInvalidArgument  = 7 // invalid argument (e.g. invalid release id)
	ExitReleaseNotReady  = 8 // release exists but is not ready to publish
)

type exitError struct {
	Code int
	Msg  string
}

func (e *exitError) Error() string { return e.Msg }
