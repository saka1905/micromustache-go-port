// Command micromustache-go-oracle exposes the public Go API through the
// validation-only NDJSON protocol used by the fixed Node oracle.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/saka1905/micromustache-go-port/internal/differential"
)

func main() {
	inputFile := flag.String("input-file", "", "read NDJSON requests from this file instead of stdin")
	flag.Parse()
	input, err := differential.OpenOracleInput(*inputFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer input.Close()
	if err := differential.RunGoOracle(input, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
