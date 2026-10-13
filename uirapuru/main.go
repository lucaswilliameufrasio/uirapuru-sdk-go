package uirapuru

import (
	"flag"
	"fmt"
	"os"
)

// Main is the whole program of an infra description: it prints the TOML, or
// with -check FILE verifies that FILE is up to date (exit status 1 if not).
func Main(m Manifest) {
	check := flag.String("check", "", "fail if this file differs from the generated manifest")
	flag.Parse()
	if *check != "" {
		if err := m.Check(*check); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	out, err := m.TOML()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Print(out)
}
