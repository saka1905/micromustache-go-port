package benchmarking

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"sort"
	"strings"
)

type ProcessResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
	TimedOut bool
}

func ValidateProcess(name string, result ProcessResult) error {
	if result.TimedOut {
		return fmt.Errorf("%s process timeout", name)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("%s process exited %d", name, result.ExitCode)
	}
	if len(bytes.TrimSpace(result.Stderr)) != 0 {
		return fmt.Errorf("%s process wrote stderr: %s", name, strings.TrimSpace(string(result.Stderr)))
	}
	return nil
}

func ExecuteProcess(ctx context.Context, executable string, args ...string) ProcessResult {
	command := exec.CommandContext(ctx, executable, args...)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	result := ProcessResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}
	if ctx.Err() != nil {
		result.TimedOut, result.ExitCode = true, -1
		return result
	}
	if err == nil {
		return result
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		result.ExitCode = exitError.ExitCode()
	} else {
		result.ExitCode = -1
	}
	return result
}

func ParseRunnerOutput(path string) (RunnerOutput, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return RunnerOutput{}, err
	}
	data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})
	var output RunnerOutput
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&output); err != nil {
		return RunnerOutput{}, err
	}
	return output, nil
}

func ValidateRunnerPair(suite Suite, workloadSHA, mode string, config RunnerConfig, node, goOutput RunnerOutput) error {
	if err := validateRunnerOutput(suite, workloadSHA, mode, "node", config, node); err != nil {
		return fmt.Errorf("Node output: %w", err)
	}
	if err := validateRunnerOutput(suite, workloadSHA, mode, "go", config, goOutput); err != nil {
		return fmt.Errorf("Go output: %w", err)
	}
	goByID := map[string]RunnerResult{}
	for _, result := range goOutput.Results {
		goByID[result.ID] = result
	}
	for _, nodeResult := range node.Results {
		goResult := goByID[nodeResult.ID]
		if nodeResult.API != goResult.API || nodeResult.Validation.API != goResult.Validation.API ||
			nodeResult.Validation.Status != "PASS" || goResult.Validation.Status != "PASS" ||
			nodeResult.Validation.ResultDigest != goResult.Validation.ResultDigest ||
			nodeResult.Validation.ResolverCalls != goResult.Validation.ResolverCalls {
			return fmt.Errorf("correctness mismatch for workload %q", nodeResult.ID)
		}
	}
	return nil
}

func validateRunnerOutput(suite Suite, workloadSHA, mode, runtimeName string, config RunnerConfig, output RunnerOutput) error {
	if output.SchemaVersion != SchemaVersion || output.Mode != mode || output.Runtime != runtimeName || output.WorkloadSHA256 != workloadSHA || output.Config != config {
		return fmt.Errorf("metadata mismatch")
	}
	expected := map[string]Workload{}
	for _, workload := range suite.Workloads {
		expected[workload.ID] = workload
	}
	seen := map[string]struct{}{}
	for _, result := range output.Results {
		workload, exists := expected[result.ID]
		if !exists {
			return fmt.Errorf("unexpected workload %q", result.ID)
		}
		if _, duplicate := seen[result.ID]; duplicate {
			return fmt.Errorf("duplicate workload %q", result.ID)
		}
		seen[result.ID] = struct{}{}
		if result.API != workload.API || result.Validation.API != workload.API || result.Validation.Status != "PASS" || result.Validation.ResultDigest == "" || result.Validation.ResolverCalls != workload.Expected.ResolverCalls {
			return fmt.Errorf("invalid validation result for %q", result.ID)
		}
		if mode == "validate" {
			if len(result.Samples) != 0 || result.SinkDigest != "" {
				return fmt.Errorf("validation output contains timing for %q", result.ID)
			}
		} else {
			if len(result.Samples) != config.Samples || result.SinkDigest == "" {
				return fmt.Errorf("invalid sample count or sink for %q", result.ID)
			}
			for _, sample := range result.Samples {
				if err := ValidateSample(sample); err != nil {
					return fmt.Errorf("workload %q: %w", result.ID, err)
				}
				if sample.ElapsedNS < config.MinDurationMS*1_000_000 {
					return fmt.Errorf("workload %q sample shorter than minimum", result.ID)
				}
			}
		}
	}
	for id := range expected {
		if _, exists := seen[id]; !exists {
			return fmt.Errorf("missing workload %q", id)
		}
	}
	return nil
}

type Metrics struct {
	Rounds          int     `json:"rounds"`
	Samples         int     `json:"samples"`
	TotalIterations int64   `json:"totalIterations"`
	MinNSPerOp      float64 `json:"minNsPerOp"`
	P25NSPerOp      float64 `json:"p25NsPerOp"`
	MedianNSPerOp   float64 `json:"medianNsPerOp"`
	P75NSPerOp      float64 `json:"p75NsPerOp"`
	MaxNSPerOp      float64 `json:"maxNsPerOp"`
	OpsPerSecond    float64 `json:"opsPerSecond"`
	IQRRatio        float64 `json:"iqrRatio"`
}

func CalculateMetricsFromSamples(samples []RawSample) (Metrics, error) {
	if len(samples) < 2 {
		return Metrics{}, fmt.Errorf("at least two samples are required")
	}
	values := make([]float64, len(samples))
	rounds := map[int]struct{}{}
	var total int64
	for index, sample := range samples {
		if err := ValidateSample(sample.Sample); err != nil {
			return Metrics{}, err
		}
		values[index] = sample.NSPerOp
		total += sample.Iterations
		rounds[sample.Round] = struct{}{}
	}
	sort.Float64s(values)
	median := Median(values)
	p25, p75 := NearestRank(values, 0.25), NearestRank(values, 0.75)
	return Metrics{
		Rounds: len(rounds), Samples: len(values), TotalIterations: total,
		MinNSPerOp: values[0], P25NSPerOp: p25, MedianNSPerOp: median, P75NSPerOp: p75, MaxNSPerOp: values[len(values)-1],
		OpsPerSecond: 1_000_000_000 / median, IQRRatio: (p75 - p25) / median,
	}, nil
}

func Median(sorted []float64) float64 {
	if len(sorted)%2 == 1 {
		return sorted[len(sorted)/2]
	}
	middle := len(sorted) / 2
	return (sorted[middle-1] + sorted[middle]) / 2
}

func NearestRank(sorted []float64, percentile float64) float64 {
	index := int(math.Ceil(percentile*float64(len(sorted)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sorted) {
		index = len(sorted) - 1
	}
	return sorted[index]
}
