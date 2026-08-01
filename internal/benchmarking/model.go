package benchmarking

import "encoding/json"

const SchemaVersion = 1

type Suite struct {
	SchemaVersion int        `json:"schemaVersion"`
	Workloads     []Workload `json:"workloads"`
}

type Workload struct {
	ID           string           `json:"id"`
	API          string           `json:"api"`
	Category     string           `json:"category"`
	Size         string           `json:"size"`
	Template     string           `json:"template,omitempty"`
	Path         string           `json:"path,omitempty"`
	Ref          []string         `json:"ref,omitempty"`
	Data         map[string]any   `json:"data,omitempty"`
	DataVariants []map[string]any `json:"dataVariants,omitempty"`
	Resolver     map[string]any   `json:"resolver,omitempty"`
	Options      WorkloadOptions  `json:"options,omitempty"`
	TimedSetup   string           `json:"timedSetup"`
	Expected     ExpectedResult   `json:"expected"`
	Metrics      WorkloadMetrics  `json:"metrics"`
}

type WorkloadOptions struct {
	MaxPathLen  int `json:"maxPathLen,omitempty"`
	MaxRefDepth int `json:"maxRefDepth,omitempty"`
}

type ExpectedResult struct {
	Mode          string `json:"mode"`
	ResolverCalls int    `json:"resolverCalls"`
}

type WorkloadMetrics struct {
	TemplateBytes      int `json:"templateBytes"`
	TemplateCharacters int `json:"templateCharacters"`
	Interpolations     int `json:"interpolations"`
	PathCount          int `json:"pathCount"`
	DataNodes          int `json:"dataNodes"`
}

type RunnerConfig struct {
	Warmup                int   `json:"warmup"`
	Samples               int   `json:"samples"`
	MinDurationMS         int64 `json:"minDurationMs"`
	MaxIterations         int64 `json:"maxIterations"`
	ProcessTimeoutSeconds int   `json:"processTimeoutSeconds"`
}

type Validation struct {
	Status        string `json:"status"`
	API           string `json:"api"`
	ResultDigest  string `json:"resultDigest"`
	ResolverCalls int    `json:"resolverCalls"`
}

type Sample struct {
	Iterations int64   `json:"iterations"`
	ElapsedNS  int64   `json:"elapsedNs"`
	NSPerOp    float64 `json:"nsPerOp"`
}

type RunnerResult struct {
	ID         string     `json:"id"`
	API        string     `json:"api"`
	Validation Validation `json:"validation"`
	Samples    []Sample   `json:"samples,omitempty"`
	SinkDigest string     `json:"sinkDigest,omitempty"`
}

type RunnerOutput struct {
	SchemaVersion  int            `json:"schemaVersion"`
	Mode           string         `json:"mode"`
	Runtime        string         `json:"runtime"`
	WorkloadSHA256 string         `json:"workloadSha256"`
	Config         RunnerConfig   `json:"config"`
	Results        []RunnerResult `json:"results"`
}

type FixtureEnvelope struct {
	Node json.RawMessage `json:"node"`
	Go   json.RawMessage `json:"go"`
}

var RequiredAPIs = []string{
	"tokenize", "get", "getRef", "render", "renderFn", "renderFnAsync",
	"compile", "renderer.construct", "renderer.render", "renderer.renderFn", "renderer.renderFnAsync",
}
