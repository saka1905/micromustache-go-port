package differential

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestCompareClassifications(t *testing.T) {
	tests := []struct {
		name       string
		current    Case
		node       Response
		goResponse Response
		want       Classification
	}{
		{"identical responses", testCase("same"), success("same", "A"), success("same", "A"), Pass},
		{"output mismatch", testCase("mismatch"), success("mismatch", "A"), success("mismatch", "B"), Fail},
		{"approved difference", Case{ID: "approved", Op: "render", Args: json.RawMessage(`{}`), DifferenceID: "DIFF-JS-PROTOTYPE"}, success("approved", "A"), success("approved", "B"), ExpectedDifference},
		{"unapproved difference", Case{ID: "unapproved", Op: "render", Args: json.RawMessage(`{}`), DifferenceID: "UNKNOWN"}, success("unapproved", "A"), success("unapproved", "B"), Fail},
		{"stale approved difference", Case{ID: "stale", Op: "render", Args: json.RawMessage(`{}`), DifferenceID: "DIFF-JS-PROTOTYPE"}, success("stale", "A"), success("stale", "A"), Fail},
		{"skip with reason", Case{ID: "skip", Op: "skip", Args: json.RawMessage(`{}`), SkipReason: "not representable"}, failure("skip", "RangeError", "unsupported"), failure("skip", "GoError", "unsupported"), Skip},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			results, err := CompareCases([]Case{test.current}, map[string]Response{test.current.ID: test.node}, map[string]Response{test.current.ID: test.goResponse})
			if err != nil {
				t.Fatalf("CompareCases() error = %v", err)
			}
			if len(results) != 1 || results[0].Classification != test.want {
				t.Fatalf("results = %#v; want %s", results, test.want)
			}
		})
	}
}

func TestCompareDetectsResponseSetFailures(t *testing.T) {
	t.Run("missing response", func(t *testing.T) {
		current := testCase("missing")
		results, err := CompareCases([]Case{current}, map[string]Response{"missing": success("missing", "A")}, map[string]Response{})
		if err != nil || len(results) != 1 || results[0].Classification != Fail || !strings.Contains(results[0].Reason, "missing response") {
			t.Fatalf("results=%#v error=%v", results, err)
		}
	})
	t.Run("unexpected response", func(t *testing.T) {
		current := testCase("known")
		_, err := CompareCases([]Case{current}, map[string]Response{"known": success("known", "A"), "extra": success("extra", "A")}, map[string]Response{"known": success("known", "A")})
		if err == nil || !strings.Contains(err.Error(), "unexpected Node response") {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestParseResponsesRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		data string
		text string
	}{
		{"malformed JSON", "{\n", "response line 1"},
		{"duplicate id", responseLine(success("x", "A")) + responseLine(success("x", "B")), "duplicate response id"},
		{"missing id", `{"id":null,"ok":false,"error":{"name":"Error","message":"bad"}}` + "\n", "has no id"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ParseResponses([]byte(test.data)); err == nil || !strings.Contains(err.Error(), test.text) {
				t.Fatalf("error = %v; want %q", err, test.text)
			}
		})
	}
}

func TestResponseCollectionIsOrderIndependentAndReportOrderingIsDeterministic(t *testing.T) {
	cases := []Case{testCase("b"), testCase("a")}
	node, err := ParseResponses([]byte(responseLine(success("a", "A")) + responseLine(success("b", "B"))))
	if err != nil {
		t.Fatal(err)
	}
	goResponses, err := ParseResponses([]byte(responseLine(success("b", "B")) + responseLine(success("a", "A"))))
	if err != nil {
		t.Fatal(err)
	}
	results, err := CompareCases(cases, node, goResponses)
	if err != nil {
		t.Fatal(err)
	}
	if got := []string{results[0].ID, results[1].ID}; !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Fatalf("result order = %#v", got)
	}
	config := RunConfig{UpstreamCommit: "upstream", GoCommit: "go", NodeVersion: "node"}
	first := buildReport(config, []byte("corpus"), results)
	second := buildReport(config, []byte("corpus"), results)
	if first.DeterministicSHA256 != second.DeterministicSHA256 || first.SummarySHA256 != second.SummarySHA256 {
		t.Fatalf("hashes differ: %#v %#v", first, second)
	}
}

func TestProcessFailuresAreFatal(t *testing.T) {
	tests := []struct {
		name   string
		result ProcessResult
		text   string
	}{
		{"Node process non-zero", ProcessResult{ExitCode: 2, Stderr: []byte("node failed")}, "Node process exited 2"},
		{"Go process non-zero", ProcessResult{ExitCode: 3, Stderr: []byte("go failed")}, "Go process exited 3"},
		{"timeout", ProcessResult{ExitCode: -1, TimedOut: true}, "process timeout"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			name := strings.Fields(test.name)[0]
			if err := ValidateProcess(name, test.result); err == nil || !strings.Contains(err.Error(), test.text) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestExecuteProcessTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("process timeout test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	result := ExecuteProcess(ctx, os.Args[0], []string{"-test.run=TestDifferentialHelperProcess", "--", "wait"}, "")
	if !result.TimedOut {
		t.Fatalf("result = %#v", result)
	}
}

func TestDifferentialHelperProcess(t *testing.T) {
	for _, argument := range os.Args {
		if argument == "wait" {
			time.Sleep(2 * time.Second)
			return
		}
	}
}

func TestIntentionallyBrokenFixtureIsDetected(t *testing.T) {
	root := filepath.Join("..", "..", "testdata", "differential", "fixtures")
	cases, _, err := LoadCases(filepath.Join(root, "broken-case.ndjson"))
	if err != nil {
		t.Fatal(err)
	}
	nodeBytes, err := os.ReadFile(filepath.Join(root, "broken-node.ndjson"))
	if err != nil {
		t.Fatal(err)
	}
	goBytes, err := os.ReadFile(filepath.Join(root, "broken-go.ndjson"))
	if err != nil {
		t.Fatal(err)
	}
	node, _ := ParseResponses(nodeBytes)
	goResponses, _ := ParseResponses(goBytes)
	results, err := CompareCases(cases, node, goResponses)
	if err != nil || len(results) != 1 || results[0].Classification != Fail {
		t.Fatalf("results=%#v error=%v", results, err)
	}
}

func testCase(id string) Case {
	return Case{ID: id, Op: "render", Args: json.RawMessage(`{}`)}
}

func success(id, value string) Response {
	raw := json.RawMessage(`{"type":"string","value":` + mustJSON(value) + `}`)
	return Response{ID: &id, OK: true, Value: raw}
}

func failure(id, name, message string) Response {
	return Response{ID: &id, OK: false, Error: &ErrorEnvelope{Name: name, Message: message}}
}

func mustJSON(value string) string {
	encoded, _ := json.Marshal(value)
	return string(encoded)
}

func responseLine(response Response) string {
	var output bytes.Buffer
	_ = json.NewEncoder(&output).Encode(response)
	return output.String()
}
