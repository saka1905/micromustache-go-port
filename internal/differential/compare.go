package differential

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"
)

type Classification string

const (
	Pass               Classification = "PASS"
	ExpectedDifference Classification = "EXPECTED_DIFFERENCE"
	Skip               Classification = "SKIP"
	Fail               Classification = "FAIL"
)

var ApprovedDifferences = map[string]string{
	"DIFF-GO-CONTEXT":      "context cancellation and deadlines are Go-only API boundaries",
	"DIFF-GO-UNSUPPORTED":  "unsupported Go values and JavaScript Symbol/stringification failures use different error boundaries",
	"DIFF-GO-ZERO-OPTION":  "zero selects Go defaults while fixed JavaScript validates an explicit numeric zero",
	"DIFF-JS-OWN-TOSTRING": "fixed JavaScript observes an own toString property during coercion",
	"DIFF-JS-PROTOTYPE":    "fixed JavaScript lookup uses the prototype chain while Go uses own map/slice values",
}

type Counts struct {
	Total              int `json:"total"`
	Pass               int `json:"pass"`
	ExpectedDifference int `json:"expectedDifference"`
	Skip               int `json:"skip"`
	Fail               int `json:"fail"`
}

type APICounts struct {
	Total              int `json:"total"`
	Pass               int `json:"pass"`
	ExpectedDifference int `json:"expectedDifference"`
	Skip               int `json:"skip"`
	Fail               int `json:"fail"`
}

type CaseResult struct {
	ID             string         `json:"id"`
	API            string         `json:"api"`
	Category       string         `json:"category,omitempty"`
	Classification Classification `json:"classification"`
	DifferenceID   string         `json:"differenceId,omitempty"`
	Reason         string         `json:"reason,omitempty"`
	Node           string         `json:"node,omitempty"`
	Go             string         `json:"go,omitempty"`
}

type CorpusEvidence struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type Environment struct {
	GoVersion   string `json:"goVersion"`
	NodeVersion string `json:"nodeVersion"`
	OS          string `json:"os"`
}

type NodeOracleEvidence struct {
	Script         string `json:"script"`
	Source         string `json:"source"`
	PackageVersion string `json:"packageVersion"`
}

type Report struct {
	SchemaVersion         int                  `json:"schemaVersion"`
	GeneratedAtUTC        string               `json:"generatedAtUtc"`
	GeneratedAtJST        string               `json:"generatedAtJst"`
	UpstreamCommit        string               `json:"upstreamCommit"`
	NodeOracle            NodeOracleEvidence   `json:"nodeOracle"`
	GoCommit              string               `json:"goCommit"`
	GoWorkingTreeModified bool                 `json:"goWorkingTreeModified"`
	Corpus                CorpusEvidence       `json:"corpus"`
	Counts                Counts               `json:"counts"`
	APIs                  map[string]APICounts `json:"apis"`
	Differences           map[string]int       `json:"differences"`
	Environment           Environment          `json:"environment"`
	Command               string               `json:"command"`
	DeterministicSHA256   string               `json:"deterministicSha256"`
	SummarySHA256         string               `json:"summarySha256"`
	Results               []CaseResult         `json:"results"`
}

type ProcessResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
	TimedOut bool
}

type RunConfig struct {
	RepositoryRoot        string
	CorpusPath            string
	NodeExecutable        string
	NodeOraclePath        string
	GoRunnerPath          string
	Timeout               time.Duration
	UpstreamCommit        string
	GoCommit              string
	GoWorkingTreeModified bool
	NodeVersion           string
	NodeOracleScript      string
	NodeOracleSource      string
	NodeOracleVersion     string
	Command               string
}

func LoadCases(path string) ([]Case, []byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	var cases []Case
	seen := map[string]struct{}{}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		if strings.TrimSpace(scanner.Text()) == "" {
			continue
		}
		var current Case
		if err := json.Unmarshal(scanner.Bytes(), &current); err != nil {
			return nil, nil, fmt.Errorf("corpus line %d: %w", line, err)
		}
		if current.ID == "" || current.Op == "" || len(current.Args) == 0 {
			return nil, nil, fmt.Errorf("corpus line %d requires id, op, and args", line)
		}
		if _, exists := seen[current.ID]; exists {
			return nil, nil, fmt.Errorf("duplicate case id %q", current.ID)
		}
		seen[current.ID] = struct{}{}
		cases = append(cases, current)
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}
	return cases, data, nil
}

func ParseResponses(data []byte) (map[string]Response, error) {
	responses := map[string]Response{}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		if strings.TrimSpace(scanner.Text()) == "" {
			continue
		}
		var response Response
		if err := json.Unmarshal(scanner.Bytes(), &response); err != nil {
			return nil, fmt.Errorf("response line %d: %w", line, err)
		}
		if response.ID == nil || *response.ID == "" {
			return nil, fmt.Errorf("response line %d has no id", line)
		}
		if _, exists := responses[*response.ID]; exists {
			return nil, fmt.Errorf("duplicate response id %q", *response.ID)
		}
		if response.OK && len(response.Value) == 0 {
			return nil, fmt.Errorf("response %q is successful without value", *response.ID)
		}
		if !response.OK && response.Error == nil {
			return nil, fmt.Errorf("response %q failed without error", *response.ID)
		}
		responses[*response.ID] = response
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return responses, nil
}

func CompareCases(cases []Case, nodeResponses, goResponses map[string]Response) ([]CaseResult, error) {
	known := make(map[string]struct{}, len(cases))
	for _, current := range cases {
		known[current.ID] = struct{}{}
	}
	for id := range nodeResponses {
		if _, ok := known[id]; !ok {
			return nil, fmt.Errorf("unexpected Node response %q", id)
		}
	}
	for id := range goResponses {
		if _, ok := known[id]; !ok {
			return nil, fmt.Errorf("unexpected Go response %q", id)
		}
	}

	results := make([]CaseResult, 0, len(cases))
	for _, current := range cases {
		result := CaseResult{ID: current.ID, API: current.Op, Category: current.Category, DifferenceID: current.DifferenceID}
		node, nodeOK := nodeResponses[current.ID]
		goResponse, goOK := goResponses[current.ID]
		if !nodeOK || !goOK {
			result.Classification = Fail
			result.Reason = fmt.Sprintf("missing response: node=%t go=%t", nodeOK, goOK)
			results = append(results, result)
			continue
		}
		nodeNormalized, err := normalizeResponse(current, node, false)
		if err != nil {
			result.Classification, result.Reason = Fail, "Node normalization: "+err.Error()
			results = append(results, result)
			continue
		}
		goNormalized, err := normalizeResponse(current, goResponse, true)
		if err != nil {
			result.Classification, result.Reason = Fail, "Go normalization: "+err.Error()
			results = append(results, result)
			continue
		}
		result.Node, result.Go = nodeNormalized, goNormalized
		if current.SkipReason != "" {
			result.Classification, result.Reason = Skip, current.SkipReason
			results = append(results, result)
			continue
		}
		equal := nodeNormalized == goNormalized
		if current.DifferenceID != "" {
			description, approved := ApprovedDifferences[current.DifferenceID]
			switch {
			case !approved:
				result.Classification, result.Reason = Fail, "unapproved difference id"
			case equal:
				result.Classification, result.Reason = Fail, "approved difference was not observed"
			default:
				result.Classification, result.Reason = ExpectedDifference, description
			}
		} else if equal {
			result.Classification = Pass
		} else {
			result.Classification, result.Reason = Fail, "unapproved normalized result mismatch"
		}
		results = append(results, result)
	}
	sort.Slice(results, func(i, j int) bool { return results[i].ID < results[j].ID })
	return results, nil
}

func normalizeResponse(current Case, response Response, goSide bool) (string, error) {
	if response.OK {
		var value any
		if err := json.Unmarshal(response.Value, &value); err != nil {
			return "", err
		}
		if strings.Contains(current.Op, "Async") {
			normalizeCallArrays(value)
		}
		encoded, err := json.Marshal(map[string]any{"status": "success", "value": value})
		return string(encoded), err
	}
	if response.Error == nil {
		return "", errors.New("missing error")
	}
	category := response.Error.Category
	message := response.Error.Message
	name := response.Error.Name
	if goSide {
		if response.Error.Category == "resolver-error" {
			message, name = response.Error.CauseMessage, response.Error.CauseName
		}
	} else {
		category = nodeErrorCategory(current, response.Error)
	}
	encoded, err := json.Marshal(map[string]any{"status": "error", "category": category, "name": normalizedErrorName(category, name), "message": message})
	return string(encoded), err
}

func normalizedErrorName(category, name string) string {
	if category == "resolver-error" {
		return name
	}
	return category
}

func nodeErrorCategory(current Case, envelope *ErrorEnvelope) string {
	message := envelope.Message
	if strings.Contains(current.Op, "Fn") && bytes.Contains(current.Args, []byte(message)) {
		return "resolver-error"
	}
	switch {
	case strings.Contains(message, "Could not parse path"):
		return "invalid-path"
	case strings.Contains(message, "Missing \"") && strings.Contains(message, "in the template"), strings.Contains(message, "Unexpected \"") && strings.Contains(message, "tag found"), strings.Contains(message, "Path cannot have"):
		return "invalid-template"
	case strings.Contains(message, "open and close symbols"), strings.Contains(message, "tags should be"), strings.Contains(message, "positive number for max"):
		return "invalid-option"
	case strings.Contains(message, "is not defined in the scope"), strings.Contains(message, "cannot be deeper"):
		return "reference"
	case strings.Contains(message, "Expected a resolver function"):
		return "invalid-resolver"
	case strings.Contains(message, "Symbol"), strings.Contains(message, "Unsupported JavaScript value"):
		return "unsupported-value"
	case strings.Contains(message, "Invalid tokens object"):
		return "invalid-tokens"
	default:
		return envelope.Name
	}
}

func normalizeCallArrays(value any) {
	switch current := value.(type) {
	case []any:
		for _, item := range current {
			normalizeCallArrays(item)
		}
	case map[string]any:
		for key, item := range current {
			if key == "calls" {
				if envelope, ok := item.(map[string]any); ok && envelope["type"] == "array" {
					if calls, ok := envelope["value"].([]any); ok {
						sort.Slice(calls, func(i, j int) bool {
							left, _ := json.Marshal(calls[i])
							right, _ := json.Marshal(calls[j])
							return bytes.Compare(left, right) < 0
						})
					}
				}
			}
			normalizeCallArrays(item)
		}
	}
}

func ExecuteProcess(ctx context.Context, executable string, args []string, directory string) ProcessResult {
	command := exec.CommandContext(ctx, executable, args...)
	command.Dir = directory
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	result := ProcessResult{Stdout: stdout.Bytes(), Stderr: stderr.Bytes()}
	if ctx.Err() != nil {
		result.TimedOut = true
		result.ExitCode = -1
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
		if result.Stderr == nil {
			result.Stderr = []byte(err.Error())
		}
	}
	return result
}

func ValidateProcess(name string, result ProcessResult) error {
	if result.TimedOut {
		return fmt.Errorf("%s process timeout", name)
	}
	if result.ExitCode != 0 {
		return fmt.Errorf("%s process exited %d: %s", name, result.ExitCode, strings.TrimSpace(string(result.Stderr)))
	}
	return nil
}

func Run(config RunConfig) (Report, error) {
	cases, corpusBytes, err := LoadCases(config.CorpusPath)
	if err != nil {
		return Report{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()
	node := ExecuteProcess(ctx, config.NodeExecutable, []string{config.NodeOraclePath, "--input-file", config.CorpusPath}, config.RepositoryRoot)
	if err := ValidateProcess("Node", node); err != nil {
		return Report{}, err
	}
	if len(node.Stderr) != 0 {
		return Report{}, fmt.Errorf("Node process wrote stderr: %s", strings.TrimSpace(string(node.Stderr)))
	}
	goResult := ExecuteProcess(ctx, config.GoRunnerPath, []string{"--input-file", config.CorpusPath}, config.RepositoryRoot)
	if err := ValidateProcess("Go", goResult); err != nil {
		return Report{}, err
	}
	if len(goResult.Stderr) != 0 {
		return Report{}, fmt.Errorf("Go process wrote stderr: %s", strings.TrimSpace(string(goResult.Stderr)))
	}
	nodeResponses, err := ParseResponses(node.Stdout)
	if err != nil {
		return Report{}, fmt.Errorf("Node responses: %w", err)
	}
	goResponses, err := ParseResponses(goResult.Stdout)
	if err != nil {
		return Report{}, fmt.Errorf("Go responses: %w", err)
	}
	results, err := CompareCases(cases, nodeResponses, goResponses)
	if err != nil {
		return Report{}, err
	}
	return buildReport(config, corpusBytes, results), nil
}

func buildReport(config RunConfig, corpus []byte, results []CaseResult) Report {
	now := time.Now().UTC()
	report := Report{
		SchemaVersion: 1, GeneratedAtUTC: now.Format(time.RFC3339), GeneratedAtJST: now.In(time.FixedZone("JST", 9*60*60)).Format(time.RFC3339),
		UpstreamCommit: config.UpstreamCommit, GoCommit: config.GoCommit, GoWorkingTreeModified: config.GoWorkingTreeModified,
		NodeOracle: NodeOracleEvidence{Script: config.NodeOracleScript, Source: config.NodeOracleSource, PackageVersion: config.NodeOracleVersion},
		Corpus:     CorpusEvidence{Path: "testdata/differential/cases.ndjson", SHA256: sha256Hex(corpus)},
		APIs:       map[string]APICounts{}, Differences: map[string]int{},
		Environment: Environment{GoVersion: runtime.Version(), NodeVersion: config.NodeVersion, OS: runtime.GOOS + "/" + runtime.GOARCH},
		Command:     config.Command, Results: results,
	}
	report.Counts.Total = len(results)
	for _, result := range results {
		apiCounts := report.APIs[result.API]
		apiCounts.Total++
		switch result.Classification {
		case Pass:
			report.Counts.Pass++
			apiCounts.Pass++
		case ExpectedDifference:
			report.Counts.ExpectedDifference++
			apiCounts.ExpectedDifference++
			report.Differences[result.DifferenceID]++
		case Skip:
			report.Counts.Skip++
			apiCounts.Skip++
		case Fail:
			report.Counts.Fail++
			apiCounts.Fail++
		}
		report.APIs[result.API] = apiCounts
	}
	payload := struct {
		CorpusSHA   string               `json:"corpusSha"`
		Counts      Counts               `json:"counts"`
		APIs        map[string]APICounts `json:"apis"`
		Differences map[string]int       `json:"differences"`
		Results     []CaseResult         `json:"results"`
	}{report.Corpus.SHA256, report.Counts, report.APIs, report.Differences, report.Results}
	encoded, _ := json.Marshal(payload)
	report.DeterministicSHA256 = sha256Hex(encoded)
	summary, _ := json.Marshal(struct {
		Upstream string `json:"upstream"`
		Go       string `json:"go"`
		Hash     string `json:"deterministicHash"`
	}{report.UpstreamCommit, report.GoCommit, report.DeterministicSHA256})
	report.SummarySHA256 = sha256Hex(summary)
	return report
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func WriteJSONReport(path string, report Report) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func WriteMarkdownReport(path string, report Report) error {
	var output strings.Builder
	fmt.Fprintf(&output, "# Differential validation summary\n\n")
	fmt.Fprintf(&output, "- Generated: `%s` / `%s`\n", report.GeneratedAtJST, report.GeneratedAtUTC)
	fmt.Fprintf(&output, "- Fixed upstream: `%s`\n", report.UpstreamCommit)
	fmt.Fprintf(&output, "- Node oracle: `%s`, source `%s`, package `%s`\n", report.NodeOracle.Script, report.NodeOracle.Source, report.NodeOracle.PackageVersion)
	fmt.Fprintf(&output, "- Go base commit: `%s` (working tree modified: `%t`)\n", report.GoCommit, report.GoWorkingTreeModified)
	fmt.Fprintf(&output, "- Corpus: `%s` (`%s`)\n", report.Corpus.Path, report.Corpus.SHA256)
	fmt.Fprintf(&output, "- Result: PASS `%d`, EXPECTED_DIFFERENCE `%d`, SKIP `%d`, FAIL `%d`, total `%d`\n", report.Counts.Pass, report.Counts.ExpectedDifference, report.Counts.Skip, report.Counts.Fail, report.Counts.Total)
	fmt.Fprintf(&output, "- Deterministic result SHA-256: `%s`\n", report.DeterministicSHA256)
	fmt.Fprintf(&output, "- Summary SHA-256: `%s`\n", report.SummarySHA256)
	fmt.Fprintf(&output, "- Environment: `%s`, Node `%s`, `%s`\n", report.Environment.GoVersion, report.Environment.NodeVersion, report.Environment.OS)
	fmt.Fprintf(&output, "- Command: `%s`\n\n", report.Command)
	fmt.Fprintf(&output, "## API counts\n\n| API | Total | PASS | EXPECTED_DIFFERENCE | SKIP | FAIL |\n| --- | ---: | ---: | ---: | ---: | ---: |\n")
	apiNames := make([]string, 0, len(report.APIs))
	for api := range report.APIs {
		apiNames = append(apiNames, api)
	}
	sort.Strings(apiNames)
	for _, api := range apiNames {
		counts := report.APIs[api]
		fmt.Fprintf(&output, "| `%s` | %d | %d | %d | %d | %d |\n", api, counts.Total, counts.Pass, counts.ExpectedDifference, counts.Skip, counts.Fail)
	}
	fmt.Fprintf(&output, "\n## Approved differences observed\n\n")
	differenceIDs := make([]string, 0, len(report.Differences))
	for id := range report.Differences {
		differenceIDs = append(differenceIDs, id)
	}
	sort.Strings(differenceIDs)
	if len(differenceIDs) == 0 {
		output.WriteString("None.\n")
	}
	for _, id := range differenceIDs {
		fmt.Fprintf(&output, "- `%s`: %d — %s\n", id, report.Differences[id], ApprovedDifferences[id])
	}
	fmt.Fprintf(&output, "\n## Non-PASS cases\n\n| ID | Classification | Difference | Reason |\n| --- | --- | --- | --- |\n")
	nonPass := 0
	for _, result := range report.Results {
		if result.Classification == Pass {
			continue
		}
		nonPass++
		fmt.Fprintf(&output, "| `%s` | %s | `%s` | %s |\n", result.ID, result.Classification, result.DifferenceID, strings.ReplaceAll(result.Reason, "|", "\\|"))
	}
	if nonPass == 0 {
		output.WriteString("| — | — | — | — |\n")
	}
	return os.WriteFile(path, []byte(output.String()), 0o644)
}
