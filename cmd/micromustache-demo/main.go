// Command micromustache-demo demonstrates the exported Go API without Node.js
// or repository-relative runtime data.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync/atomic"

	mm "github.com/saka1905/micromustache-go-port"
)

type demoSection struct {
	name string
	run  func() ([]string, error)
}

func main() {
	if err := runDemo(os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "DEMO_ERROR: %v\n", err)
		os.Exit(1)
	}
}

func runDemo(output io.Writer) error {
	return runDemoSections(output, []demoSection{
		{name: "Basic Render", run: basicRenderSection},
		{name: "Tokenize", run: tokenizeSection},
		{name: "Get and GetRef", run: getSection},
		{name: "Compile and Renderer Reuse", run: compileSection},
		{name: "Synchronous Resolver", run: syncResolverSection},
		{name: "Asynchronous Resolver", run: asyncResolverSection},
	})
}

func runDemoSections(output io.Writer, sections []demoSection) error {
	if _, err := fmt.Fprintln(output, "micromustache Go port demo"); err != nil {
		return fmt.Errorf("write heading: %w", err)
	}
	if _, err := fmt.Fprintln(output); err != nil {
		return fmt.Errorf("write heading separator: %w", err)
	}
	for index, section := range sections {
		lines, err := section.run()
		if err != nil {
			return fmt.Errorf("section %q: %w", section.name, err)
		}
		if _, err := fmt.Fprintf(output, "[%d/%d] %s\n", index+1, len(sections), section.name); err != nil {
			return fmt.Errorf("write section %q heading: %w", section.name, err)
		}
		for _, line := range lines {
			if _, err := fmt.Fprintln(output, line); err != nil {
				return fmt.Errorf("write section %q output: %w", section.name, err)
			}
		}
		if _, err := fmt.Fprintln(output, "status: PASS"); err != nil {
			return fmt.Errorf("write section %q status: %w", section.name, err)
		}
		if _, err := fmt.Fprintln(output); err != nil {
			return fmt.Errorf("write section %q separator: %w", section.name, err)
		}
	}
	if _, err := fmt.Fprintln(output, "DEMO_STATUS: PASS"); err != nil {
		return fmt.Errorf("write final status: %w", err)
	}
	return nil
}

func basicRenderSection() ([]string, error) {
	result, err := mm.Render(
		"{{greeting}}, {{person.name}} from {{person.city}}!",
		mm.Scope{"greeting": "こんにちは", "person": mm.Scope{"name": "Aoi", "city": "Sample City"}},
		mm.CompileOptions{},
	)
	if err != nil {
		return nil, err
	}
	if err := requireString("Render", result, "こんにちは, Aoi from Sample City!"); err != nil {
		return nil, err
	}
	return []string{"output: " + result}, nil
}

func tokenizeSection() ([]string, error) {
	tokens, err := mm.Tokenize("Hello {{ user.name }}; second={{items[1]}}.", mm.TokenizeOptions{})
	if err != nil {
		return nil, err
	}
	if len(tokens.Strings) != 3 || len(tokens.Paths) != 2 || tokens.Paths[0] != "user.name" || tokens.Paths[1] != "items[1]" {
		return nil, fmt.Errorf("Tokenize returned unexpected public tokens: %#v", tokens)
	}
	return []string{
		fmt.Sprintf("literal fragments: %d", len(tokens.Strings)),
		"paths: " + strings.Join(tokens.Paths, ", "),
	}, nil
}

func getSection() ([]string, error) {
	scope := mm.Scope{"team": mm.Scope{"members": []any{mm.Scope{"name": "Aoi"}, mm.Scope{"name": "Ren"}}}}
	fromPath, err := mm.Get(scope, "team.members[1].name", mm.GetOptions{})
	if err != nil {
		return nil, err
	}
	fromRef, err := mm.GetRef(scope, mm.Ref{"team", "members", "0", "name"}, mm.GetOptions{})
	if err != nil {
		return nil, err
	}
	pathValue, ok := fromPath.(string)
	if !ok {
		return nil, fmt.Errorf("Get returned %T, want string", fromPath)
	}
	refValue, ok := fromRef.(string)
	if !ok {
		return nil, fmt.Errorf("GetRef returned %T, want string", fromRef)
	}
	if err := requireString("Get", pathValue, "Ren"); err != nil {
		return nil, err
	}
	if err := requireString("GetRef", refValue, "Aoi"); err != nil {
		return nil, err
	}
	return []string{"Get: " + pathValue, "GetRef: " + refValue}, nil
}

func compileSection() ([]string, error) {
	renderer, err := mm.Compile("Hello, {{name}}!", mm.CompileOptions{})
	if err != nil {
		return nil, err
	}
	first, err := renderer.Render(mm.Scope{"name": "Aoi"})
	if err != nil {
		return nil, err
	}
	second, err := renderer.Render(mm.Scope{"name": "Ren"})
	if err != nil {
		return nil, err
	}
	if err := requireString("first Renderer.Render", first, "Hello, Aoi!"); err != nil {
		return nil, err
	}
	if err := requireString("second Renderer.Render", second, "Hello, Ren!"); err != nil {
		return nil, err
	}
	tokens, err := mm.Tokenize("{{left}} + {{right}}", mm.TokenizeOptions{})
	if err != nil {
		return nil, err
	}
	constructed, err := mm.NewRenderer(tokens, mm.RendererOptions{})
	if err != nil {
		return nil, err
	}
	constructedOutput, err := constructed.Render(mm.Scope{"left": "left", "right": "right"})
	if err != nil {
		return nil, err
	}
	if err := requireString("NewRenderer.Render", constructedOutput, "left + right"); err != nil {
		return nil, err
	}
	return []string{"render 1: " + first, "render 2: " + second, "NewRenderer: " + constructedOutput}, nil
}

func syncResolverSection() ([]string, error) {
	makeResolver := func(calls *[]string) mm.Resolver {
		return func(path string, scope mm.Scope) (mm.Value, error) {
			*calls = append(*calls, path)
			value, exists := scope[path]
			if !exists {
				return nil, fmt.Errorf("fixed resolver has no value for %q", path)
			}
			return value, nil
		}
	}
	var topCalls []string
	top, err := mm.RenderFunc("{{name}} from {{city}}", makeResolver(&topCalls), mm.Scope{"name": "Aoi", "city": "Sample City"}, mm.CompileOptions{})
	if err != nil {
		return nil, err
	}
	renderer, err := mm.Compile("{{city}} welcomes {{name}}", mm.CompileOptions{})
	if err != nil {
		return nil, err
	}
	var compiledCalls []string
	compiled, err := renderer.RenderFunc(makeResolver(&compiledCalls), mm.Scope{"name": "Ren", "city": "Example City"})
	if err != nil {
		return nil, err
	}
	if err := requireString("RenderFunc", top, "Aoi from Sample City"); err != nil {
		return nil, err
	}
	if err := requireString("Renderer.RenderFunc", compiled, "Example City welcomes Ren"); err != nil {
		return nil, err
	}
	if strings.Join(topCalls, ", ") != "name, city" || strings.Join(compiledCalls, ", ") != "city, name" {
		return nil, fmt.Errorf("unexpected synchronous resolver calls: top=%v compiled=%v", topCalls, compiledCalls)
	}
	return []string{
		"top-level: " + top,
		"top-level calls: " + strings.Join(topCalls, ", "),
		"compiled: " + compiled,
		"compiled calls: " + strings.Join(compiledCalls, ", "),
	}, nil
}

func asyncResolverSection() ([]string, error) {
	makeResolver := func(calls *atomic.Int64) mm.AsyncResolver {
		return func(ctx context.Context, path string, scope mm.Scope) (mm.Value, error) {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			calls.Add(1)
			value, exists := scope[path]
			if !exists {
				return nil, fmt.Errorf("fixed async resolver has no value for %q", path)
			}
			return value, nil
		}
	}
	var topCalls atomic.Int64
	top, err := mm.RenderFuncAsync(context.Background(), "{{first}} then {{second}}", makeResolver(&topCalls), mm.Scope{"first": "first", "second": "second"}, mm.CompileOptions{})
	if err != nil {
		return nil, err
	}
	renderer, err := mm.Compile("{{left}} + {{right}}", mm.CompileOptions{})
	if err != nil {
		return nil, err
	}
	var compiledCalls atomic.Int64
	compiled, err := renderer.RenderFuncAsync(context.Background(), makeResolver(&compiledCalls), mm.Scope{"left": "left", "right": "right"})
	if err != nil {
		return nil, err
	}
	if err := requireString("RenderFuncAsync", top, "first then second"); err != nil {
		return nil, err
	}
	if err := requireString("Renderer.RenderFuncAsync", compiled, "left + right"); err != nil {
		return nil, err
	}
	if topCalls.Load() != 2 || compiledCalls.Load() != 2 {
		return nil, fmt.Errorf("unexpected asynchronous resolver counts: top=%d compiled=%d", topCalls.Load(), compiledCalls.Load())
	}
	return []string{
		"top-level: " + top,
		fmt.Sprintf("top-level calls: %d", topCalls.Load()),
		"compiled: " + compiled,
		fmt.Sprintf("compiled calls: %d", compiledCalls.Load()),
	}, nil
}

func requireString(operation, got, want string) error {
	if got != want {
		return fmt.Errorf("%s returned %q, want %q", operation, got, want)
	}
	return nil
}
