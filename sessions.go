package frags

import (
	"bytes"
	"encoding/json"
	"fmt"
	"iter"

	"gopkg.in/yaml.v3" // Assumed version based on your UnmarshalYAML signature
)

// Sessions is an ordered map of session IDs to sessions.
// Note: Struct tags on Data and Order are ignored now because
// the custom Marshal/Unmarshal methods control the entire serialization.
type Sessions struct {
	Data  map[string]Session `validate:"required,min=1"`
	Order []string
}

func NewSessions() Sessions {
	return Sessions{
		Data:  make(map[string]Session),
		Order: make([]string, 0),
	}
}

func (s *Sessions) Set(key string, value Session) {
	if s.Data == nil {
		s.Data = make(map[string]Session)
	}

	if _, exists := s.Data[key]; !exists {
		s.Order = append(s.Order, key)
	}

	s.Data[key] = value
}

func (s Sessions) Iter() iter.Seq2[string, Session] {
	return func(yield func(string, Session) bool) {
		for _, k := range s.Order {
			v, ok := s.Data[k]
			if !ok {
				continue
			}
			if !yield(k, v) {
				return
			}
		}
	}
}

func (s Sessions) Get(key string) Session {
	return s.Data[key]
}

// --- JSON Handling ---

// MarshalJSON serializes the map as a flat JSON object, keeping s.Order sequence.
func (s Sessions) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')

	for i, k := range s.Order {
		v, ok := s.Data[k]
		if !ok {
			continue
		}
		if i > 0 {
			buf.WriteByte(',')
		}

		keyBytes, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		buf.Write(keyBytes)
		buf.WriteByte(':')

		valBytes, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		buf.Write(valBytes)
	}

	buf.WriteByte('}')
	return buf.Bytes(), nil
}

// UnmarshalJSON uses a token decoder to capture the exact order keys appear in the JSON object.
func (s *Sessions) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewReader(data))

	t, err := dec.Token()
	if err != nil {
		return err
	}
	if t != json.Delim('{') {
		return fmt.Errorf("expected JSON object")
	}

	s.Data = make(map[string]Session)
	s.Order = make([]string, 0)

	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			return err
		}
		key, ok := t.(string)
		if !ok {
			return fmt.Errorf("expected string key")
		}

		var val Session
		if err := dec.Decode(&val); err != nil {
			return err
		}

		s.Data[key] = val
		s.Order = append(s.Order, key)
	}

	// Read closing object delimiter
	_, err = dec.Token()
	return err
}

// --- YAML Handling ---
// MarshalYAML converts the ordered map into a yaml.Node mapping sequence.
func (s Sessions) MarshalYAML() (any, error) {
	node := yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
	}

	for _, k := range s.Order {
		v, ok := s.Data[k]
		if !ok {
			continue
		}

		// Encode the key scalar
		var keyNode yaml.Node
		if err := keyNode.Encode(k); err != nil {
			return nil, err
		}

		// Encode the value struct/scalar
		var valNode yaml.Node
		if err := valNode.Encode(v); err != nil {
			return nil, err
		}

		// In a MappingNode, Content pairs up elements: [key1, value1, key2, value2...]
		node.Content = append(node.Content, &keyNode, &valNode)
	}

	return &node, nil
}

// UnmarshalYAML processes the raw AST node sequence to preserve structural order.
func (s *Sessions) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("line %d: expected a YAML map object", value.Line)
	}

	// Content slice length will be 2x the number of map entries (keys + values)
	s.Data = make(map[string]Session)
	s.Order = make([]string, 0, len(value.Content)/2)

	for i := 0; i < len(value.Content); i += 2 {
		keyNode := value.Content[i]
		valNode := value.Content[i+1]

		var key string
		if err := keyNode.Decode(&key); err != nil {
			return err
		}

		var val Session
		if err := valNode.Decode(&val); err != nil {
			return err
		}

		s.Data[key] = val
		s.Order = append(s.Order, key)
	}

	return nil
}
