package rellog

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func cmdCheck() *cobra.Command {
	return &cobra.Command{
		Use:          "check",
		Short:        "Validate unreleased entries",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			results, totalMd, err := checkRepository()
			if err != nil {
				return err
			}
			return reportCheckResults(results, totalMd)
		},
	}
}

func reportCheckResults(results []fileCheckResult, totalMd int) error {
	if len(results) == 0 {
		fmt.Printf("rellog check: OK (entries: %d)\n", totalMd)
		return nil
	}

	totalErrs := 0
	totalWarnings := 0
	for _, r := range results {
		for _, ce := range r.Errors {
			if isCheckWarning(ce) {
				totalWarnings++
			} else {
				totalErrs++
			}
		}
	}
	if totalErrs == 0 {
		fmt.Printf("rellog check: OK (entries: %d)\n", totalMd)
		fmt.Fprintf(os.Stderr, "rellog check: WARNING\n\n%d files\n%d warnings\n\n", len(results), totalWarnings)
		printCheckDiagnostics(results)
		return nil
	}

	if totalWarnings == 0 {
		fmt.Fprintf(os.Stderr, "rellog check: FAILED\n\n%d files\n%d errors\n\n", len(results), totalErrs)
	} else {
		fmt.Fprintf(os.Stderr, "rellog check: FAILED\n\n%d files\n%d errors\n%d warnings\n\n", len(results), totalErrs, totalWarnings)
	}
	printCheckDiagnostics(results)

	return &exitError{ExitCheckFailed, ""}
}

func isCheckWarning(ce checkError) bool {
	return strings.HasPrefix(ce.Code, "warning[")
}

func printCheckDiagnostics(results []fileCheckResult) {
	for i, r := range results {
		fmt.Fprintf(os.Stderr, "%s\n", r.Path)
		for j, ce := range r.Errors {
			fmt.Fprintf(os.Stderr, "  %s\n", ce.Code)
			for _, msgLine := range strings.Split(ce.Message, "\n") {
				if msgLine == "" {
					fmt.Fprintln(os.Stderr)
				} else {
					fmt.Fprintf(os.Stderr, "    %s\n", msgLine)
				}
			}
			if i < len(results)-1 || j < len(r.Errors)-1 {
				fmt.Fprintln(os.Stderr)
			}
		}
	}
}
