// Command micromustache-go-benchmark executes validation-only benchmark workloads against exported Go APIs.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/saka1905/micromustache-go-port/internal/benchmarking"
)

func main() {
	workloads := flag.String("workloads", "", "tracked benchmark workload JSON")
	mode := flag.String("mode", "validate", "validate or benchmark")
	warmup := flag.Int("warmup", 3, "warmup batches per workload")
	samples := flag.Int("samples", 7, "measured samples per workload")
	minimum := flag.Int64("min-duration-ms", 200, "minimum duration of every measured sample")
	maximum := flag.Int64("max-iterations", 16777216, "maximum calibrated iterations per sample")
	processTimeout := flag.Int("process-timeout-seconds", 300, "orchestrator process timeout recorded in output")
	flag.Parse()
	if *workloads == "" {
		fmt.Fprintln(os.Stderr, "workloads is required")
		os.Exit(2)
	}
	suite, data, err := benchmarking.LoadSuite(*workloads)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	output, err := benchmarking.RunGoSuite(suite, data, *mode, benchmarking.RunnerConfig{Warmup: *warmup, Samples: *samples, MinDurationMS: *minimum, MaxIterations: *maximum, ProcessTimeoutSeconds: *processTimeout})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
