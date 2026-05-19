package frags

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// --- JSON TESTS ---

func TestSessions_JSON_Marshal_PreservesOrder(t *testing.T) {
	s := NewSessions()
	s.Set("zebra", Session{Prompt: "1"})
	s.Set("monkey", Session{Prompt: "2"})
	s.Set("aardvark", Session{Prompt: "3"})

	gotBytes, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	gotStr := string(gotBytes)

	// Since standard Go maps randomize order, a regular map stringification
	// would fluctuate. We assert our explicit sequence here.
	idxZebra := strings.Index(gotStr, "zebra")
	idxMonkey := strings.Index(gotStr, "monkey")
	idxAardvark := strings.Index(gotStr, "aardvark")

	if idxZebra == -1 || idxMonkey == -1 || idxAardvark == -1 {
		t.Fatalf("Missing expected keys in JSON output: %s", gotStr)
	}

	if !(idxZebra < idxMonkey && idxMonkey < idxAardvark) {
		t.Errorf("JSON keys are out of order. Got payload: %s", gotStr)
	}
}

func TestSessions_JSON_Unmarshal_PreservesOrder(t *testing.T) {
	inputJSON := `{
		"omega": {"prompt": "99"},
		"alpha": {"prompt": "01"},
		"beta":  {"prompt": "02"}
	}`

	var s Sessions
	if err := json.Unmarshal([]byte(inputJSON), &s); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify internal tracking slice matches document layout order
	expectedOrder := []string{"omega", "alpha", "beta"}
	if !reflect.DeepEqual(s.Order, expectedOrder) {
		t.Errorf("Unmarshaled JSON order tracking mismatch.\nGot : %v\nWant: %v", s.Order, expectedOrder)
	}

	// Verify data map retrieval remains intact
	if s.Get("alpha").Prompt != "01" {
		t.Errorf("Expected key 'alpha' to have ID '01', got '%s'", s.Get("alpha").Prompt)
	}
}

// --- YAML TESTS ---

func TestSessions_YAML_Marshal_PreservesOrder(t *testing.T) {
	s := NewSessions()
	s.Set("charlie", Session{Prompt: "C"})
	s.Set("alpha", Session{Prompt: "A"})
	s.Set("bravo", Session{Prompt: "B"})

	gotBytes, err := yaml.Marshal(s)
	if err != nil {
		t.Fatalf("Failed to marshal YAML: %v", err)
	}

	gotStr := string(gotBytes)

	idxCharlie := strings.Index(gotStr, "charlie")
	idxAlpha := strings.Index(gotStr, "alpha")
	idxBravo := strings.Index(gotStr, "bravo")

	if idxCharlie == -1 || idxAlpha == -1 || idxBravo == -1 {
		t.Fatalf("Missing expected keys in YAML output: %s", gotStr)
	}

	if !(idxCharlie < idxAlpha && idxAlpha < idxBravo) {
		t.Errorf("YAML keys are out of order. Got payload:\n%s", gotStr)
	}
}

func TestSessions_YAML_Unmarshal_PreservesOrder(t *testing.T) {
	inputYAML := `
delta:
  prompt: "4"
echo:
  prompt: "5"
foxtrot:
  prompt: "6"
`

	var s Sessions
	if err := yaml.Unmarshal([]byte(inputYAML), &s); err != nil {
		t.Fatalf("Failed to unmarshal YAML: %v", err)
	}

	expectedOrder := []string{"delta", "echo", "foxtrot"}
	if !reflect.DeepEqual(s.Order, expectedOrder) {
		t.Errorf("Unmarshaled YAML order tracking mismatch.\nGot : %v\nWant: %v", s.Order, expectedOrder)
	}

	if s.Get("echo").Prompt != "5" {
		t.Errorf("Expected key 'echo' to have ID '5', got '%s'", s.Get("echo").Prompt)
	}
}

// --- ITERATOR & DEFENSIVE TESTS ---

func TestSessions_IterationOrder(t *testing.T) {
	s := NewSessions()
	s.Set("1st", Session{Prompt: "first"})
	s.Set("2nd", Session{Prompt: "second"})
	s.Set("3rd", Session{Prompt: "third"})

	var iteratedKeys []string
	for k := range s.Iter() {
		iteratedKeys = append(iteratedKeys, k)
	}

	expected := []string{"1st", "2nd", "3rd"}
	if !reflect.DeepEqual(iteratedKeys, expected) {
		t.Errorf("Iterator sequence broken.\nGot : %v\nWant: %v", iteratedKeys, expected)
	}
}
