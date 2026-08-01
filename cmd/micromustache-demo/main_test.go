package main

import (
	"bytes"
	"errors"
	"os"
	"regexp"
	"strings"
	"testing"
)

const expectedDemoOutput = `micromustache Go port demo

[1/6] Basic Render
output: こんにちは, Aoi from 米沢!
status: PASS

[2/6] Tokenize
literal fragments: 3
paths: user.name, items[1]
status: PASS

[3/6] Get and GetRef
Get: Ren
GetRef: Aoi
status: PASS

[4/6] Compile and Renderer Reuse
render 1: Hello, Aoi!
render 2: Hello, Ren!
NewRenderer: left + right
status: PASS

[5/6] Synchronous Resolver
top-level: Aoi from Yonezawa
top-level calls: name, city
compiled: Sendai welcomes Ren
compiled calls: city, name
status: PASS

[6/6] Asynchronous Resolver
top-level: first then second
top-level calls: 2
compiled: left + right
compiled calls: 2
status: PASS

DEMO_STATUS: PASS
`

func TestRunDemoOutputAndPublicAPIs(t *testing.T) {
	var output bytes.Buffer
	if err := runDemo(&output); err != nil {
		t.Fatalf("runDemo() error = %v", err)
	}
	if got := output.String(); got != expectedDemoOutput {
		t.Fatalf("runDemo() output:\n%s\nwant:\n%s", got, expectedDemoOutput)
	}
	if strings.Count(output.String(), "DEMO_STATUS: PASS") != 1 {
		t.Fatal("final PASS status must occur exactly once")
	}
	privatePath := regexp.MustCompile(`(?i)([a-z]:\\|/users/|\\users\\|/home/)`)
	if privatePath.MatchString(output.String()) {
		t.Fatal("demo output contains an absolute or private path")
	}
	if strings.Contains(strings.ToLower(output.String()), "node") {
		t.Fatal("demo output contains a Node dependency expression")
	}
	timestamp := regexp.MustCompile(`\b\d{4}-\d{2}-\d{2}(?:[tT ]\d{2}:\d{2})?\b`)
	if timestamp.MatchString(output.String()) {
		t.Fatal("demo output contains a timestamp")
	}
}

func TestRunDemoIsDeterministic(t *testing.T) {
	for run := 0; run < 20; run++ {
		var output bytes.Buffer
		if err := runDemo(&output); err != nil {
			t.Fatalf("run %d error = %v", run, err)
		}
		if output.String() != expectedDemoOutput {
			t.Fatalf("run %d was not deterministic", run)
		}
	}
}

func TestTrackedEvidenceMatchesOutput(t *testing.T) {
	evidence, err := os.ReadFile("../../evidence/demo-output.txt")
	if err != nil {
		t.Fatalf("read tracked demo evidence: %v", err)
	}
	if string(evidence) != expectedDemoOutput {
		t.Fatal("tracked demo evidence does not match the verified direct output")
	}
}

func TestFailureDoesNotEmitFinalPass(t *testing.T) {
	var output bytes.Buffer
	err := runDemoSections(&output, []demoSection{{
		name: "Broken Fixture",
		run: func() ([]string, error) {
			return nil, errors.New("intentional failure")
		},
	}})
	if err == nil || !strings.Contains(err.Error(), "Broken Fixture") {
		t.Fatalf("runDemoSections() error = %v", err)
	}
	if strings.Contains(output.String(), "DEMO_STATUS: PASS") {
		t.Fatal("failure output contains final PASS status")
	}
}
