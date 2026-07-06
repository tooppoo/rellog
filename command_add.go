package rellog

import (
	"bufio"
	"io"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
)

func cmdAdd() *cobra.Command {
	var kind, body, debugDatetime string
	var targets, links []string

	cmd := &cobra.Command{
		Use:          "add",
		Short:        "Add a changelog entry",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if realAddFlagsChanged(cmd) {
				return addEntry(addOptions{
					Kind:          kind,
					Targets:       targets,
					Body:          body,
					Links:         links,
					DebugDatetime: debugDatetime,
				})
			}

			// Interactive mode: rich TTY form when attached to a terminal,
			// 4-line stdin fallback otherwise.
			in, out := cmd.InOrStdin(), cmd.OutOrStdout()
			if shouldUseAddForm(in, out) {
				return runAddForm(in, out, debugDatetime)
			}
			return addEntryInteractive(in, debugDatetime)
		},
	}

	cmd.Flags().StringVar(&kind, "kind", "", "Change kind (e.g. changed, fix)")
	cmd.Flags().StringArrayVar(&targets, "target", nil, "Target component (repeatable)")
	cmd.Flags().StringVar(&body, "body", "", "Change description")
	cmd.Flags().StringArrayVar(&links, "link", nil, "Related URL (repeatable)")
	cmd.Flags().StringVar(&debugDatetime, "debug-datetime", "", "Override entry timestamp for testing")

	return cmd
}

// realAddFlagsChanged reports whether any real add flag was given.
// --debug-datetime alone does not count as a real add flag.
func realAddFlagsChanged(cmd *cobra.Command) bool {
	return cmd.Flags().Changed("kind") ||
		cmd.Flags().Changed("target") ||
		cmd.Flags().Changed("body") ||
		cmd.Flags().Changed("link")
}

// runAddFormProgram runs the Bubble Tea program until the user submits or
// cancels. It is a package variable so tests can stub the terminal loop.
var runAddFormProgram = func(m addFormModel, in io.Reader, out io.Writer) (addFormModel, error) {
	p := tea.NewProgram(m, tea.WithInput(in), tea.WithOutput(out))
	finalModel, err := p.Run()
	if err != nil {
		return addFormModel{}, err
	}
	return finalModel.(addFormModel), nil
}

// runAddForm collects entry values through the rich TTY form, then funnels
// them into the same addEntry path as the non-interactive modes.
func runAddForm(in io.Reader, out io.Writer, debugDatetime string) error {
	// Fail fast before taking over the terminal: reject conditions under
	// which a submit could never succeed. addEntry re-checks these.
	if err := checkEntryStorePreconditions(); err != nil {
		return err
	}
	if err := checkNoEmptyEntryConflict(); err != nil {
		return err
	}

	cfg, err := readEntryValidationConfig()
	if err != nil {
		return err
	}

	finalModel, err := runAddFormProgram(newAddFormModel(cfg), in, out)
	if err != nil {
		return err
	}

	opts, submitted := finalModel.result(debugDatetime)
	if !submitted {
		return nil
	}
	return addEntry(opts)
}

func addEntryInteractive(r io.Reader, debugDatetime string) error {
	scanner := bufio.NewScanner(r)

	readLine := func() string {
		if scanner.Scan() {
			return strings.TrimRight(scanner.Text(), "\r")
		}
		return ""
	}

	kind := readLine()
	targetsLine := readLine()
	body := readLine()
	linksLine := readLine()

	return addEntry(addOptions{
		Kind:          kind,
		Targets:       splitTokens(targetsLine),
		Body:          body,
		Links:         splitTokens(linksLine),
		DebugDatetime: debugDatetime,
	})
}

// splitTokens splits a string on whitespace and commas, returning non-empty tokens.
func splitTokens(s string) []string {
	s = strings.ReplaceAll(s, ",", " ")
	var result []string
	for _, t := range strings.Fields(s) {
		if t != "" {
			result = append(result, t)
		}
	}
	return result
}
