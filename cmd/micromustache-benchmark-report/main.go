// Command micromustache-benchmark-report validates cross-runtime results and writes benchmark evidence.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/saka1905/micromustache-go-port/internal/benchmarking"
)

func main() {
	mode := flag.String("mode", "validate", "validate or report")
	workloads := flag.String("workloads", "", "tracked workload suite")
	validationNode := flag.String("validation-node", "", "Node validation JSON")
	validationGo := flag.String("validation-go", "", "Go validation JSON")
	round1Node := flag.String("round1-node", "", "round 1 Node benchmark JSON")
	round1Go := flag.String("round1-go", "", "round 1 Go benchmark JSON")
	round2Node := flag.String("round2-node", "", "round 2 Node benchmark JSON")
	round2Go := flag.String("round2-go", "", "round 2 Go benchmark JSON")
	environment := flag.String("environment", "", "sanitized environment JSON")
	repositoryCommit := flag.String("repository-commit", "", "repository base commit")
	repositoryDirty := flag.Bool("repository-dirty", false, "record modified source state")
	jsonReport := flag.String("json-report", "", "machine report path")
	markdownReport := flag.String("markdown-report", "", "human report path")
	flag.Parse()
	if *workloads == "" || *validationNode == "" || *validationGo == "" {
		fatal("workloads, validation-node, and validation-go are required")
	}
	suite, workloadBytes, err := benchmarking.LoadSuite(*workloads)
	if err != nil {
		fatal(err.Error())
	}
	nodeValidation, err := benchmarking.ParseRunnerOutput(*validationNode)
	if err != nil {
		fatal(err.Error())
	}
	goValidation, err := benchmarking.ParseRunnerOutput(*validationGo)
	if err != nil {
		fatal(err.Error())
	}
	if err := benchmarking.ValidateRunnerPair(suite, benchmarking.SHA256Hex(workloadBytes), "validate", nodeValidation.Config, nodeValidation, goValidation); err != nil {
		fatal(err.Error())
	}
	if *mode == "validate" {
		fmt.Printf("PASS benchmark correctness gate workloads=%d\n", len(suite.Workloads))
		return
	}
	if *mode != "report" || *round1Node == "" || *round1Go == "" || *round2Node == "" || *round2Go == "" || *environment == "" || *jsonReport == "" || *markdownReport == "" {
		fatal("report mode requires four round files, environment, and report paths")
	}
	read := func(path string) benchmarking.RunnerOutput {
		output, err := benchmarking.ParseRunnerOutput(path)
		if err != nil {
			fatal(err.Error())
		}
		return output
	}
	var environmentEvidence benchmarking.EnvironmentEvidence
	environmentBytes, err := os.ReadFile(*environment)
	if err != nil {
		fatal(err.Error())
	}
	if err := json.Unmarshal(environmentBytes, &environmentEvidence); err != nil {
		fatal(err.Error())
	}
	report, err := benchmarking.BuildReport(benchmarking.ReportInputs{
		Suite: suite, WorkloadBytes: workloadBytes, ValidationNode: nodeValidation, ValidationGo: goValidation,
		Round1Node: read(*round1Node), Round1Go: read(*round1Go), Round2Node: read(*round2Node), Round2Go: read(*round2Go),
		RepositoryCommit: *repositoryCommit, RepositoryDirty: *repositoryDirty, Environment: environmentEvidence,
	})
	if err != nil {
		fatal(err.Error())
	}
	if err := benchmarking.WriteJSONReport(*jsonReport, report); err != nil {
		fatal(err.Error())
	}
	if err := benchmarking.WriteMarkdownReport(*markdownReport, report); err != nil {
		fatal(err.Error())
	}
	fmt.Printf("PASS benchmark report workloads=%d correctness=%s hash=%s\n", len(report.Workloads), report.CorrectnessStatus, report.ContentSHA256)
}

func fatal(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
