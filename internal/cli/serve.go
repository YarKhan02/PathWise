package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/YarKhan02/PathWise/internal/project"
	"github.com/YarKhan02/PathWise/internal/server"
	"github.com/YarKhan02/PathWise/internal/storage/sqlite"
)

func newServeCommand(g *globalFlags) *cobra.Command {
	var addr string

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve the interactive authorization-graph dashboard on localhost",
		Long: `Serve hosts an interactive dashboard for the latest scan: API endpoints on
one side, database tables on the other, with the edges between them. Select an
endpoint to trace the tables it reads, or a table to see which endpoints reach
it, and filter by the role each endpoint demands.

It reads the stored scan — run ` + "`pathwise scan`" + ` first — and serves a
self-contained page embedded in the binary, so there is nothing to build and no
network dependency.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runServe(cmd, g, addr)
		},
	}
	cmd.Flags().StringVar(&addr, "addr", "127.0.0.1:7777", "listen address for the dashboard")
	return cmd
}

func runServe(cmd *cobra.Command, g *globalFlags, addr string) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	proj, err := project.Resolve(g.projectDir)
	if err != nil {
		return err
	}
	store, err := sqlite.Open(ctx, g.resolveStorePath(proj.Root))
	if err != nil {
		return err
	}
	defer store.Close()

	stored, err := store.UpsertProject(ctx, filepath.Base(proj.Root), proj.Root, proj.ModulePath)
	if err != nil {
		return err
	}
	scan, err := store.LatestScan(ctx, stored.ID)
	if err != nil {
		return err
	}
	if scan == nil {
		return fmt.Errorf("no scan found for %s — run `pathwise scan` first", stored.Name)
	}

	rep, err := loadReport(ctx, store, scan, stored.Name)
	if err != nil {
		return err
	}
	data := server.BuildScan(rep)

	// Attach the latest runtime verification, if `pathwise test` recorded one.
	// It lives in its own tables and is projected onto the scan here, never
	// merged into the static findings.
	runs, err := store.LatestTestResults(ctx, scan.ID)
	if err != nil {
		return err
	}
	for _, r := range runs {
		data.RuntimeTests = append(data.RuntimeTests, server.RuntimeTest{
			Method: r.Method, Path: r.Path, Endpoint: r.Endpoint(), Role: r.IdentityRole,
			Expectation: r.Expectation, Outcome: r.Outcome, Status: r.StatusCode, Detail: r.Detail,
		})
		data.RuntimeTestedAt = r.ExecutedAt
	}

	out := cmd.OutOrStdout()
	return server.Serve(ctx, addr, data, func(url string) {
		fmt.Fprintf(out, "PathWise dashboard — %s\n", stored.Name)
		fmt.Fprintf(out, "  %d endpoints · %d roles · %d tables · %d findings\n",
			data.Counts.Endpoints, data.Counts.Roles, data.Counts.Tables, data.Counts.Findings)
		fmt.Fprintf(out, "  open %s  (Ctrl-C to stop)\n", url)
	})
}
