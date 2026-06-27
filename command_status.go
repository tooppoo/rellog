package rellog

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

func cmdStatus() *cobra.Command {
	return &cobra.Command{
		Use:          "status",
		Short:        "Show unreleased entries",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			entries, err := readEntries()
			if err != nil {
				return err
			}

			hasEmpty := false
			hasNormal := false
			for _, e := range entries {
				if e.Kind == "empty" {
					hasEmpty = true
				} else {
					hasNormal = true
				}
			}

			state := "normal"
			if hasEmpty && !hasNormal {
				state = "empty"
			}

			prepareAllowed := "yes"
			if hasEmpty && hasNormal {
				prepareAllowed = "no"
			}

			fmt.Printf("Unreleased entries: %d\n", len(entries))
			fmt.Printf("State: %s\n", state)
			fmt.Printf("Prepare allowed: %s\n", prepareAllowed)
			fmt.Println()

			if state == "empty" {
				fmt.Println("No changelog-worthy changes.")
				return nil
			}

			var kindOrder []string
			kindEntries := map[string][]entry{}
			for _, e := range entries {
				if e.Kind == "empty" {
					continue
				}
				if _, seen := kindEntries[e.Kind]; !seen {
					kindOrder = append(kindOrder, e.Kind)
				}
				kindEntries[e.Kind] = append(kindEntries[e.Kind], e)
			}

			for i, kind := range kindOrder {
				if i > 0 {
					fmt.Println()
				}
				fmt.Printf("### %s\n\n", kind)
				for _, e := range kindEntries[kind] {
					fmt.Printf("- %s\n", e.Body)
					if len(e.Targets) > 0 {
						fmt.Printf("  targets: %s\n", strings.Join(e.Targets, ", "))
					}
					if len(e.Issues) > 0 {
						fmt.Printf("  issues: %s\n", strings.Join(e.Issues, ", "))
					}
					if len(e.PRs) > 0 {
						fmt.Printf("  prs: %s\n", strings.Join(e.PRs, ", "))
					}
				}
			}

			return nil
		},
	}
}
