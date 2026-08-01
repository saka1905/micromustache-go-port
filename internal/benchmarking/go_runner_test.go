package benchmarking

import "testing"

func TestGoRunnerUsesEveryDeclaredPublicOperation(t *testing.T) {
	suite, data := loadTestSuite(t)
	config := RunnerConfig{Warmup: 3, Samples: 7, MinDurationMS: 1, MaxIterations: 1024, ProcessTimeoutSeconds: 300}
	output, err := RunGoSuite(suite, data, "validate", config)
	if err != nil {
		t.Fatal(err)
	}
	if len(output.Results) != len(suite.Workloads) {
		t.Fatalf("results=%d workloads=%d", len(output.Results), len(suite.Workloads))
	}
	for _, result := range output.Results {
		if result.Validation.Status != "PASS" || result.Validation.API != result.API {
			t.Fatalf("invalid result %+v", result)
		}
	}
}
