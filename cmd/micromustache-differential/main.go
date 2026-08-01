// Command micromustache-differential runs the fixed Node reference and the Go
// validation runner against one corpus and writes normalized evidence reports.
package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/saka1905/micromustache-go-port/internal/differential"
)

func main() {
	repositoryRoot := flag.String("repository-root", "", "repository root used as the process working directory")
	corpus := flag.String("corpus", "", "NDJSON differential corpus")
	node := flag.String("node", "node", "Node executable")
	nodeOracle := flag.String("node-oracle", "", "fixed Node oracle script")
	goRunner := flag.String("go-runner", "", "compiled Go validation runner")
	jsonReport := flag.String("json-report", "", "machine-readable report output")
	markdownReport := flag.String("markdown-report", "", "Markdown report output")
	upstreamCommit := flag.String("upstream-commit", "", "fixed upstream commit")
	goCommit := flag.String("go-commit", "", "Go base commit")
	workingTreeModified := flag.Bool("working-tree-modified", false, "record that the validated Go source includes working-tree changes")
	nodeVersion := flag.String("node-version", "", "measured Node version")
	nodeOracleScript := flag.String("node-oracle-script", "oracle/node/oracle.mjs", "repository-relative oracle script recorded in evidence")
	nodeOracleSource := flag.String("node-oracle-source", "oracle/upstream/dist/micromustache.cjs", "repository-relative fixed implementation source recorded in evidence")
	nodeOracleVersion := flag.String("node-oracle-version", "", "fixed upstream package version")
	command := flag.String("command", "scripts/verify-differential.ps1", "documented reproduction command")
	timeout := flag.Duration("timeout", 30*time.Second, "combined process timeout")
	flag.Parse()

	if *repositoryRoot == "" || *corpus == "" || *nodeOracle == "" || *goRunner == "" || *jsonReport == "" || *markdownReport == "" {
		fmt.Fprintln(os.Stderr, "repository-root, corpus, node-oracle, go-runner, json-report, and markdown-report are required")
		os.Exit(2)
	}
	report, err := differential.Run(differential.RunConfig{
		RepositoryRoot: *repositoryRoot, CorpusPath: *corpus, NodeExecutable: *node,
		NodeOraclePath: *nodeOracle, GoRunnerPath: *goRunner, Timeout: *timeout,
		UpstreamCommit: *upstreamCommit, GoCommit: *goCommit, GoWorkingTreeModified: *workingTreeModified,
		NodeVersion: *nodeVersion, NodeOracleScript: *nodeOracleScript, NodeOracleSource: *nodeOracleSource,
		NodeOracleVersion: *nodeOracleVersion, Command: *command,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := differential.WriteJSONReport(*jsonReport, report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := differential.WriteMarkdownReport(*markdownReport, report); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("PASS differential total=%d pass=%d expected_difference=%d skip=%d fail=%d hash=%s\n", report.Counts.Total, report.Counts.Pass, report.Counts.ExpectedDifference, report.Counts.Skip, report.Counts.Fail, report.DeterministicSHA256)
	if report.Counts.Fail != 0 {
		os.Exit(1)
	}
}
