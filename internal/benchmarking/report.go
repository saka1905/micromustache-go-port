package benchmarking

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

type RawSample struct {
	Round       int    `json:"round"`
	Runtime     string `json:"runtime"`
	SampleIndex int    `json:"sampleIndex"`
	Sample
}

type EnvironmentEvidence struct {
	GoVersion         string `json:"goVersion"`
	NodeVersion       string `json:"nodeVersion"`
	OS                string `json:"os"`
	Architecture      string `json:"architecture"`
	CPUModel          string `json:"cpuModel"`
	LogicalProcessors int    `json:"logicalProcessors"`
	InstalledMemory   string `json:"installedMemory"`
	PowerPlan         string `json:"powerPlan"`
}

type CorrectnessEvidence struct {
	Status        string `json:"status"`
	ResultDigest  string `json:"resultDigest"`
	ResolverCalls int    `json:"resolverCalls"`
}

type WorkloadEvidence struct {
	ID                     string              `json:"id"`
	API                    string              `json:"api"`
	Category               string              `json:"category"`
	Size                   string              `json:"size"`
	TimedSetup             string              `json:"timedSetup"`
	Metrics                WorkloadMetrics     `json:"workloadMetrics"`
	Correctness            CorrectnessEvidence `json:"correctness"`
	Node                   Metrics             `json:"node"`
	Go                     Metrics             `json:"go"`
	GoMedianOverNodeMedian float64             `json:"goMedianOverNodeMedian"`
	RawSamples             []RawSample         `json:"rawSamples"`
}

type Report struct {
	SchemaVersion     int                 `json:"schemaVersion"`
	GeneratedAtUTC    string              `json:"generatedAtUtc"`
	GeneratedAtJST    string              `json:"generatedAtJst"`
	RepositoryCommit  string              `json:"repositoryCommit"`
	RepositoryDirty   bool                `json:"repositoryDirty"`
	UpstreamCommit    string              `json:"upstreamCommit"`
	UpstreamVersion   string              `json:"upstreamPackageVersion"`
	WorkloadPath      string              `json:"workloadPath"`
	WorkloadSHA256    string              `json:"workloadSha256"`
	Config            RunnerConfig        `json:"config"`
	Environment       EnvironmentEvidence `json:"environment"`
	RuntimeOrder      []string            `json:"runtimeOrder"`
	Commands          []string            `json:"commands"`
	APICounts         map[string]int      `json:"apiCounts"`
	CorrectnessStatus string              `json:"correctnessStatus"`
	Warnings          []string            `json:"warnings"`
	Workloads         []WorkloadEvidence  `json:"workloads"`
	ContentSHA256     string              `json:"contentSha256"`
}

type ReportInputs struct {
	Suite            Suite
	WorkloadBytes    []byte
	ValidationNode   RunnerOutput
	ValidationGo     RunnerOutput
	Round1Node       RunnerOutput
	Round1Go         RunnerOutput
	Round2Node       RunnerOutput
	Round2Go         RunnerOutput
	RepositoryCommit string
	RepositoryDirty  bool
	Environment      EnvironmentEvidence
}

func BuildReport(input ReportInputs) (Report, error) {
	config := input.ValidationNode.Config
	sha := SHA256Hex(input.WorkloadBytes)
	if err := ValidateRunnerPair(input.Suite, sha, "validate", config, input.ValidationNode, input.ValidationGo); err != nil {
		return Report{}, err
	}
	for round, pair := range []struct{ node, goOutput RunnerOutput }{{input.Round1Node, input.Round1Go}, {input.Round2Node, input.Round2Go}} {
		if err := ValidateRunnerPair(input.Suite, sha, "benchmark", config, pair.node, pair.goOutput); err != nil {
			return Report{}, fmt.Errorf("round %d: %w", round+1, err)
		}
	}
	now := time.Now().UTC()
	report := Report{
		SchemaVersion: SchemaVersion, GeneratedAtUTC: now.Format(time.RFC3339), GeneratedAtJST: now.In(time.FixedZone("JST", 9*60*60)).Format(time.RFC3339),
		RepositoryCommit: input.RepositoryCommit, RepositoryDirty: input.RepositoryDirty,
		UpstreamCommit: "da3420db27b7a2fdfbb768811a1280b34952dc95", UpstreamVersion: "8.0.3",
		WorkloadPath: "testdata/benchmark/workloads.json", WorkloadSHA256: sha, Config: config, Environment: input.Environment,
		RuntimeOrder: []string{"round 1: Node -> Go", "round 2: Go -> Node"},
		Commands: []string{
			"powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/verify-benchmark.ps1",
			"node oracle/node/benchmark.mjs --workloads testdata/benchmark/workloads.json --mode validate|benchmark",
			"micromustache-go-benchmark.exe -workloads testdata/benchmark/workloads.json -mode validate|benchmark",
			"micromustache-benchmark-report.exe -mode validate|report",
		},
		APICounts: map[string]int{}, CorrectnessStatus: "PASS",
		Warnings: []string{
			"Performance values are observations from one environment, not universal guarantees.",
			"Immediate async workloads measure runtime-specific Promise and goroutine overhead, not I/O latency or parallel capacity.",
			"Process startup, module load, workload parsing, setup, calibration, and report generation are outside measured API samples.",
		},
	}
	validation := resultsByID(input.ValidationNode)
	rounds := []struct {
		round int
		node  RunnerOutput
		goOut RunnerOutput
	}{{1, input.Round1Node, input.Round1Go}, {2, input.Round2Node, input.Round2Go}}
	for _, workload := range SortedWorkloads(input.Suite) {
		evidence := WorkloadEvidence{ID: workload.ID, API: workload.API, Category: workload.Category, Size: workload.Size, TimedSetup: workload.TimedSetup, Metrics: workload.Metrics}
		valid := validation[workload.ID].Validation
		evidence.Correctness = CorrectnessEvidence{Status: "PASS", ResultDigest: valid.ResultDigest, ResolverCalls: valid.ResolverCalls}
		for _, round := range rounds {
			for _, runtimeResult := range []struct {
				name   string
				output RunnerOutput
			}{{"node", round.node}, {"go", round.goOut}} {
				result := resultsByID(runtimeResult.output)[workload.ID]
				for index, sample := range result.Samples {
					evidence.RawSamples = append(evidence.RawSamples, RawSample{Round: round.round, Runtime: runtimeResult.name, SampleIndex: index + 1, Sample: sample})
				}
			}
		}
		nodeSamples, goSamples := filterSamples(evidence.RawSamples, "node"), filterSamples(evidence.RawSamples, "go")
		var err error
		evidence.Node, err = CalculateMetricsFromSamples(nodeSamples)
		if err != nil {
			return Report{}, err
		}
		evidence.Go, err = CalculateMetricsFromSamples(goSamples)
		if err != nil {
			return Report{}, err
		}
		evidence.GoMedianOverNodeMedian = evidence.Go.MedianNSPerOp / evidence.Node.MedianNSPerOp
		report.APICounts[workload.API]++
		report.Workloads = append(report.Workloads, evidence)
	}
	content := report
	content.GeneratedAtUTC, content.GeneratedAtJST, content.ContentSHA256 = "", "", ""
	encoded, _ := json.Marshal(content)
	sum := sha256.Sum256(encoded)
	report.ContentSHA256 = hex.EncodeToString(sum[:])
	return report, nil
}

func resultsByID(output RunnerOutput) map[string]RunnerResult {
	result := make(map[string]RunnerResult, len(output.Results))
	for _, current := range output.Results {
		result[current.ID] = current
	}
	return result
}

func filterSamples(samples []RawSample, runtimeName string) []RawSample {
	result := make([]RawSample, 0, len(samples)/2)
	for _, sample := range samples {
		if sample.Runtime == runtimeName {
			result = append(result, sample)
		}
	}
	return result
}

func WriteJSONReport(path string, report Report) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func WriteMarkdownReport(path string, report Report) error {
	var output strings.Builder
	fmt.Fprintf(&output, "# Cross-runtime benchmark baseline\n\n")
	fmt.Fprintf(&output, "This report records a reproducible measurement baseline. It does not set a performance threshold or claim universal superiority.\n\n")
	fmt.Fprintf(&output, "- Generated: `%s` / `%s`\n", report.GeneratedAtJST, report.GeneratedAtUTC)
	fmt.Fprintf(&output, "- Repository base commit: `%s` (working tree modified: `%t`)\n", report.RepositoryCommit, report.RepositoryDirty)
	fmt.Fprintf(&output, "- Fixed upstream: `%s`, package `%s`\n", report.UpstreamCommit, report.UpstreamVersion)
	fmt.Fprintf(&output, "- Workloads: `%s` (`%s`), total `%d`\n", report.WorkloadPath, report.WorkloadSHA256, len(report.Workloads))
	fmt.Fprintf(&output, "- Config: warmup `%d`, samples `%d` per round/runtime/workload, minimum `%d ms`, max iterations `%d`, process timeout `%d s`\n", report.Config.Warmup, report.Config.Samples, report.Config.MinDurationMS, report.Config.MaxIterations, report.Config.ProcessTimeoutSeconds)
	fmt.Fprintf(&output, "- Runtime order: `%s`; `%s`\n", report.RuntimeOrder[0], report.RuntimeOrder[1])
	fmt.Fprintf(&output, "- Environment: Go `%s`, Node `%s`, `%s/%s`, CPU `%s`, logical processors `%d`, memory `%s`, power `%s`\n", report.Environment.GoVersion, report.Environment.NodeVersion, report.Environment.OS, report.Environment.Architecture, report.Environment.CPUModel, report.Environment.LogicalProcessors, report.Environment.InstalledMemory, report.Environment.PowerPlan)
	fmt.Fprintf(&output, "- Correctness gate: `%s`\n", report.CorrectnessStatus)
	fmt.Fprintf(&output, "- Content SHA-256: `%s`\n", report.ContentSHA256)
	fmt.Fprintf(&output, "- Reproduce: `%s`\n\n", report.Commands[0])
	fmt.Fprintf(&output, "## Results\n\nThe ratio is **Go median / Node median**. Values above 1 mean the observed Go median took more time; values below 1 mean it took less time.\n\n")
	fmt.Fprintf(&output, "| Workload | API | Size | Bytes/chars/interpolations/paths/data nodes | Node median ns/op | Go median ns/op | Node ops/s | Go ops/s | Go/Node | Node IQR/median | Go IQR/median |\n")
	fmt.Fprintf(&output, "| --- | --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |\n")
	for _, workload := range report.Workloads {
		m := workload.Metrics
		fmt.Fprintf(&output, "| `%s` | `%s` | %s | %d/%d/%d/%d/%d | %.2f | %.2f | %.0f | %.0f | %.3f | %.3f | %.3f |\n", workload.ID, workload.API, workload.Size, m.TemplateBytes, m.TemplateCharacters, m.Interpolations, m.PathCount, m.DataNodes, workload.Node.MedianNSPerOp, workload.Go.MedianNSPerOp, workload.Node.OpsPerSecond, workload.Go.OpsPerSecond, workload.GoMedianOverNodeMedian, workload.Node.IQRRatio, workload.Go.IQRRatio)
	}
	fmt.Fprintf(&output, "\n## Variability and raw evidence\n\nEach runtime/workload combines two rounds and `%d` raw samples. Min, p25, median, p75, max, iterations, elapsed nanoseconds, and ns/op remain in the JSON report. Percentiles use nearest-rank; median averages the two center values for even counts.\n", report.Config.Samples*2)
	fmt.Fprintf(&output, "\n## Limitations\n\n")
	for _, warning := range report.Warnings {
		fmt.Fprintf(&output, "- %s\n", warning)
	}
	fmt.Fprintf(&output, "- The shared workloads cover successful compatibility cases only; error, cancellation, and known language-boundary differences remain correctness concerns rather than normal-performance baselines.\n")
	return os.WriteFile(path, []byte(output.String()), 0o644)
}

func SortedAPINames(counts map[string]int) []string {
	result := make([]string, 0, len(counts))
	for api := range counts {
		result = append(result, api)
	}
	sort.Strings(result)
	return result
}
