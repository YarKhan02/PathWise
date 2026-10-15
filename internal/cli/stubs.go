package cli

import "github.com/spf13/cobra"

func newDiffCommand(g *globalFlags) *cobra.Command {
	var base, head string

	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Compare authorization paths between two scans or revisions",
		Long: `Diff compares the authorization graph of two scans.

Comparison joins on stable identities — normalized method and route,
qualified Go function names, database object names — because graph IDs and
source positions move between revisions even when behavior does not.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return notImplemented(cmd, "phase 7",
				"requires at least two immutable scans with stable identities")
		},
	}

	f := cmd.Flags()
	f.StringVar(&base, "base", "HEAD~1", "base git revision or scan ID")
	f.StringVar(&head, "head", "HEAD", "head git revision or scan ID")
	return cmd
}
