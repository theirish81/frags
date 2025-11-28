package frags

import "encoding/json"

// safeUnmarshalDataStructure is a thread-safe version of json.Unmarshal, and is used to unmarshal the runner's
// data structure.
func (r *Runner[T]) safeUnmarshalDataStructure(data []byte) error {
	r.marshalingMutex.Lock()
	defer r.marshalingMutex.Unlock()
	err := json.Unmarshal(data, r.dataStructure)
	return err
}

// safeMarshalDataStructure is a thread-safe version of json.Marshal, and is used to marshal the runner's
// data structure.
func (r *Runner[T]) safeMarshalDataStructure(indent bool) ([]byte, error) {
	r.marshalingMutex.Lock()
	defer r.marshalingMutex.Unlock()
	if indent {
		return json.MarshalIndent(r.dataStructure, "", "  ")
	}
	return json.Marshal(r.dataStructure)
}
