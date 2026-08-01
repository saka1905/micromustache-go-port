package differential

import (
	"bytes"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"

	mm "github.com/saka1905/micromustache-go-port"
)

func TestGoOracleSmokeUsesPublicOperations(t *testing.T) {
	path := filepath.Join("..", "..", "oracle", "cases", "smoke.ndjson")
	input, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := RunGoOracle(bytes.NewReader(input), &output); err != nil {
		t.Fatalf("RunGoOracle() error = %v", err)
	}
	responses, err := ParseResponses(output.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	cases, _, err := LoadCases(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(responses) != len(cases) || len(cases) < 31 {
		t.Fatalf("responses=%d cases=%d", len(responses), len(cases))
	}
}

func TestWireCodecSpecialValues(t *testing.T) {
	values := []any{mm.Undefined{}, nil, false, true, float64(0), math.Copysign(0, -1), 1.5, math.NaN(), math.Inf(1), math.Inf(-1), "世界", []any{1.0, nil}, map[string]any{"x": "y"}}
	for _, value := range values {
		encoded, err := EncodeValue(value)
		if err != nil {
			t.Fatalf("EncodeValue(%T) error = %v", value, err)
		}
		if _, err := DecodeValue(encoded); err != nil {
			t.Fatalf("DecodeValue(%s) error = %v", encoded, err)
		}
	}
}

func TestGoOracleMalformedJSONFails(t *testing.T) {
	var output bytes.Buffer
	err := RunGoOracle(bytes.NewBufferString("{\n"), &output)
	if err == nil {
		t.Fatal("RunGoOracle() accepted malformed JSON")
	}
	var response Response
	if decodeErr := json.Unmarshal(output.Bytes(), &response); decodeErr != nil || response.OK || response.Error == nil || response.Error.Category != "protocol" {
		t.Fatalf("response=%#v decodeError=%v", response, decodeErr)
	}
}
