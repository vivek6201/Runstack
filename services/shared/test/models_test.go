package test

import (
	"testing"

	"code-runner/shared/models"
)

func TestJSONBTestResults_ScanAndValue(t *testing.T) {
	results := models.JSONBTestResults{
		{
			TestCaseID:     "1",
			Passed:         true,
			ActualOutput:   "10\n",
			ExpectedOutput: "10\n",
			TimeTakenMs:    15,
		},
	}

	val, err := results.Value()
	if err != nil {
		t.Fatalf("Value() error: %v", err)
	}

	valBytes, ok := val.([]byte)
	if !ok {
		// If string was returned
		if str, ok := val.(string); ok {
			valBytes = []byte(str)
		} else {
			t.Fatalf("unexpected type for Valuer: %T", val)
		}
	}

	var scanned models.JSONBTestResults
	if err := scanned.Scan(valBytes); err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	if len(scanned) != 1 {
		t.Fatalf("expected 1 record, got %d", len(scanned))
	}

	if scanned[0].TestCaseID != "1" || !scanned[0].Passed {
		t.Errorf("unexpected scanned content: %+v", scanned[0])
	}
}
