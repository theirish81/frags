package schema

import (
	"encoding/json"
	"testing"
)

func pretty(s *Schema) string {
	b, _ := json.MarshalIndent(s, "", "  ")
	return string(b)
}

func TestScalars(t *testing.T) {
	cases := []struct {
		in       any
		wantType string
	}{
		{"hello", String},
		{true, Boolean},
		{42, Integer},
		{3.14, Number},
		{3.0, Integer}, // whole float → integer
	}
	for _, c := range cases {
		s := GuessSchema(c.in)
		if s.Type != Type(c.wantType) {
			t.Errorf("GuessSchema(%v): got type %q, want %q", c.in, s.Type, c.wantType)
		}
	}
}

func TestNil(t *testing.T) {
	s := GuessSchema(nil)
	if s.Nullable == nil || !*s.Nullable {
		t.Errorf("expected nullable schema, got %s", pretty(s))
	}
}

func TestSimpleMap(t *testing.T) {
	input := map[string]any{
		"name": "Alice",
		"age":  30,
	}
	s := GuessSchema(input)
	if s.Type != Object {
		t.Fatalf("expected object, got %q", s.Type)
	}
	if s.Properties["name"].Type != String {
		t.Errorf("name should be string")
	}
	if s.Properties["age"].Type != Integer {
		t.Errorf("age should be integer")
	}
}

func TestSliceOfObjects_RequiredVsOptional(t *testing.T) {
	// "id" present in all → required; "email" only in some → not required
	input := []any{
		map[string]any{"id": 1, "name": "Alice", "email": "alice@example.com"},
		map[string]any{"id": 2, "name": "Bob"},
		map[string]any{"id": 3, "name": "Carol", "email": "carol@example.com"},
	}
	s := GuessSchema(input)
	if s.Type != Array {
		t.Fatalf("expected array, got %q", s.Type)
	}
	items := s.Items
	if items == nil {
		t.Fatal("items is nil")
	}
	if items.Type != Object {
		t.Fatalf("items type: got %q, want object", items.Type)
	}

	required := make(map[string]bool)
	for _, r := range items.Required {
		required[r] = true
	}

	if !required["id"] {
		t.Error("id should be required (present in all objects)")
	}
	if !required["name"] {
		t.Error("name should be required")
	}
	if required["email"] {
		t.Error("email should NOT be required (missing in some objects)")
	}

	t.Logf("items schema:\n%s", pretty(items))
}

func TestTypeVariation_OneOf(t *testing.T) {
	// "value" field is sometimes string, sometimes integer → oneOf
	input := []any{
		map[string]any{"value": "hello"},
		map[string]any{"value": 42},
	}
	s := GuessSchema(input)
	valueSchema := s.Items.Properties["value"]
	if valueSchema == nil {
		t.Fatal("value property is nil")
	}
	if len(valueSchema.OneOf) < 2 {
		t.Errorf("expected oneOf with 2 branches, got: %s", pretty(valueSchema))
	}
	t.Logf("value schema:\n%s", pretty(valueSchema))
}

func TestNullableField(t *testing.T) {
	// "tag" is a string in one object, null in another → nullable string
	input := []any{
		map[string]any{"tag": "go"},
		map[string]any{"tag": nil},
	}
	s := GuessSchema(input)
	tagSchema := s.Items.Properties["tag"]
	if tagSchema == nil {
		t.Fatal("tag property is nil")
	}
	if tagSchema.Type != String || tagSchema.Nullable == nil || !*tagSchema.Nullable {
		t.Errorf("expected nullable string, got: %s", pretty(tagSchema))
	}
}

func TestNestedObjects(t *testing.T) {
	input := map[string]any{
		"user": map[string]any{
			"id":   1,
			"name": "Alice",
			"address": map[string]any{
				"city": "Rome",
				"zip":  "00100",
			},
		},
	}
	s := GuessSchema(input)
	user := s.Properties["user"]
	if user == nil || user.Type != Object {
		t.Fatal("user should be an object")
	}
	addr := user.Properties["address"]
	if addr == nil || addr.Type != Object {
		t.Fatal("address should be an object")
	}
	if addr.Properties["city"].Type != String {
		t.Error("city should be string")
	}
}

func TestEmptySlice(t *testing.T) {
	s := GuessSchema([]any{})
	if s.Type != Array {
		t.Errorf("expected array, got %q", s.Type)
	}
	if s.Items != nil {
		t.Errorf("expected nil items for empty slice")
	}
}

func TestMixedSlice_OneOf(t *testing.T) {
	// top-level slice with mixed element types
	input := []any{"hello", 42, true}
	s := GuessSchema(input)
	if s.Items == nil {
		t.Fatal("items is nil")
	}
	if len(s.Items.OneOf) < 3 {
		t.Errorf("expected oneOf with 3 branches, got: %s", pretty(s.Items))
	}
}
