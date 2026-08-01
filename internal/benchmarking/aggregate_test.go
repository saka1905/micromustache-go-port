package benchmarking

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func loadTestSuite(t *testing.T) (Suite, []byte) {
	t.Helper()
	suite, data, err := LoadSuite(filepath.Join("..", "..", "testdata", "benchmark", "workloads.json"))
	if err != nil {
		t.Fatal(err)
	}
	return suite, data
}

func TestSampleStatistics(t *testing.T) {
	if got := Median([]float64{1, 2, 3}); got != 2 {
		t.Fatalf("odd median=%v", got)
	}
	if got := Median([]float64{1, 2, 3, 4}); got != 2.5 {
		t.Fatalf("even median=%v", got)
	}
	if got := NearestRank([]float64{1, 2, 3, 4}, .25); got != 1 {
		t.Fatalf("p25=%v", got)
	}
	if got := NearestRank([]float64{1, 2, 3, 4}, .75); got != 3 {
		t.Fatalf("p75=%v", got)
	}
	samples := []RawSample{{Round: 1, Runtime: "node", Sample: Sample{Iterations: 10, ElapsedNS: 100, NSPerOp: 10}}, {Round: 2, Runtime: "node", Sample: Sample{Iterations: 10, ElapsedNS: 300, NSPerOp: 30}}}
	metrics, err := CalculateMetricsFromSamples(samples)
	if err != nil {
		t.Fatal(err)
	}
	if metrics.MinNSPerOp != 10 || metrics.MedianNSPerOp != 20 || metrics.MaxNSPerOp != 30 || metrics.OpsPerSecond != 50_000_000 || metrics.Rounds != 2 || metrics.Samples != 2 || metrics.TotalIterations != 20 {
		t.Fatalf("metrics=%+v", metrics)
	}
}

func TestInvalidSamplesAreRejected(t *testing.T) {
	invalid := []Sample{{}, {Iterations: 1, ElapsedNS: -1, NSPerOp: -1}, {Iterations: 1, ElapsedNS: 1, NSPerOp: math.NaN()}, {Iterations: 1, ElapsedNS: 1, NSPerOp: math.Inf(1)}}
	for _, sample := range invalid {
		if ValidateSample(sample) == nil {
			t.Fatalf("accepted %+v", sample)
		}
	}
	if _, err := CalculateMetricsFromSamples([]RawSample{{Sample: Sample{Iterations: 1, ElapsedNS: 1, NSPerOp: 1}}}); err == nil {
		t.Fatal("accepted insufficient samples")
	}
}

func TestSuiteRejectsDuplicateAndMissingWorkloads(t *testing.T) {
	suite, _ := loadTestSuite(t)
	duplicate := suite
	duplicate.Workloads = append(append([]Workload(nil), suite.Workloads...), suite.Workloads[0])
	if err := ValidateSuite(duplicate); err == nil {
		t.Fatal("accepted duplicate workload")
	}
	missing := suite
	missing.Workloads = nil
	for _, workload := range suite.Workloads {
		if workload.API != "tokenize" {
			missing.Workloads = append(missing.Workloads, workload)
		}
	}
	if err := ValidateSuite(missing); err == nil {
		t.Fatal("accepted missing required API")
	}
	if _, _, err := LoadSuite(filepath.Join("..", "..", "testdata", "benchmark", "fixtures", "broken-workloads.json")); err == nil {
		t.Fatal("broken fixture was accepted")
	}
}

func makeOutput(suite Suite, sha, runtimeName, mode string, config RunnerConfig) RunnerOutput {
	output := RunnerOutput{SchemaVersion: SchemaVersion, Mode: mode, Runtime: runtimeName, WorkloadSHA256: sha, Config: config}
	for _, workload := range SortedWorkloads(suite) {
		result := RunnerResult{ID: workload.ID, API: workload.API, Validation: Validation{Status: "PASS", API: workload.API, ResultDigest: "same", ResolverCalls: workload.Expected.ResolverCalls}}
		if mode == "benchmark" {
			for range config.Samples {
				result.Samples = append(result.Samples, Sample{Iterations: 1, ElapsedNS: config.MinDurationMS * 1_000_000, NSPerOp: float64(config.MinDurationMS * 1_000_000)})
			}
			result.SinkDigest = "sink"
		}
		output.Results = append(output.Results, result)
	}
	return output
}

func TestRunnerValidationDetectsSetAndCorrectnessFailures(t *testing.T) {
	suite, data := loadTestSuite(t)
	sha, config := SHA256Hex(data), RunnerConfig{Warmup: 3, Samples: 7, MinDurationMS: 200, MaxIterations: 1024, ProcessTimeoutSeconds: 300}
	node, goOutput := makeOutput(suite, sha, "node", "validate", config), makeOutput(suite, sha, "go", "validate", config)
	if err := ValidateRunnerPair(suite, sha, "validate", config, node, goOutput); err != nil {
		t.Fatal(err)
	}
	goOutput.Results[0].Validation.ResultDigest = "different"
	if err := ValidateRunnerPair(suite, sha, "validate", config, node, goOutput); err == nil {
		t.Fatal("accepted result mismatch")
	}
	goOutput = makeOutput(suite, sha, "go", "validate", config)
	goOutput.Results = goOutput.Results[1:]
	if err := ValidateRunnerPair(suite, sha, "validate", config, node, goOutput); err == nil {
		t.Fatal("accepted missing workload")
	}
	goOutput = makeOutput(suite, sha, "go", "validate", config)
	goOutput.Results = append(goOutput.Results, goOutput.Results[0])
	if err := ValidateRunnerPair(suite, sha, "validate", config, node, goOutput); err == nil {
		t.Fatal("accepted duplicate workload")
	}
	goOutput = makeOutput(suite, sha, "go", "validate", config)
	goOutput.Results[0].ID = "unexpected"
	if err := ValidateRunnerPair(suite, sha, "validate", config, node, goOutput); err == nil {
		t.Fatal("accepted unexpected workload")
	}
}

func TestMalformedRunnerJSON(t *testing.T) {
	if _, err := ParseRunnerOutput(filepath.Join("..", "..", "testdata", "benchmark", "fixtures", "malformed-runner.json")); err == nil {
		t.Fatal("malformed JSON was accepted")
	}
}

func TestProcessFailureTimeoutAndStderr(t *testing.T) {
	if ValidateProcess("bad", ProcessResult{ExitCode: 2}) == nil {
		t.Fatal("accepted non-zero process")
	}
	if ValidateProcess("bad", ProcessResult{TimedOut: true}) == nil {
		t.Fatal("accepted timeout")
	}
	if ValidateProcess("bad", ProcessResult{Stderr: []byte("diagnostic")}) == nil {
		t.Fatal("accepted stderr")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	result := ExecuteProcess(ctx, os.Args[0], "-test.run=TestBenchmarkHelperProcess", "--", "sleep")
	if !result.TimedOut {
		t.Fatalf("timeout result=%+v", result)
	}
	exitResult := ExecuteProcess(context.Background(), os.Args[0], "-test.run=TestBenchmarkHelperProcess", "--", "exit")
	if exitResult.ExitCode != 7 || ValidateProcess("exit", exitResult) == nil {
		t.Fatalf("non-zero result=%+v", exitResult)
	}
	stderrResult := ExecuteProcess(context.Background(), os.Args[0], "-test.run=TestBenchmarkHelperProcess", "--", "stderr")
	if ValidateProcess("stderr", stderrResult) == nil {
		t.Fatalf("stderr result=%+v", stderrResult)
	}
}

func TestBenchmarkHelperProcess(t *testing.T) {
	switch os.Args[len(os.Args)-1] {
	case "sleep":
		time.Sleep(time.Second)
	case "exit":
		os.Exit(7)
	case "stderr":
		fmt.Fprint(os.Stderr, "diagnostic")
	}
}

func TestReportOrderingHashAndMissingRound(t *testing.T) {
	suite, data := loadTestSuite(t)
	sha, config := SHA256Hex(data), RunnerConfig{Warmup: 3, Samples: 7, MinDurationMS: 200, MaxIterations: 1024, ProcessTimeoutSeconds: 300}
	validationNode := makeOutput(suite, sha, "node", "validate", config)
	validationGo := makeOutput(suite, sha, "go", "validate", config)
	roundNode := makeOutput(suite, sha, "node", "benchmark", config)
	roundGo := makeOutput(suite, sha, "go", "benchmark", config)
	inputs := ReportInputs{Suite: suite, WorkloadBytes: data, ValidationNode: validationNode, ValidationGo: validationGo, Round1Node: roundNode, Round1Go: roundGo, Round2Node: roundNode, Round2Go: roundGo, RepositoryCommit: "commit"}
	first, err := BuildReport(inputs)
	if err != nil {
		t.Fatal(err)
	}
	second, err := BuildReport(inputs)
	if err != nil {
		t.Fatal(err)
	}
	if first.ContentSHA256 != second.ContentSHA256 || !reflect.DeepEqual(first.Workloads, second.Workloads) {
		t.Fatal("report is not deterministic")
	}
	for index := 1; index < len(first.Workloads); index++ {
		if first.Workloads[index-1].ID > first.Workloads[index].ID {
			t.Fatal("report not sorted")
		}
	}
	if first.WorkloadSHA256 != sha {
		t.Fatal("workload hash missing")
	}
	if first.Workloads[0].GoMedianOverNodeMedian != 1 {
		t.Fatal("observed median ratio is incorrect")
	}
	inputs.Round2Go = RunnerOutput{}
	if _, err := BuildReport(inputs); err == nil {
		t.Fatal("accepted missing runtime round")
	}
}
