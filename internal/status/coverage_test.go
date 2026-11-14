package status

import (
	"encoding/json"
	"testing"
)

func TestStatus_JSONRoundTrip(t *testing.T) {
	in := CoverageStatus{
		Path:      "/proj/main.go",
		Status:    "stale",
		FileHash:  "abc123",
		IndexHash: "def456",
	}
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var out CoverageStatus
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out != in {
		t.Errorf("round trip mismatch: got %+v, want %+v", out, in)
	}
}

func TestChange_JSONRoundTrip(t *testing.T) {
	in := CoverageChange{Path: "/proj/new.go", Status: "new"}
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var out CoverageChange
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out != in {
		t.Errorf("round trip mismatch: got %+v, want %+v", out, in)
	}
}
