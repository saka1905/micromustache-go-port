package benchmarking

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"unicode/utf8"

	mm "github.com/saka1905/micromustache-go-port"
)

func LoadSuite(path string) (Suite, []byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Suite{}, nil, err
	}
	var suite Suite
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&suite); err != nil {
		return Suite{}, nil, err
	}
	if suite.SchemaVersion != SchemaVersion {
		return Suite{}, nil, fmt.Errorf("unsupported workload schema %d", suite.SchemaVersion)
	}
	if err := ValidateSuite(suite); err != nil {
		return Suite{}, nil, err
	}
	return suite, data, nil
}

func ValidateSuite(suite Suite) error {
	seen := map[string]struct{}{}
	covered := map[string]int{}
	for index, workload := range suite.Workloads {
		if workload.ID == "" || workload.API == "" || workload.Category == "" || workload.Size == "" {
			return fmt.Errorf("workload %d requires id, api, category, and size", index)
		}
		if _, exists := seen[workload.ID]; exists {
			return fmt.Errorf("duplicate workload id %q", workload.ID)
		}
		seen[workload.ID] = struct{}{}
		covered[workload.API]++
		if workload.Size != "small" && workload.Size != "medium" && workload.Size != "large" {
			return fmt.Errorf("workload %q has invalid size %q", workload.ID, workload.Size)
		}
		if workload.TimedSetup != "included" && workload.TimedSetup != "excluded" {
			return fmt.Errorf("workload %q has invalid timedSetup %q", workload.ID, workload.TimedSetup)
		}
		if workload.Expected.Mode != "success" || workload.Expected.ResolverCalls < 0 {
			return fmt.Errorf("workload %q must describe a successful result", workload.ID)
		}
		actual, err := CalculateMetrics(workload)
		if err != nil {
			return fmt.Errorf("workload %q: %w", workload.ID, err)
		}
		if actual != workload.Metrics {
			return fmt.Errorf("workload %q metrics mismatch: declared=%+v actual=%+v", workload.ID, workload.Metrics, actual)
		}
		if err := validateWorkloadShape(workload); err != nil {
			return fmt.Errorf("workload %q: %w", workload.ID, err)
		}
	}
	for _, api := range RequiredAPIs {
		if covered[api] == 0 {
			return fmt.Errorf("missing required API workload %q", api)
		}
	}
	return nil
}

func validateWorkloadShape(workload Workload) error {
	switch workload.API {
	case "tokenize", "render", "renderFn", "renderFnAsync", "compile", "renderer.render", "renderer.renderFn", "renderer.renderFnAsync", "renderer.construct":
		if workload.Template == "" {
			return fmt.Errorf("template is required")
		}
	case "get":
		if workload.Path == "" {
			return fmt.Errorf("path is required")
		}
	case "getRef":
		if len(workload.Ref) == 0 {
			return fmt.Errorf("ref is required")
		}
	default:
		return fmt.Errorf("unsupported API %q", workload.API)
	}
	if (workload.API == "renderFn" || workload.API == "renderFnAsync" || workload.API == "renderer.renderFn" || workload.API == "renderer.renderFnAsync") && len(workload.Resolver) == 0 {
		return fmt.Errorf("resolver table is required")
	}
	return nil
}

func CalculateMetrics(workload Workload) (WorkloadMetrics, error) {
	metrics := WorkloadMetrics{
		TemplateBytes:      len([]byte(workload.Template)),
		TemplateCharacters: utf8.RuneCountInString(workload.Template),
		DataNodes:          countDataNodes(workload.Data),
	}
	for _, variant := range workload.DataVariants {
		if nodes := countDataNodes(variant); nodes > metrics.DataNodes {
			metrics.DataNodes = nodes
		}
	}
	if workload.Template != "" {
		tokens, err := mm.Tokenize(workload.Template, mm.TokenizeOptions{MaxPathLen: workload.Options.MaxPathLen})
		if err != nil {
			return WorkloadMetrics{}, err
		}
		metrics.Interpolations = len(tokens.Paths)
		metrics.PathCount = len(tokens.Paths)
	} else if workload.API == "get" || workload.API == "getRef" {
		metrics.PathCount = 1
	}
	return metrics, nil
}

func countDataNodes(value any) int {
	switch current := value.(type) {
	case nil:
		return 0
	case map[string]any:
		if current == nil {
			return 0
		}
		count := 1
		for _, child := range current {
			count += countDataNodes(child)
		}
		return count
	case []any:
		count := 1
		for _, child := range current {
			count += countDataNodes(child)
		}
		return count
	default:
		return 1
	}
}

func SHA256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func ValidateConfig(config RunnerConfig) error {
	if config.Warmup < 3 || config.Samples < 7 || config.MinDurationMS <= 0 || config.MaxIterations <= 0 || config.ProcessTimeoutSeconds <= 0 {
		return fmt.Errorf("invalid benchmark config: %+v", config)
	}
	return nil
}

func ValidateSample(sample Sample) error {
	if sample.Iterations <= 0 || sample.ElapsedNS <= 0 || sample.NSPerOp <= 0 || math.IsNaN(sample.NSPerOp) || math.IsInf(sample.NSPerOp, 0) {
		return fmt.Errorf("invalid sample: %+v", sample)
	}
	return nil
}

func SortedWorkloads(suite Suite) []Workload {
	result := append([]Workload(nil), suite.Workloads...)
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}
