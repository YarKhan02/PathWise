// Command pathwise traces execution and authorization paths through a Go
// application and reports where authorization is missing or unenforced.
package main

import (
	"fmt"
	"os"

	"github.com/YarKhan02/pathwise/internal/cli"
)

func main() {
	if err := cli.NewRootCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "pathwise:", err)
		os.Exit(1)
	}
}
