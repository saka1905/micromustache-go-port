package benchmarking

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	mm "github.com/saka1905/micromustache-go-port"
)

var goSink any

type goPrepared struct {
	workload      Workload
	renderer      *mm.Renderer
	tokens        mm.Tokens
	options       mm.CompileOptions
	data          []mm.Scope
	resolver      mm.Resolver
	asyncResolver mm.AsyncResolver
}

func RunGoSuite(suite Suite, workloadBytes []byte, mode string, config RunnerConfig) (RunnerOutput, error) {
	if mode != "validate" && mode != "benchmark" {
		return RunnerOutput{}, fmt.Errorf("unsupported mode %q", mode)
	}
	if err := ValidateConfig(config); err != nil {
		return RunnerOutput{}, err
	}
	output := RunnerOutput{SchemaVersion: SchemaVersion, Mode: mode, Runtime: "go", WorkloadSHA256: SHA256Hex(workloadBytes), Config: config}
	for _, workload := range SortedWorkloads(suite) {
		prepared, err := prepareGo(workload)
		if err != nil {
			return RunnerOutput{}, fmt.Errorf("prepare %s: %w", workload.ID, err)
		}
		validation, err := validateGo(prepared)
		if err != nil {
			return RunnerOutput{}, fmt.Errorf("validate %s: %w", workload.ID, err)
		}
		result := RunnerResult{ID: workload.ID, API: workload.API, Validation: validation}
		if mode == "benchmark" {
			result.Samples, err = benchmarkGo(prepared, config)
			if err != nil {
				return RunnerOutput{}, fmt.Errorf("benchmark %s: %w", workload.ID, err)
			}
			encodedSink, _ := json.Marshal(goSink)
			result.SinkDigest = SHA256Hex(encodedSink)
		}
		output.Results = append(output.Results, result)
	}
	return output, nil
}

func prepareGo(workload Workload) (goPrepared, error) {
	prepared := goPrepared{workload: workload, options: goCompileOptions(workload)}
	dataInputs := workload.DataVariants
	if len(dataInputs) == 0 {
		dataInputs = []map[string]any{workload.Data}
	}
	for _, input := range dataInputs {
		prepared.data = append(prepared.data, goScope(input))
	}
	resolverValues := make(map[string]any, len(workload.Resolver))
	for path, value := range workload.Resolver {
		resolverValues[path] = normalizeGoData(value)
	}
	prepared.resolver = func(path string, _ mm.Scope) (mm.Value, error) {
		if value, ok := resolverValues[path]; ok {
			return value, nil
		}
		return mm.Undefined{}, nil
	}
	prepared.asyncResolver = func(_ context.Context, path string, _ mm.Scope) (mm.Value, error) {
		if value, ok := resolverValues[path]; ok {
			return value, nil
		}
		return mm.Undefined{}, nil
	}
	var err error
	switch workload.API {
	case "renderer.render", "renderer.renderFn", "renderer.renderFnAsync":
		prepared.renderer, err = mm.Compile(workload.Template, prepared.options)
	case "renderer.construct":
		prepared.tokens, err = mm.Tokenize(workload.Template, prepared.options.TokenizeOptions)
	}
	return prepared, err
}

func validateGo(prepared goPrepared) (Validation, error) {
	value, calls, err := executeGo(prepared, 0, true)
	if err != nil {
		return Validation{}, err
	}
	if calls != prepared.workload.Expected.ResolverCalls {
		return Validation{}, fmt.Errorf("resolver calls=%d want=%d", calls, prepared.workload.Expected.ResolverCalls)
	}
	normalized, err := normalizeGoResult(prepared, value)
	if err != nil {
		return Validation{}, err
	}
	encoded, err := json.Marshal(normalized)
	if err != nil {
		return Validation{}, err
	}
	return Validation{Status: "PASS", API: prepared.workload.API, ResultDigest: SHA256Hex(encoded), ResolverCalls: calls}, nil
}

func executeGo(prepared goPrepared, iteration int64, validation bool) (any, int, error) {
	workload := prepared.workload
	data := prepared.data[iteration%int64(len(prepared.data))]
	options := prepared.options
	calls := 0
	resolver := prepared.resolver
	asyncResolver := prepared.asyncResolver
	if validation {
		resolver = func(path string, scope mm.Scope) (mm.Value, error) {
			calls++
			return prepared.resolver(path, scope)
		}
		asyncResolver = func(ctx context.Context, path string, scope mm.Scope) (mm.Value, error) {
			calls++
			return prepared.asyncResolver(ctx, path, scope)
		}
	}

	switch workload.API {
	case "tokenize":
		value, err := mm.Tokenize(workload.Template, options.TokenizeOptions)
		return value, calls, err
	case "get":
		value, err := mm.Get(data, workload.Path, options.GetOptions)
		return value, calls, err
	case "getRef":
		value, err := mm.GetRef(data, mm.Ref(append([]string(nil), workload.Ref...)), options.GetOptions)
		return value, calls, err
	case "render":
		value, err := mm.Render(workload.Template, data, options)
		return value, calls, err
	case "renderFn":
		value, err := mm.RenderFunc(workload.Template, resolver, data, options)
		return value, calls, err
	case "renderFnAsync":
		value, err := mm.RenderFuncAsync(context.Background(), workload.Template, asyncResolver, data, options)
		return value, calls, err
	case "compile":
		renderer, err := mm.Compile(workload.Template, options)
		if err != nil {
			return nil, calls, err
		}
		if validation {
			value, err := renderer.Render(data)
			return value, calls, err
		}
		return renderer, calls, nil
	case "renderer.construct":
		renderer, err := mm.NewRenderer(prepared.tokens, options.RendererOptions)
		if err != nil {
			return nil, calls, err
		}
		if validation {
			value, err := renderer.Render(data)
			return value, calls, err
		}
		return renderer, calls, nil
	case "renderer.render":
		value, err := prepared.renderer.Render(data)
		return value, calls, err
	case "renderer.renderFn":
		value, err := prepared.renderer.RenderFunc(resolver, data)
		return value, calls, err
	case "renderer.renderFnAsync":
		value, err := prepared.renderer.RenderFuncAsync(context.Background(), asyncResolver, data)
		return value, calls, err
	default:
		return nil, calls, fmt.Errorf("unsupported API %q", workload.API)
	}
}

func normalizeGoResult(prepared goPrepared, value any) (any, error) {
	if tokens, ok := value.(mm.Tokens); ok {
		return struct {
			Strings []string `json:"strings"`
			Paths   []string `json:"paths"`
		}{tokens.Strings, tokens.Paths}, nil
	}
	return value, nil
}

func benchmarkGo(prepared goPrepared, config RunnerConfig) ([]Sample, error) {
	iterations := int64(1)
	minimum := time.Duration(config.MinDurationMS) * time.Millisecond
	for {
		elapsed, err := measureGoBatch(prepared, iterations)
		if err != nil {
			return nil, err
		}
		if elapsed >= minimum {
			break
		}
		if iterations > config.MaxIterations/2 {
			return nil, fmt.Errorf("calibration reached max iterations before minimum duration")
		}
		iterations *= 2
	}
	for range config.Warmup {
		if _, err := measureGoBatch(prepared, iterations); err != nil {
			return nil, err
		}
	}

	samples := make([]Sample, 0, config.Samples)
	for len(samples) < config.Samples {
		elapsed, err := measureGoBatch(prepared, iterations)
		if err != nil {
			return nil, err
		}
		if elapsed < minimum {
			if iterations > config.MaxIterations/2 {
				return nil, fmt.Errorf("measured duration below minimum at max iterations")
			}
			iterations *= 2
			samples = samples[:0]
			continue
		}
		sample := Sample{Iterations: iterations, ElapsedNS: elapsed.Nanoseconds(), NSPerOp: float64(elapsed.Nanoseconds()) / float64(iterations)}
		if err := ValidateSample(sample); err != nil {
			return nil, err
		}
		samples = append(samples, sample)
	}
	return samples, nil
}

func measureGoBatch(prepared goPrepared, iterations int64) (time.Duration, error) {
	start := time.Now()
	for iteration := int64(0); iteration < iterations; iteration++ {
		value, _, err := executeGo(prepared, iteration, false)
		if err != nil {
			return 0, err
		}
		goSink = value
	}
	return time.Since(start), nil
}

func goScope(input map[string]any) mm.Scope {
	result := make(mm.Scope, len(input))
	for key, value := range input {
		result[key] = normalizeGoData(value)
	}
	return result
}

func normalizeGoData(value any) any {
	switch current := value.(type) {
	case map[string]any:
		result := make(map[string]any, len(current))
		for key, child := range current {
			result[key] = normalizeGoData(child)
		}
		return result
	case []any:
		result := make([]any, len(current))
		for index, child := range current {
			result[index] = normalizeGoData(child)
		}
		return result
	default:
		return value
	}
}

func goCompileOptions(workload Workload) mm.CompileOptions {
	return mm.CompileOptions{
		RendererOptions: mm.RendererOptions{GetOptions: mm.GetOptions{MaxRefDepth: workload.Options.MaxRefDepth}},
		TokenizeOptions: mm.TokenizeOptions{MaxPathLen: workload.Options.MaxPathLen},
	}
}
