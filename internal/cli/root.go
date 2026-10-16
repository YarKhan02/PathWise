// Package cli defines the pathwise command tree.
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/YarKhan02/PathWise/internal/version"
)

// globalFlags are shared by every subcommand.
type globalFlags struct {
	projectDir string
	storePath  string
	verbose    bool
}

func (g *globalFlags) resolveStorePath(projectRoot string) string {
	if g.storePath == "" {
		return filepath.Join(projectRoot, ".pathwise", "pathwise.db")
	}
	// A directory (trailing slash, or an existing dir) is taken as the store's
	// home and the db file placed inside it. SQLite given a directory fails to
	// open it with a cryptic "out of memory" (CANTOPEN), so handle it here.
	if strings.HasSuffix(g.storePath, string(os.PathSeparator)) {
		return filepath.Join(g.storePath, "pathwise.db")
	}
	if info, err := os.Stat(g.storePath); err == nil && info.IsDir() {
		return filepath.Join(g.storePath, "pathwise.db")
	}
	return g.storePath
}

// NewRootCommand builds the command tree.
func NewRootCommand() *cobra.Command {
	g := &globalFlags{}

	root := &cobra.Command{
		Use:   "pathwise",
		Short: "Trace execution and authorization paths through a Go application",
		Long: `PathWise reconstructs how an API request travels through a Go application,
determines which authorization checks actually affect it, and traces its
reachable PostgreSQL data — citing the source code or HTTP response that
proves each claim.

Where analysis cannot reach a conclusion it says so explicitly, naming what
defeated it. A silent pass is not a clean bill of health.`,
		Version:       version.Analyzer,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	pf := root.PersistentFlags()
	pf.StringVar(&g.projectDir, "project", ".", "path to the Go project to analyze")
	pf.StringVar(&g.storePath, "store", "", "path to PathWise's SQLite database (default <project>/.pathwise/pathwise.db)")
	pf.BoolVarP(&g.verbose, "verbose", "v", false, "verbose diagnostics")

	root.AddCommand(
		newScanCommand(g),
		newReportCommand(g),
		newServeCommand(g),
		newTestCommand(g),
		newDiffCommand(g),
	)
	return root
}

// notImplemented renders a consistent message for commands whose phase has
// not landed, rather than a stub that appears to succeed.
func notImplemented(cmd *cobra.Command, phase, what string) error {
	return fmt.Errorf("%s is not implemented yet (%s): %s", cmd.Name(), phase, what)
}
