package rellog

import (
	"io"
	"os"

	"golang.org/x/term"
)

// checkIsTerminal reports whether f is attached to a real terminal.
// It is a package variable so tests can stub terminal detection.
var checkIsTerminal = func(f *os.File) bool {
	return term.IsTerminal(int(f.Fd()))
}

// shouldUseAddForm reports whether the rich TTY form should be used:
// both in and out must be *os.File values attached to a terminal.
func shouldUseAddForm(in io.Reader, out io.Writer) bool {
	inFile, ok := in.(*os.File)
	if !ok {
		return false
	}
	outFile, ok := out.(*os.File)
	if !ok {
		return false
	}
	return checkIsTerminal(inFile) && checkIsTerminal(outFile)
}
