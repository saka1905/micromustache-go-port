package differential

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	mm "github.com/saka1905/micromustache-go-port"
)

type operationArgs struct {
	Template string          `json:"template"`
	Data     json.RawMessage `json:"data"`
	Scope    json.RawMessage `json:"scope"`
	Options  json.RawMessage `json:"options"`
	Path     string          `json:"path"`
	Ref      []string        `json:"ref"`
	Tokens   wireTokens      `json:"tokens"`
	Resolver resolverSpec    `json:"resolver"`
	Trace    bool            `json:"trace"`
	Context  contextSpec     `json:"context"`
	Steps    []sequenceStep  `json:"steps"`
}

type contextSpec struct {
	State string `json:"state"`
}

type resolverSpec struct {
	Paths   map[string]resolverAction `json:"paths"`
	Default *resolverAction           `json:"default"`
}

type resolverAction struct {
	Action  string          `json:"action"`
	Value   json.RawMessage `json:"value"`
	Error   namedErrorSpec  `json:"error"`
	DelayMS int             `json:"delayMs"`
}

type namedErrorSpec struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}

type sequenceStep struct {
	Op       string          `json:"op"`
	Data     json.RawMessage `json:"data"`
	Scope    json.RawMessage `json:"scope"`
	Resolver resolverSpec    `json:"resolver"`
	Trace    bool            `json:"trace"`
	Context  contextSpec     `json:"context"`
}

type namedResolverError struct {
	name    string
	message string
}

func (e *namedResolverError) Error() string { return e.message }

type unsupportedResolverValue struct{}

type callRecorder struct {
	mutex sync.Mutex
	paths []string
}

func (r *callRecorder) Add(path string) {
	r.mutex.Lock()
	r.paths = append(r.paths, path)
	r.mutex.Unlock()
}

func (r *callRecorder) Values() []string {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return append([]string(nil), r.paths...)
}

func HandleRequest(request Request) Response {
	id := request.ID
	value, err := invokeGo(request.Op, request.Args)
	if err != nil {
		return Response{ID: &id, OK: false, Error: goErrorEnvelope(err)}
	}
	encoded, err := EncodeValue(value)
	if err != nil {
		return Response{ID: &id, OK: false, Error: &ErrorEnvelope{Name: "GoCodecError", Message: err.Error(), Category: "codec"}}
	}
	return Response{ID: &id, OK: true, Value: encoded}
}

func invokeGo(op string, rawArgs json.RawMessage) (any, error) {
	var args operationArgs
	if err := decodeStrict(rawArgs, &args); err != nil {
		return nil, err
	}
	compileOptions, err := DecodeCompileOptions(args.Options)
	if err != nil {
		return nil, err
	}
	scope, err := DecodeScope(args.Scope)
	if err != nil {
		return nil, err
	}
	data, err := DecodeScope(args.Data)
	if err != nil {
		return nil, err
	}

	switch op {
	case "render":
		return mm.Render(args.Template, data, compileOptions)
	case "renderFn":
		return callTopLevelResolver(args, scope, compileOptions, false)
	case "renderFnAsync":
		return callTopLevelResolver(args, scope, compileOptions, true)
	case "compile":
		if _, err := mm.Compile(args.Template, compileOptions); err != nil {
			return nil, err
		}
		return map[string]any{"kind": "renderer"}, nil
	case "compile.render":
		renderer, err := mm.Compile(args.Template, compileOptions)
		if err != nil {
			return nil, err
		}
		return renderer.Render(data)
	case "compile.renderFn", "compile.renderFnAsync":
		renderer, err := mm.Compile(args.Template, compileOptions)
		if err != nil {
			return nil, err
		}
		return callRendererResolver(renderer, args.Resolver, args.Trace, scope, args.Context, op == "compile.renderFnAsync")
	case "compile.sequence":
		renderer, err := mm.Compile(args.Template, compileOptions)
		if err != nil {
			return nil, err
		}
		return runSequence(renderer, args.Steps)
	case "get":
		options, err := DecodeGetOptions(args.Options)
		if err != nil {
			return nil, err
		}
		return mm.Get(scope, args.Path, options)
	case "getRef":
		options, err := DecodeGetOptions(args.Options)
		if err != nil {
			return nil, err
		}
		return mm.GetRef(scope, mm.Ref(append([]string(nil), args.Ref...)), options)
	case "tokenize":
		options, err := DecodeTokenizeOptions(args.Options)
		if err != nil {
			return nil, err
		}
		return mm.Tokenize(args.Template, options)
	case "renderer.construct":
		options, err := DecodeRendererOptions(args.Options)
		if err != nil {
			return nil, err
		}
		if _, err := mm.NewRenderer(args.Tokens.Go(), options); err != nil {
			return nil, err
		}
		return map[string]any{"kind": "renderer"}, nil
	case "renderer.render":
		renderer, err := newWireRenderer(args)
		if err != nil {
			return nil, err
		}
		return renderer.Render(data)
	case "renderer.renderFn", "renderer.renderFnAsync":
		renderer, err := newWireRenderer(args)
		if err != nil {
			return nil, err
		}
		return callRendererResolver(renderer, args.Resolver, args.Trace, scope, args.Context, op == "renderer.renderFnAsync")
	case "renderer.sequence":
		renderer, err := newWireRenderer(args)
		if err != nil {
			return nil, err
		}
		return runSequence(renderer, args.Steps)
	default:
		return nil, fmt.Errorf("unsupported operation: %s", op)
	}
}

func newWireRenderer(args operationArgs) (*mm.Renderer, error) {
	options, err := DecodeRendererOptions(args.Options)
	if err != nil {
		return nil, err
	}
	return mm.NewRenderer(args.Tokens.Go(), options)
}

func callTopLevelResolver(args operationArgs, scope mm.Scope, options mm.CompileOptions, asynchronous bool) (any, error) {
	recorder := &callRecorder{}
	if asynchronous {
		ctx, cancel := makeContext(args.Context)
		defer cancel()
		value, err := mm.RenderFuncAsync(ctx, args.Template, makeAsyncResolver(args.Resolver, recorder), scope, options)
		return traced(value, recorder.Values(), args.Trace), err
	}
	value, err := mm.RenderFunc(args.Template, makeResolver(args.Resolver, recorder), scope, options)
	return traced(value, recorder.Values(), args.Trace), err
}

func callRendererResolver(renderer *mm.Renderer, spec resolverSpec, trace bool, scope mm.Scope, contextOptions contextSpec, asynchronous bool) (any, error) {
	recorder := &callRecorder{}
	if asynchronous {
		ctx, cancel := makeContext(contextOptions)
		defer cancel()
		value, err := renderer.RenderFuncAsync(ctx, makeAsyncResolver(spec, recorder), scope)
		return traced(value, recorder.Values(), trace), err
	}
	value, err := renderer.RenderFunc(makeResolver(spec, recorder), scope)
	return traced(value, recorder.Values(), trace), err
}

func makeResolver(spec resolverSpec, recorder *callRecorder) mm.Resolver {
	return func(path string, _ mm.Scope) (mm.Value, error) {
		recorder.Add(path)
		return resolve(spec, path, false, context.Background())
	}
}

func makeAsyncResolver(spec resolverSpec, recorder *callRecorder) mm.AsyncResolver {
	return func(ctx context.Context, path string, _ mm.Scope) (mm.Value, error) {
		recorder.Add(path)
		return resolve(spec, path, true, ctx)
	}
}

func resolve(spec resolverSpec, path string, asynchronous bool, ctx context.Context) (mm.Value, error) {
	action, ok := spec.Paths[path]
	if !ok {
		if spec.Default == nil {
			return nil, errors.New("resolver action is missing")
		}
		action = *spec.Default
	}
	if asynchronous && action.DelayMS > 0 {
		timer := time.NewTimer(time.Duration(action.DelayMS) * time.Millisecond)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	switch action.Action {
	case "value":
		return DecodeValue(action.Value)
	case "undefined":
		return mm.Undefined{}, nil
	case "error":
		return nil, &namedResolverError{name: action.Error.Name, message: action.Error.Message}
	case "unsupported":
		return unsupportedResolverValue{}, nil
	default:
		return nil, fmt.Errorf("unknown resolver action %q", action.Action)
	}
}

func traced(value any, calls []string, enabled bool) any {
	if !enabled {
		return value
	}
	return map[string]any{"result": value, "calls": calls}
}

func makeContext(spec contextSpec) (context.Context, context.CancelFunc) {
	switch spec.State {
	case "canceled":
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		return ctx, func() {}
	case "deadline":
		return context.WithDeadline(context.Background(), time.Unix(1, 0))
	default:
		return context.WithCancel(context.Background())
	}
}

func runSequence(renderer *mm.Renderer, steps []sequenceStep) ([]any, error) {
	results := make([]any, 0, len(steps))
	for _, step := range steps {
		var value any
		var err error
		switch step.Op {
		case "render":
			var data mm.Scope
			data, err = DecodeScope(step.Data)
			if err == nil {
				value, err = renderer.Render(data)
			}
		case "renderFn", "renderFnAsync":
			var scope mm.Scope
			scope, err = DecodeScope(step.Scope)
			if err == nil {
				value, err = callRendererResolver(renderer, step.Resolver, step.Trace, scope, step.Context, step.Op == "renderFnAsync")
			}
		default:
			err = fmt.Errorf("unsupported sequence step: %s", step.Op)
		}
		if err != nil {
			results = append(results, map[string]any{"ok": false})
		} else {
			results = append(results, map[string]any{"ok": true, "value": value})
		}
	}
	return results, nil
}

func goErrorEnvelope(err error) *ErrorEnvelope {
	envelope := &ErrorEnvelope{Name: "GoError", Message: err.Error(), Category: goErrorCategory(err)}
	var resolverError *namedResolverError
	if errors.As(err, &resolverError) {
		envelope.Category = "resolver-error"
		envelope.CauseName = resolverError.name
		envelope.CauseMessage = resolverError.message
	}
	return envelope
}

func goErrorCategory(err error) string {
	switch {
	case errors.Is(err, mm.ErrInvalidTemplate):
		return "invalid-template"
	case errors.Is(err, mm.ErrInvalidPath):
		return "invalid-path"
	case errors.Is(err, mm.ErrInvalidOption):
		return "invalid-option"
	case errors.Is(err, mm.ErrReference):
		return "reference"
	case errors.Is(err, mm.ErrUnsupportedValue):
		return "unsupported-value"
	case errors.Is(err, mm.ErrInvalidResolver):
		return "invalid-resolver"
	case errors.Is(err, mm.ErrInvalidContext):
		return "invalid-context"
	case errors.Is(err, mm.ErrInvalidTokens):
		return "invalid-tokens"
	case errors.Is(err, mm.ErrInvalidRenderer):
		return "invalid-renderer"
	case errors.Is(err, context.Canceled):
		return "context-canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline-exceeded"
	default:
		return "go-error"
	}
}

func RunGoOracle(input io.Reader, output io.Writer) error {
	decoder := json.NewDecoder(input)
	encoder := json.NewEncoder(output)
	for {
		var request Request
		if err := decoder.Decode(&request); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			response := Response{ID: nil, OK: false, Error: &ErrorEnvelope{Name: "GoProtocolError", Message: err.Error(), Category: "protocol"}}
			if encodeErr := encoder.Encode(response); encodeErr != nil {
				return encodeErr
			}
			return err
		}
		if err := encoder.Encode(HandleRequest(request)); err != nil {
			return err
		}
	}
}

func OpenOracleInput(path string) (io.ReadCloser, error) {
	if path == "" {
		return io.NopCloser(os.Stdin), nil
	}
	return os.Open(path)
}
